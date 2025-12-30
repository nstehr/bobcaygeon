package server

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	api "github.com/nstehr/bobcaygeon/cmd/mgmt/api"
)

// ManagementClient handles communication with management servers
type ManagementClient struct {
	endpoints []MgmtEndpoint
	conns     []*grpc.ClientConn
	mu        sync.RWMutex
}

// NewManagementClient creates a new management client
func NewManagementClient(endpoints []MgmtEndpoint) *ManagementClient {
	mc := &ManagementClient{
		endpoints: endpoints,
		conns:     make([]*grpc.ClientConn, 0, len(endpoints)),
	}

	// Create connections to all endpoints
	for _, ep := range endpoints {
		conn, err := mc.createConnection(ep)
		if err != nil {
			log.Printf("Failed to connect to mgmt endpoint %s:%d: %v", ep.Host, ep.Port, err)
			continue
		}
		mc.conns = append(mc.conns, conn)
	}

	return mc
}

// createConnection creates a gRPC connection to an endpoint
func (mc *ManagementClient) createConnection(endpoint MgmtEndpoint) (*grpc.ClientConn, error) {
	target := fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)

	// NewClient creates a client connection but doesn't block
	// The connection happens lazily when the first RPC is made
	return grpc.NewClient(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

// getRandomConnection returns a random healthy connection
func (mc *ManagementClient) getRandomConnection() (*grpc.ClientConn, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if len(mc.conns) == 0 {
		return nil, fmt.Errorf("no management servers available")
	}

	// Simple random load balancing
	idx := rand.Intn(len(mc.conns))
	return mc.conns[idx], nil
}

// GetSpeakers retrieves all speakers
func (mc *ManagementClient) GetSpeakers(ctx context.Context) (*api.GetSpeakersResponse, error) {
	conn, err := mc.getRandomConnection()
	if err != nil {
		return nil, err
	}

	client := api.NewBobcaygeonManagementClient(conn)
	return client.GetSpeakers(ctx, &api.GetSpeakersRequest{})
}

// GetZones retrieves all zones
func (mc *ManagementClient) GetZones(ctx context.Context) (*api.GetZonesResponse, error) {
	conn, err := mc.getRandomConnection()
	if err != nil {
		return nil, err
	}

	client := api.NewBobcaygeonManagementClient(conn)
	return client.GetZones(ctx, &api.GetZonesRequest{})
}

// GetCurrentTrack gets the current track for a speaker
func (mc *ManagementClient) GetCurrentTrack(ctx context.Context, speakerID string) (*api.Track, error) {
	conn, err := mc.getRandomConnection()
	if err != nil {
		return nil, err
	}

	client := api.NewBobcaygeonManagementClient(conn)
	return client.GetCurrentTrack(ctx, &api.GetTrackRequest{
		SpeakerId: speakerID,
	})
}

// SetSpeakerDisplayName updates a speaker's display name
func (mc *ManagementClient) SetSpeakerDisplayName(ctx context.Context, speakerID, displayName string, updateBroadcast bool) (*api.UpdateResponse, error) {
	conn, err := mc.getRandomConnection()
	if err != nil {
		return nil, err
	}

	client := api.NewBobcaygeonManagementClient(conn)
	return client.SetDisplayNameForSpeaker(ctx, &api.SetSpeakerDisplayNameRequest{
		SpeakerId:       speakerID,
		DisplayName:     displayName,
		UpdateBroadcast: updateBroadcast,
	})
}

// GetMuteForSpeaker gets the mute status of a speaker
func (mc *ManagementClient) GetMuteForSpeaker(ctx context.Context, speakerID string) (*api.SpeakerMuteResponse, error) {
	conn, err := mc.getRandomConnection()
	if err != nil {
		return nil, err
	}

	client := api.NewBobcaygeonManagementClient(conn)
	return client.GetMuteForSpeaker(ctx, &api.GetMuteRequest{
		SpeakerId: speakerID,
	})
}

// SetMuteForSpeaker sets the mute status of a speaker
func (mc *ManagementClient) SetMuteForSpeaker(ctx context.Context, speakerID string, isMuted bool) (*api.UpdateResponse, error) {
	conn, err := mc.getRandomConnection()
	if err != nil {
		return nil, err
	}

	client := api.NewBobcaygeonManagementClient(conn)
	return client.SetMuteForSpeaker(ctx, &api.SetMuteRequest{
		SpeakerId: speakerID,
		IsMuted:   isMuted,
	})
}

// Close closes all connections
func (mc *ManagementClient) Close() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	for _, conn := range mc.conns {
		conn.Close()
	}
	mc.conns = nil
}
