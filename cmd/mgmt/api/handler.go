package api

import (
	"github.com/nstehr/bobcaygeon/cmd/mgmt/service"

	context "golang.org/x/net/context"
)

// Server represents the gRPC server
type Server struct {
	service service.MgmtService
}

// NewServer instantiates a new RPC server
func NewServer(service service.MgmtService) *Server {
	return &Server{service: service}
}

// GetSpeakers will get all the music playing nodes
func (s *Server) GetSpeakers(ctx context.Context, in *GetSpeakersRequest) (*GetSpeakersResponse, error) {
	var speakers []*Speaker
	for _, member := range s.service.GetSpeakers() {
		speaker := &Speaker{Id: member.ID, DisplayName: member.DisplayName}
		speakers = append(speakers, speaker)
	}
	return &GetSpeakersResponse{Speakers: speakers}, nil
}
