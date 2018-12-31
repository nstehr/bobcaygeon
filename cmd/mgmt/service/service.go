package service

import (
	"log"

	"github.com/hashicorp/memberlist"
	"github.com/nstehr/bobcaygeon/cluster"
	"github.com/nstehr/bobcaygeon/cmd/mgmt/store"
)

// MgmtService interface for handling management capabilities
type MgmtService interface {
	GetSpeakers() []*Speaker
	SetDisplayName(ID string, displayName string) error
}

// Speaker speaker instance
type Speaker struct {
	ID          string
	DisplayName string
}

// DistributedMgmtService implements MgmtService with a distributed backing store
type DistributedMgmtService struct {
	nodes *memberlist.Memberlist
	store store.Store
}

// NewDistributedMgmtService instantiates the DistributedMgmtService
func NewDistributedMgmtService(nodes *memberlist.Memberlist, store store.Store) *DistributedMgmtService {
	return &DistributedMgmtService{nodes: nodes, store: store}
}

// GetSpeakers returns information about the speaker (bcg apps) under our management
func (dms *DistributedMgmtService) GetSpeakers() []*Speaker {
	var speakers []*Speaker
	for _, member := range cluster.FilterMembers(cluster.Music, dms.nodes) {
		displayName := member.Name
		speakerConfig, err := dms.store.GetSpeakerConfig(member.Name)
		if err != nil {
			log.Printf("Error retrieving config for: %s. Error: %s", member.Name, err)
		}
		if speakerConfig.DisplayName != "" {
			displayName = speakerConfig.DisplayName
		}
		speaker := &Speaker{ID: member.Name, DisplayName: displayName}
		speakers = append(speakers, speaker)
	}

	return speakers
}

// SetDisplayName will change the user visible name of the speaker
func (dms *DistributedMgmtService) SetDisplayName(ID string, displayName string) error {
	speakerConfig, err := dms.store.GetSpeakerConfig(ID)
	if err != nil {
		log.Printf("Error retrieving config for: %s. Error: %s", ID, err)
	}
	if speakerConfig.ID == "" {
		speakerConfig.ID = ID
	}
	speakerConfig.DisplayName = displayName
	return dms.store.SaveSpeakerConfig(speakerConfig)
}
