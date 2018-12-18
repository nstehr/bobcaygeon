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

// GetNodes will get all the music playing nodes
func (s *Server) GetNodes(ctx context.Context, in *GetNodesRequest) (*GetNodesResponse, error) {
	var nodes []*Node
	for _, member := range cluster.FilterMembers(cluster.Music, s.nodes) {
		node := &Node{Id: member.Name, DisplayName: member.Name}
		nodes = append(nodes, node)
	}
	return &GetNodesResponse{Nodes: nodes}, nil
}
