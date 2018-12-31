package service

import (
	"github.com/hashicorp/memberlist"
	"github.com/nstehr/bobcaygeon/cluster"
)

// MgmtService interface for handling management capabilities
type MgmtService interface {
	// GetSpeakers, returns information about the speaker (bcg apps) under our management
	GetSpeakers() []*Speaker
}

// Speaker speaker instance
type Speaker struct {
	ID          string
	DisplayName string
}

// DistributedMgmtService implements MgmtService with a distributed backing store
type DistributedMgmtService struct {
	nodes *memberlist.Memberlist
}

// NewDistributedMgmtService instantiates the DistributedMgmtService
func NewDistributedMgmtService(nodes *memberlist.Memberlist) *DistributedMgmtService {
	return &DistributedMgmtService{nodes: nodes}
}

// GetSpeakers returns information about the speaker (bcg apps) under our management
func (dms *DistributedMgmtService) GetSpeakers() []*Speaker {
	var speakers []*Speaker
	for _, member := range cluster.FilterMembers(cluster.Music, dms.nodes) {
		speaker := &Speaker{ID: member.Name, DisplayName: member.Name}
		speakers = append(speakers, speaker)
	}

	return speakers
}
