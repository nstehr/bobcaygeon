package raft

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/hashicorp/memberlist"
	speakerAPI "github.com/nstehr/bobcaygeon/api"
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
		client, err := dms.getLeaderClient(dms.store.GetLeader())
		if err != nil {
			return err
		}
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
	// first update the actual name broadcast by speaker
	speakerClient, err := dms.getSpeakerClient(ID)
	if err != nil {
		return err
	}
	resp, err := speakerClient.ChangeServiceName(context.Background(), &speakerAPI.NameChangeRequest{NewName: displayName})
	if err != nil {
		return err
	}
	if resp.ReturnCode != 200 {
		return fmt.Errorf("Error changing name of speaker")
	}
	// now we can save the display name in the store
	speakerConfig.DisplayName = displayName
	return dms.store.SaveSpeakerConfig(speakerConfig)
}

// CreateZone will create a new zone with 0 to more speakers
func (dms *DistributedMgmtService) CreateZone(displayName string, speakerIDs []string) (string, error) {
	if !dms.store.AmLeader() {
		client, err := dms.getLeaderClient(dms.store.GetLeader())
		if err != nil {
			return "", err
		}
		resp, err := client.CreateZone(context.Background(), &api.ZoneRequest{DisplayName: displayName, SpeakerIds: speakerIDs})
		if err != nil {
			return "", err
		}
		if resp.ResponseCode != 200 {
			return "", fmt.Errorf(resp.Message)
		}
		return resp.Id, nil
	}

	// create a random id for the zone
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1).Int63()
	id := fmt.Sprintf("%d", r1)
	zc := ZoneConfig{ID: id, DisplayName: displayName, Speakers: speakerIDs}
	if len(speakerIDs) > 0 {
		zc.Leader = speakerIDs[0]
	}
	for _, speakerID := range speakerIDs {
		client, err := dms.getSpeakerClient(speakerID)
		if err != nil {
			return "", err
		}
		// first, clear any sessions the speakers may have
		log.Printf("Clearing sessions from: %s \n", speakerID)
		_, err = client.RemoveForwardToNodes(context.Background(), &speakerAPI.AddRemoveNodesRequest{RemoveAll: true})
		if err != nil {
			return "", err
		}
		// next, make sure the non-leaders aren't broadcasting
		if speakerID != zc.Leader {
			log.Printf("Setting broadcast to false for: %s \n", speakerID)
			_, err = client.ToggleBroadcast(context.Background(), &speakerAPI.BroadcastRequest{ShouldBroadcast: false})
			if err != nil {
				return "", err
			}
		} else {
			// make sure the leader is broadcasting, and has the cluster name
			log.Printf("Changing display name for: %s to %s\n", speakerID, displayName)
			_, err = client.ChangeServiceName(context.Background(), &speakerAPI.NameChangeRequest{NewName: displayName})
			if err != nil {
				return "", err
			}
			log.Printf("Setting broadcast to true for: %s \n", speakerID)
			_, err = client.ToggleBroadcast(context.Background(), &speakerAPI.BroadcastRequest{ShouldBroadcast: true})
			if err != nil {
				return "", err
			}
		}
	}
	// finally, add the ids to be forwarded to
	if len(speakerIDs) > 0 {
		client, err := dms.getSpeakerClient(zc.Leader)
		if err != nil {
			return "", err
		}
		forwardToIDs := make([]string, 0)
		for _, v := range speakerIDs {
			if v != zc.Leader {
				forwardToIDs = append(forwardToIDs, v)
			}
		}
		log.Printf("Telling %s to forward to nodes \n", zc.Leader)
		_, err = client.ForwardToNodes(context.Background(), &speakerAPI.AddRemoveNodesRequest{Ids: forwardToIDs})
		if err != nil {
			return "", err
		}
	}
	dms.store.SaveZoneConfig(zc)
	return id, nil
}

