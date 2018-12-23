package api

import (
	"github.com/nstehr/bobcaygeon/cluster"

	"github.com/hashicorp/memberlist"
	context "golang.org/x/net/context"
)

// Server represents the gRPC server
type Server struct {
	nodes *memberlist.Memberlist
}

// NewServer instantiates a new RPC server
func NewServer(list *memberlist.Memberlist) *Server {
	return &Server{nodes: list}
}

// GetSpeakers will get all the music playing nodes
func (s *Server) GetSpeakers(ctx context.Context, in *GetSpeakersRequest) (*GetSpeakersResponse, error) {
	var speakers []*Speaker
	for _, member := range cluster.FilterMembers(cluster.Music, s.nodes) {
		speaker := &Speaker{Id: member.Name, DisplayName: member.Name}
		speakers = append(speakers, speaker)
	}
	return &GetSpeakersResponse{Speakers: speakers}, nil
}
