package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
	speakerAPI "github.com/nstehr/bobcaygeon/api"
	"github.com/nstehr/bobcaygeon/cluster"
	"google.golang.org/grpc"
)

// VirtualSpeaker allows the frontend to act as a speaker for receiving streams
type VirtualSpeaker struct {
	memberlist     *memberlist.Memberlist
	virtualNode    *memberlist.Memberlist
	rtspPort       int // Port to advertise for RTSP (managed by StreamManager)
	isActive       bool
	mu             sync.RWMutex
	grpcServer     *grpc.Server
	virtualAPIPort int
}

// NewVirtualSpeaker creates a new virtual speaker
func NewVirtualSpeaker(ml *memberlist.Memberlist, rtspPort int, hlsDir string, virtualAPIPort int) (*VirtualSpeaker, error) {
	// The RTSP server is managed by StreamManager, we just need to advertise the port
	return &VirtualSpeaker{
		memberlist:     ml,
		rtspPort:       rtspPort,
		virtualAPIPort: virtualAPIPort,
	}, nil
}

// Start activates the virtual speaker
func (vs *VirtualSpeaker) Start(virtualName string) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if vs.isActive {
		return fmt.Errorf("virtual speaker already active")
	}

	// Create a new memberlist node that appears as a Music node
	metaData := &cluster.NodeMeta{
		NodeType: cluster.Music,
		RtspPort: vs.rtspPort, // Use the RTSP port managed by StreamManager
		APIPort:  vs.virtualAPIPort,
	}

	config := memberlist.DefaultLANConfig()
	config.Name = virtualName
	config.BindPort = 0 // Let OS assign a port
	config.AdvertisePort = config.BindPort
	config.Delegate = cluster.Delegate{MetaData: metaData}

	virtualNode, err := memberlist.Create(config)
	if err != nil {
		return fmt.Errorf("failed to create virtual node: %w", err)
	}

	// Join the existing cluster
	existingNode := vs.memberlist.LocalNode()
	_, err = virtualNode.Join([]string{fmt.Sprintf("%s:%d", existingNode.Addr, existingNode.Port)})
	if err != nil {
		virtualNode.Shutdown()
		return fmt.Errorf("failed to join cluster as virtual speaker: %w", err)
	}

	// Start a minimal gRPC server to handle speaker API calls
	vs.startGRPCServer()

	vs.virtualNode = virtualNode
	vs.isActive = true

	log.Printf("Virtual speaker '%s' started and joined cluster with RTSP port %d", virtualName, vs.rtspPort)
	return nil
}

// Stop deactivates the virtual speaker
func (vs *VirtualSpeaker) Stop() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if !vs.isActive {
		return nil
	}

	// Leave cluster
	if vs.virtualNode != nil {
		vs.virtualNode.Leave(5 * time.Second)
		vs.virtualNode.Shutdown()
		vs.virtualNode = nil
	}

	// Stop gRPC server
	if vs.grpcServer != nil {
		vs.grpcServer.Stop()
		vs.grpcServer = nil
	}

	vs.isActive = false
	log.Println("Virtual speaker stopped")
	return nil
}

// startGRPCServer starts a minimal gRPC server for the virtual speaker
func (vs *VirtualSpeaker) startGRPCServer() {
	// Start a minimal gRPC server that implements just enough of the speaker API
	// to handle basic requests
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", vs.virtualAPIPort))
		if err != nil {
			log.Printf("Failed to start virtual speaker gRPC server: %v", err)
			return
		}

		vs.grpcServer = grpc.NewServer()
		speakerAPI.RegisterAirPlayManagementServer(vs.grpcServer, &virtualSpeakerAPI{})

		if err := vs.grpcServer.Serve(lis); err != nil {
			log.Printf("Virtual speaker gRPC server error: %v", err)
		}
	}()
}

// GetRtspPort returns the RTSP port used by this virtual speaker
func (vs *VirtualSpeaker) GetRtspPort() int {
	return vs.rtspPort
}

// IsActive returns whether the virtual speaker is active
func (vs *VirtualSpeaker) IsActive() bool {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return vs.isActive
}

// virtualSpeakerAPI implements a minimal speaker API
type virtualSpeakerAPI struct {
	speakerAPI.UnimplementedAirPlayManagementServer
}

func (api *virtualSpeakerAPI) ForwardToNodes(ctx context.Context, req *speakerAPI.AddRemoveNodesRequest) (*speakerAPI.ManagementResponse, error) {
	// Virtual speaker doesn't forward to other nodes
	return &speakerAPI.ManagementResponse{ReturnCode: 200}, nil
}

func (api *virtualSpeakerAPI) RemoveForwardToNodes(ctx context.Context, req *speakerAPI.AddRemoveNodesRequest) (*speakerAPI.ManagementResponse, error) {
	return &speakerAPI.ManagementResponse{ReturnCode: 200}, nil
}

func (api *virtualSpeakerAPI) ToggleBroadcast(ctx context.Context, req *speakerAPI.BroadcastRequest) (*speakerAPI.ManagementResponse, error) {
	return &speakerAPI.ManagementResponse{ReturnCode: 200}, nil
}