// AddSpeakersToZone updates a zone with the given speakers
func (dms *DistributedMgmtService) AddSpeakersToZone(zoneID string, speakerIDs []string) error {
	if !dms.store.AmLeader() {
		client, err := dms.getLeaderClient(dms.store.GetLeader())
		if err != nil {
			return err
		}
		resp, err := client.AddSpeakersToZone(context.Background(), &api.ZoneRequest{ZoneId: zoneID, SpeakerIds: speakerIDs})
		if err != nil {
			return err
		}
		if resp.ResponseCode != 200 {
			return fmt.Errorf(resp.Message)
		}
		return nil
	}
	if len(speakerIDs) == 0 {
		return nil
	}
	zc := dms.store.GetZoneConfigs()
	var zone ZoneConfig
	for _, zoneConfig := range zc {
		if zoneConfig.ID == zoneID {
			zone = zoneConfig
			break
		}
	}
	if zone.ID == "" {
		return fmt.Errorf("Zone: %s not found", zoneID)
	}
	for _, speakerID := range speakerIDs {
		if speakerID == zone.Leader {
			continue
		}
		client, err := dms.getSpeakerClient(speakerID)
		if err != nil {
			return err
		}
		// first, clear any sessions the speakers may have
		log.Printf("Clearing sessions from: %s \n", speakerID)
		_, err = client.RemoveForwardToNodes(context.Background(), &speakerAPI.AddRemoveNodesRequest{RemoveAll: true})
		if err != nil {
			return err
		}
		// next, make sure the non-leaders aren't broadcasting
		log.Printf("Setting broadcast to false for: %s \n", speakerID)
		_, err = client.ToggleBroadcast(context.Background(), &speakerAPI.BroadcastRequest{ShouldBroadcast: false})
		if err != nil {
			return err
		}
	}

	client, err := dms.getSpeakerClient(zone.Leader)
	if err != nil {
		return err
	}
	log.Printf("Telling %s to forward to nodes \n", zone.Leader)
	_, err = client.ForwardToNodes(context.Background(), &speakerAPI.AddRemoveNodesRequest{Ids: speakerIDs})
	if err != nil {
		return err
	}
	zone.Speakers = append(zone.Speakers, speakerIDs...)
	dms.store.SaveZoneConfig(zone)
	return nil
}

// RemoveSpeakersFromZone updates a zone with the given speakers
func (dms *DistributedMgmtService) RemoveSpeakersFromZone(zoneID string, speakerIDs []string) error {
	if !dms.store.AmLeader() {
		client, err := dms.getLeaderClient(dms.store.GetLeader())
		if err != nil {
			return err
		}
		resp, err := client.RemoveSpeakersFromZone(context.Background(), &api.ZoneRequest{ZoneId: zoneID, SpeakerIds: speakerIDs})
		if err != nil {
			return err
		}
		if resp.ResponseCode != 200 {
			return fmt.Errorf(resp.Message)
		}
		return nil
	}
	if len(speakerIDs) == 0 {
		return nil
	}
	zc := dms.store.GetZoneConfigs()
	var zone ZoneConfig
	for _, zoneConfig := range zc {
		if zoneConfig.ID == zoneID {
			zone = zoneConfig
			break
		}
	}
	if zone.ID == "" {
		return fmt.Errorf("Zone: %s not found", zoneID)
	}
	for _, speakerID := range speakerIDs {
		if speakerID == zone.Leader {
			continue
		}
		client, err := dms.getSpeakerClient(speakerID)
		if err != nil {
			return err
		}

		// re-enable removed speakers broadcasting
		_, err = client.ToggleBroadcast(context.Background(), &speakerAPI.BroadcastRequest{ShouldBroadcast: true})
		if err != nil {
			return err
		}
	}

	client, err := dms.getSpeakerClient(zone.Leader)
	if err != nil {
		return err
	}

	_, err = client.RemoveForwardToNodes(context.Background(), &speakerAPI.AddRemoveNodesRequest{Ids: speakerIDs})
	if err != nil {
		return err
	}
	newSpeakers := make([]string, 0)
	for _, speaker := range zone.Speakers {
		found := false
		for _, speakerToRemove := range speakerIDs {
			if speakerToRemove == speaker {
				found = true
			}
			if !found {
				newSpeakers = append(newSpeakers, speaker)
			}
		}
	}
	zone.Speakers = newSpeakers
	dms.store.SaveZoneConfig(zone)
	return nil
}

// DeleteZone will delete a zone
func (dms *DistributedMgmtService) DeleteZone(zoneID string) error {
	if !dms.store.AmLeader() {
		client, err := dms.getLeaderClient(dms.store.GetLeader())
		if err != nil {
			return err
		}
		resp, err := client.DeleteZone(context.Background(), &api.ZoneRequest{ZoneId: zoneID})
		if err != nil {
			return err
		}
		if resp.ResponseCode != 200 {
			return fmt.Errorf(resp.Message)
		}
		return nil
	}

	zc := dms.store.GetZoneConfigs()
	var zone ZoneConfig
	for _, zoneConfig := range zc {
		if zoneConfig.ID == zoneID {
			zone = zoneConfig
			break
		}
	}
	if zone.ID == "" {
		return fmt.Errorf("Zone: %s not found", zoneID)
	}
	for _, speakerID := range zone.Speakers {
		if speakerID == zone.Leader {
			continue
		}
		client, err := dms.getSpeakerClient(speakerID)
		if err != nil {
			return err
		}

		// re-enable broadcasting
		log.Printf("Setting broadcast to true for: %s \n", speakerID)
		_, err = client.ToggleBroadcast(context.Background(), &speakerAPI.BroadcastRequest{ShouldBroadcast: true})
		if err != nil {
			return err
		}
	}

	client, err := dms.getSpeakerClient(zone.Leader)
	if err != nil {
		return err
	}
	log.Printf("Clearing sessions from: %s \n", zone.Leader)
	_, err = client.RemoveForwardToNodes(context.Background(), &speakerAPI.AddRemoveNodesRequest{RemoveAll: true})
	if err != nil {
		return err
	}
	// change the name back from the zone name
	speakerConfig, err := dms.store.GetSpeakerConfig(zone.Leader)
	if err != nil {
		return err
	}
	log.Printf("Changing service name of: %s from: %s to: %s", zone.Leader, zone.DisplayName, speakerConfig.DisplayName)
	_, err = client.ChangeServiceName(context.Background(), &speakerAPI.NameChangeRequest{NewName: speakerConfig.DisplayName})
	if err != nil {
		return err
	}
	dms.store.DeleteZoneConfig(zone.ID)
	return nil
}

