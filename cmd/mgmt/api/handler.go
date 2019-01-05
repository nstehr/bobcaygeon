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
	return &GetSpeakersResponse{ReturnCode: 200, Speakers: speakers}, nil
}

// SetDisplayNameForSpeaker will update the speakers display name
func (s *Server) SetDisplayNameForSpeaker(ctx context.Context, in *SetSpeakerDisplayNameRequest) (*UpdateResponse, error) {
	err := s.service.SetDisplayName(in.SpeakerId, in.DisplayName)
	if err != nil {
		return &UpdateResponse{ResponseCode: 500, Message: err.Error()}, nil
	}
	return &UpdateResponse{ResponseCode: 200}, nil
}
