package api

import (
	"log"

	"github.com/hashicorp/memberlist"
	"github.com/nstehr/bobcaygeon/cluster"
	"github.com/nstehr/bobcaygeon/raop"
	"golang.org/x/net/context"
)

// Server represents the gRPC server
type Server struct {
	airplayServer    *raop.AirplayServer
	forwardingPlayer *cluster.ForwardingPlayer
	nodes            *memberlist.Memberlist
}

// NewServer instantiates a new RPC server
func NewServer(airplayServer *raop.AirplayServer, forwardingPlayer *cluster.ForwardingPlayer, nodes *memberlist.Memberlist) *Server {
	return &Server{airplayServer: airplayServer, forwardingPlayer: forwardingPlayer, nodes: nodes}
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

// ForwardToNodes adds nodes to forward music to
func (s *Server) ForwardToNodes(ctx context.Context, in *AddRemoveNodesRequest) (*ManagementResponse, error) {
	self := s.nodes.LocalNode().Name

	filter := func(node *memberlist.Node) bool {
		if node.Name == self {
			return false
		}
		for _, name := range in.GetIds() {
			if name == node.Name {
				return true
			}
		}
		return false
	}

	nodesToAdd := cluster.FilterMembersByFn(filter, s.nodes)
	for _, nodeToAdd := range nodesToAdd {
		s.forwardingPlayer.AddSessionForNode(nodeToAdd)
	}
	return &ManagementResponse{ReturnCode: int32(200)}, nil
}

// RemoveForwardToNodes removes nodes we were forwarding music to
func (s *Server) RemoveForwardToNodes(ctx context.Context, in *AddRemoveNodesRequest) (*ManagementResponse, error) {
	self := s.nodes.LocalNode().Name
	filter := func(node *memberlist.Node) bool {
		if node.Name == self {
			return false
		}
		for _, name := range in.GetIds() {
			if name == node.Name {
				return true
			}
		}
		return false
	}

	if !in.GetRemoveAll() {
		nodesToRemove := cluster.FilterMembersByFn(filter, s.nodes)
		for _, nodeToRemove := range nodesToRemove {
			s.forwardingPlayer.RemoveSessionForNode(nodeToRemove)
		}
	} else {
		s.forwardingPlayer.RemoveAllSessions()
	}

	return &ManagementResponse{ReturnCode: int32(200)}, nil
}
