package store

import (
	"encoding/json"
	"fmt"
	"io"
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

// Store is the interface for saving information
type Store interface {
	SaveSpeakerConfig(config SpeakerConfig) error
	GetSpeakerConfig(ID string) (SpeakerConfig, error)
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
func (ds *DistributedStore) Open() error {
	r, err := initRaft(ds.localID, ds.raftPort, ds.raftDir, ds)
	if err != nil {
		return err
	}
	ds.raft = r
	return nil
}

// Apply applies a Raft log entry to the key-value store.
func (ds *DistributedStore) Apply(l *raft.Log) interface{} {
	var c command
	if err := json.Unmarshal(l.Data, &c); err != nil {
		panic(fmt.Sprintf("failed to unmarshal command: %s", err.Error()))
	}

	switch c.Op {
	case "set":
		return ds.applySet(c.Key, c.Value)
	default:
		panic(fmt.Sprintf("unrecognized command op: %s", c.Op))
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

func initRaft(localID string, raftPort int, raftDir string, s *DistributedStore) (*raft.Raft, error) {
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

	configuration := raft.Configuration{
		Servers: []raft.Server{
			{
				ID:      config.LocalID,
				Address: transport.LocalAddr(),
			},
		},
	}
	ra.BootstrapCluster(configuration)

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
