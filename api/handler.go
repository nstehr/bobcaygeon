package api

import (
	"log"

	"github.com/nstehr/bobcaygeon/raop"

	"golang.org/x/net/context"
)

// Server represents the gRPC server
type Server struct {
	airplayServer *raop.AirplayServer
}

// NewServer instantiates a new RPC server
func NewServer(airplayServer *raop.AirplayServer) *Server {
	return &Server{airplayServer: airplayServer}
}

// ToggleBroadcast tells node to broadcast that it is an airplay service
func (s *Server) ToggleBroadcast(ctx context.Context, in *BroadcastRequest) (*ManagementResponse, error) {
	s.airplayServer.ToggleAdvertise(in.ShouldBroadcast)
	return &ManagementResponse{ReturnCode: 200}, nil
}

// ChangeServiceName will change the name of that is broadcast for the airplay service
func (s *Server) ChangeServiceName(ctx context.Context, in *NameChangeRequest) (*ManagementResponse, error) {
	err := s.airplayServer.ChangeName(in.NewName)
	returnCode := 200
	if err != nil {
		log.Println("Problem changing name: ", err)
		returnCode = 400
	}
	return &ManagementResponse{ReturnCode: int32(returnCode)}, nil
}
