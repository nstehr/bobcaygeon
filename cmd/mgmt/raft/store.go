package raft

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

const (
	retainSnapshotCount = 2
	raftTimeout         = 10 * time.Second
)

// SpeakerConfig used to store persistent speaker configuration
type SpeakerConfig struct {
	ID          string
	DisplayName string
}

// DistributedStore is a raft backed store based on: https://github.com/otoolep/hraftd/blob/master/store/store.go
type DistributedStore struct {
	raftPort int
	raftDir  string
	localID  string
	mu       sync.Mutex
	m        map[string]SpeakerConfig
	raft     *raft.Raft
}

type command struct {
	Op    string        `json:"op,omitempty"`
	Key   string        `json:"key,omitempty"`
	Value SpeakerConfig `json:"value,omitempty"`
}

// GetSpeakerConfig returns the speaker configuration for the given speaker
func (ds *DistributedStore) GetSpeakerConfig(ID string) (SpeakerConfig, error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return ds.m[ID], nil
}

// SaveSpeakerConfig saves the specified SpeakerConfig
func (ds *DistributedStore) SaveSpeakerConfig(config SpeakerConfig) error {
	if ds.raft.State() != raft.Leader {
		return fmt.Errorf("not leader")
	}

	c := &command{
		Op:    "set",
		Key:   config.ID,
		Value: config,
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	f := ds.raft.Apply(b, raftTimeout)
	return f.Error()
}

// NewDistributedStore initializes the store
func NewDistributedStore(localID string, raftPort int, raftDir string) *DistributedStore {
	return &DistributedStore{localID: localID,
		raftPort: raftPort,
		raftDir:  raftDir,
		m:        make(map[string]SpeakerConfig)}
}

// Open will open the database for usage
func (ds *DistributedStore) Open(bootstrap bool) error {
	r, err := initRaft(ds.localID, ds.raftPort, ds.raftDir, ds, bootstrap)
	if err != nil {
		return err
	}
	ds.raft = r
	return nil
}

// GetLeader returns the address of the leader
func (ds *DistributedStore) GetLeader() string {
	return string(ds.raft.Leader())
}

// AmLeader whether or not the store instance is the leader of the cluster
func (ds *DistributedStore) AmLeader() bool {
	return ds.raft.State() == raft.Leader
}

// Join will join the specified node to participate in the raft cluster
func (ds *DistributedStore) Join(nodeID string, nodeAddress string) error {
	log.Printf("received join request for remote node %s at %s\n", nodeID, nodeAddress)

	if ds.raft.State() != raft.Leader {
		log.Printf("I am not leader, ignoring join request")
		return nil
	}

	configFuture := ds.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		log.Printf("failed to get raft configuration: %v\n", err)
		return err
	}

	for _, srv := range configFuture.Configuration().Servers {
		// If a node already exists with either the joining node's ID or address,
		// that node may need to be removed from the config first.
		if srv.ID == raft.ServerID(nodeID) || srv.Address == raft.ServerAddress(nodeAddress) {
			// However if *both* the ID and the address are the same, then nothing -- not even
			// a join operation -- is needed.
			if srv.Address == raft.ServerAddress(nodeAddress) && srv.ID == raft.ServerID(nodeID) {
				log.Printf("node %s at %s already member of cluster, ignoring join request\n", nodeID, nodeAddress)
				return nil
			}

			future := ds.raft.RemoveServer(srv.ID, 0, 0)
			if err := future.Error(); err != nil {
				return fmt.Errorf("error removing existing node %s at %s: %s", nodeID, nodeAddress, err)
			}
		}
	}

	f := ds.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(nodeAddress), 0, 0)
	if f.Error() != nil {
		return f.Error()
	}
	log.Printf("node %s at %s joined successfully\n", nodeID, nodeAddress)
	return nil
}

// Apply applies a Raft log entry to the key-value store.
func (ds *DistributedStore) Apply(l *raft.Log) interface{} {
	var c command
	if err := json.Unmarshal(l.Data, &c); err != nil {
		log.Printf("failed to unmarshal command: %s\n", err.Error())
	}

	switch c.Op {
	case "set":
		return ds.applySet(c.Key, c.Value)
	default:
		log.Printf("unrecognized command op: %s\n", c.Op)
		return nil
	}
}

func (ds *DistributedStore) applySet(key string, value SpeakerConfig) interface{} {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.m[key] = value
	return nil
}

// Snapshot returns a snapshot of the key-value store.
func (ds *DistributedStore) Snapshot() (raft.FSMSnapshot, error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Clone the map.
	o := make(map[string]SpeakerConfig)
	for k, v := range ds.m {
		o[k] = v
	}
	return &fsmSnapshot{store: o}, nil
}

// Restore stores the key-value store to a previous state.
func (ds *DistributedStore) Restore(rc io.ReadCloser) error {
	o := make(map[string]SpeakerConfig)
	if err := json.NewDecoder(rc).Decode(&o); err != nil {
		return err
	}

	// Set the state from the snapshot, no lock required according to
	// Hashicorp docs.
	ds.m = o
	return nil
}

func initRaft(localID string, raftPort int, raftDir string, s *DistributedStore, bootstrap bool) (*raft.Raft, error) {
	// Setup Raft configuration.
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(localID)
	raftAddr := fmt.Sprintf(":%d", raftPort)
	// Setup Raft communication.
	addr, err := net.ResolveTCPAddr("tcp", raftAddr)
	if err != nil {
		return nil, err
	}
	transport, err := raft.NewTCPTransport(raftAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, err
	}

	// Create the snapshot store. This allows the Raft to truncate the log.
	snapshots, err := raft.NewFileSnapshotStore(raftDir, retainSnapshotCount, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("file snapshot store: %s", err)
	}

	// Create the log store and stable store.
	var logStore raft.LogStore
	var stableStore raft.StableStore

	boltDB, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft.db"))
	if err != nil {
		return nil, fmt.Errorf("new bolt store: %s", err)
	}
	logStore = boltDB
	stableStore = boltDB

	// Instantiate the Raft systems.
	ra, err := raft.NewRaft(config, s, logStore, stableStore, snapshots, transport)
	if err != nil {
		return nil, fmt.Errorf("new raft: %s", err)
	}

	if bootstrap {
		log.Println("bootstrapping raft cluster")
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		ra.BootstrapCluster(configuration)
	}

	return ra, nil
}

type fsmSnapshot struct {
	store map[string]SpeakerConfig
}

func (f *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		// Encode data.
		b, err := json.Marshal(f.store)
		if err != nil {
			return err
		}

		// Write data to sink.
		if _, err := sink.Write(b); err != nil {
			return err
		}

		// Close the sink.
		return sink.Close()
	}()

	if err != nil {
		sink.Cancel()
	}

	return err
}

func (f *fsmSnapshot) Release() {}
