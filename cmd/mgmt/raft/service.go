package raft

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/hashicorp/memberlist"
	"github.com/nstehr/bobcaygeon/cluster"
	"github.com/nstehr/bobcaygeon/cmd/mgmt/api"
	"github.com/nstehr/bobcaygeon/cmd/mgmt/service"
	"google.golang.org/grpc"
)

// DistributedMgmtService implements MgmtService with a distributed backing store
type DistributedMgmtService struct {
	nodes *memberlist.Memberlist
	store *DistributedStore
}

// NewDistributedMgmtService instantiates the DistributedMgmtService
func NewDistributedMgmtService(nodes *memberlist.Memberlist, store *DistributedStore) *DistributedMgmtService {
	return &DistributedMgmtService{nodes: nodes, store: store}
}

// GetSpeakers returns information about the speaker (bcg apps) under our management
func (dms *DistributedMgmtService) GetSpeakers() []*service.Speaker {
	var speakers []*service.Speaker
	for _, member := range cluster.FilterMembers(cluster.Music, dms.nodes) {
		displayName := member.Name
		speakerConfig, err := dms.store.GetSpeakerConfig(member.Name)
		if err != nil {
			log.Printf("Error retrieving config for: %s. Error: %s\n", member.Name, err)
		}
		if speakerConfig.DisplayName != "" {
			displayName = speakerConfig.DisplayName
		}
		speaker := &service.Speaker{ID: member.Name, DisplayName: displayName}
		speakers = append(speakers, speaker)
	}

	return speakers
}

// SetDisplayName will change the user visible name of the speaker
func (dms *DistributedMgmtService) SetDisplayName(ID string, displayName string) error {
	if !dms.store.AmLeader() {
		leader, _ := net.ResolveTCPAddr("tcp", dms.store.GetLeader())
		apiAddress := dms.getAPIAddress(leader)
		if apiAddress == "" {
			return fmt.Errorf("Could not resolve API address for: %s", dms.store.GetLeader())
		}
		log.Printf("Forwarding request to leader: %s \n", apiAddress)
		conn, err := grpc.Dial(apiAddress, grpc.WithInsecure())
		if err != nil {
			log.Println("Could not open connection", err)
			return err
		}
		client := api.NewBobcaygeonManagementClient(conn)
		resp, err := client.SetDisplayNameForSpeaker(context.Background(), &api.SetSpeakerDisplayNameRequest{SpeakerId: ID, DisplayName: displayName})
		if err != nil {
			return err
		}
		if resp.ResponseCode != 200 {
			return fmt.Errorf(resp.Message)
		}
		return nil
	}
	speakerConfig, err := dms.store.GetSpeakerConfig(ID)
	if err != nil {
		log.Printf("Error retrieving config for: %s. Error: %s\n", ID, err)
		return err
	}
	if speakerConfig.ID == "" {
		speakerConfig.ID = ID
	}
	speakerConfig.DisplayName = displayName
	return dms.store.SaveSpeakerConfig(speakerConfig)
}

func (dms *DistributedMgmtService) getAPIAddress(leader *net.TCPAddr) string {
	for _, member := range cluster.FilterMembers(cluster.Mgmt, dms.nodes) {
		memberIP := member.Addr.String()
		meta := cluster.DecodeNodeMeta(member.Meta)
		if (leader.IP == nil && isLocalIP(memberIP) || leader.IP.String() == memberIP) && leader.Port == meta.RaftPort {
			memberAPIAddress := fmt.Sprintf("%s:%d", memberIP, meta.APIPort)
			return memberAPIAddress
		}
	}
	return ""
}

func isLocalIP(ipAddr string) bool {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4().String() != ipAddr {
				return true
			}
		}
	}
	return false
}