// ChangeZoneName changes the name of the zone
func (dms *DistributedMgmtService) ChangeZoneName(zoneID string, newName string) error {
	if !dms.store.AmLeader() {
		client, err := dms.getLeaderClient(dms.store.GetLeader())
		if err != nil {
			return err
		}
		resp, err := client.ChangeZoneName(context.Background(), &api.ZoneRequest{ZoneId: zoneID, DisplayName: newName})
		if err != nil {
			return err
		}
		if resp.ResponseCode != 200 {
			return fmt.Errorf(resp.Message)
		}
		return nil
	}

	zc := dms.store.GetZoneConfigs()
	var zone ZoneConfig
	for _, zoneConfig := range zc {
		if zoneConfig.ID == zoneID {
			zone = zoneConfig
			break
		}
	}
	if zone.ID == "" {
		return fmt.Errorf("Zone: %s not found", zoneID)
	}

	client, err := dms.getSpeakerClient(zone.Leader)
	if err != nil {
		return err
	}
	log.Printf("Changing service name of: %s from: %s to: %s", zone.Leader, zone.DisplayName, newName)
	_, err = client.ChangeServiceName(context.Background(), &speakerAPI.NameChangeRequest{NewName: newName})
	if err != nil {
		return err
	}
	zone.DisplayName = newName
	dms.store.SaveZoneConfig(zone)
	return nil
}

// GetZones returns information about the zones under our management
func (dms *DistributedMgmtService) GetZones() []*service.Zone {
	var zones []*service.Zone
	zc := dms.store.GetZoneConfigs()
	speakers := dms.GetSpeakers()
	for _, config := range zc {
		var zoneSpeakers []*service.Speaker
		for _, speakerID := range config.Speakers {
			for _, speaker := range speakers {
				if speaker.ID == speakerID {
					zoneSpeakers = append(zoneSpeakers, speaker)
				}
			}
		}
		z := &service.Zone{ID: config.ID, DisplayName: config.DisplayName, Speakers: zoneSpeakers}
		zones = append(zones, z)
	}
	return zones
}

func (dms *DistributedMgmtService) getLeaderAPIAddress(leader *net.TCPAddr) string {
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

func (dms *DistributedMgmtService) getLeaderClient(leader string) (api.BobcaygeonManagementClient, error) {
	leaderAddr, _ := net.ResolveTCPAddr("tcp", leader)
	apiAddress := dms.getLeaderAPIAddress(leaderAddr)
	if apiAddress == "" {
		return nil, fmt.Errorf("Could not resolve API address for: %s", leader)
	}
	log.Printf("Forwarding request to leader: %s \n", apiAddress)
	conn, err := grpc.Dial(apiAddress, grpc.WithInsecure())
	if err != nil {
		log.Println("Could not open connection", err)
		return nil, err
	}
	client := api.NewBobcaygeonManagementClient(conn)
	return client, nil
}

func (dms *DistributedMgmtService) getSpeakerClient(speakerID string) (speakerAPI.AirPlayManagementClient, error) {

	filter := func(node *memberlist.Node) bool {
		meta := cluster.DecodeNodeMeta(node.Meta)
		return meta.NodeType == cluster.Music && speakerID == node.Name
	}

	speakers := cluster.FilterMembersByFn(filter, dms.nodes)
	if len(speakers) != 1 {
		return nil, fmt.Errorf("Could not find speaker with id: %s", speakerID)
	}
	speaker := speakers[0]
	meta := cluster.DecodeNodeMeta(speaker.Meta)
	speakerAPIAddress := fmt.Sprintf("%s:%d", speaker.Addr.String(), meta.APIPort)
	conn, err := grpc.Dial(speakerAPIAddress, grpc.WithInsecure())
	if err != nil {
		log.Println("Could not open connection", err)
		return nil, err
	}
	client := speakerAPI.NewAirPlayManagementClient(conn)
	return client, nil
}
