package server

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/nstehr/bobcaygeon/cluster"
	"github.com/nstehr/bobcaygeon/player/hls"
)

// MgmtEndpoint represents a management server endpoint
type MgmtEndpoint struct {
	Host string
	Port uint32
}

// Server represents the HTTP server for the frontend
type Server struct {
	webPort       int
	apiPort       int
	mgmtEndpoints []MgmtEndpoint
	mgmtClient    *ManagementClient
	memberlist    *memberlist.Memberlist
	mu            sync.RWMutex
	router        *http.ServeMux
	// HLS streaming support
	streamManager  *hls.StreamManager
	hlsDir         string
	rtspBasePort   int
	virtualSpeaker *VirtualSpeaker
	virtualAPIPort int
}

// ServerOption is a function that configures a Server
type ServerOption func(*Server)

// WithHLSConfig configures HLS streaming settings
func WithHLSConfig(hlsDir string, maxSessions int, rtspBasePort int, virtualAPIPort int) ServerOption {
	return func(s *Server) {
		s.hlsDir = hlsDir
		s.rtspBasePort = rtspBasePort
		s.virtualAPIPort = virtualAPIPort
		s.streamManager = hls.NewStreamManager(hlsDir, maxSessions, rtspBasePort)
	}
}

// New creates a new frontend server
func New(webPort, apiPort int, initialEndpoints []MgmtEndpoint, ml *memberlist.Memberlist, opts ...ServerOption) *Server {
	s := &Server{
		webPort:       webPort,
		apiPort:       apiPort,
		mgmtEndpoints: initialEndpoints,
		memberlist:    ml,
		router:        http.NewServeMux(),
		hlsDir:        "./hls_output", // default
		rtspBasePort:  5001,           // default
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Initialize management client with initial endpoints
	if len(initialEndpoints) > 0 {
		s.mgmtClient = NewManagementClient(initialEndpoints)
	}

	// Initialize virtual speaker if HLS is configured
	if s.streamManager != nil {
		virtualSpeaker, err := NewVirtualSpeaker(ml, s.rtspBasePort, s.hlsDir, s.virtualAPIPort)
		if err != nil {
			log.Printf("Failed to create virtual speaker: %v", err)
		} else {
			s.virtualSpeaker = virtualSpeaker
		}
	}

	// Set up routes
	s.setupRoutes()

	// Start cleanup goroutine if HLS is enabled
	if s.streamManager != nil {
		go s.startSessionCleanup()
	}

	return s
}

// startSessionCleanup periodically cleans up inactive HLS sessions
func (s *Server) startSessionCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		s.streamManager.CleanupInactiveSessions(30 * time.Minute)
	}
}

// Start starts the HTTP server
func (s *Server) Start() {
	log.Printf("Starting web server on port %d\n", s.webPort)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.webPort),
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Static files
	s.router.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Main page
	s.router.HandleFunc("/", s.handleIndex)

	// API endpoints
	s.router.HandleFunc("/api/speakers", s.handleGetSpeakers)
	s.router.HandleFunc("/api/zones", s.handleGetZones)
	s.router.HandleFunc("/api/speaker/", s.handleSpeakerOperations)
	s.router.HandleFunc("/api/now-playing/", s.handleNowPlaying)

	// Web player endpoints (if HLS is enabled)
	if s.streamManager != nil {
		s.router.HandleFunc("/api/web-player", s.handleWebPlayer)
		s.router.HandleFunc("/api/web-player/enable", s.handleEnableWebPlayback)
		s.router.HandleFunc("/api/web-player/disable", s.handleDisableWebPlayback)
		s.router.HandleFunc("/api/web-player/stream-status", s.handleStreamStatus)

		// HLS file serving
		s.router.HandleFunc("/hls/", s.handleHLSFiles)
	}

	// Health check
	s.router.HandleFunc("/health", s.handleHealth)
}

// NotifyJoin is called when a node joins the cluster
func (s *Server) NotifyJoin(node *memberlist.Node) {
	log.Printf("Node joined: %s\n", node.Name)
	meta := cluster.DecodeNodeMeta(node.Meta)
	if meta.NodeType == cluster.Mgmt {
		s.addMgmtEndpoint(MgmtEndpoint{
			Host: node.Addr.String(),
			Port: uint32(meta.APIPort),
		})
	}
}

// NotifyLeave is called when a node leaves the cluster
func (s *Server) NotifyLeave(node *memberlist.Node) {
	log.Printf("Node left: %s\n", node.Name)
	meta := cluster.DecodeNodeMeta(node.Meta)
	if meta.NodeType == cluster.Mgmt {
		s.removeMgmtEndpoint(MgmtEndpoint{
			Host: node.Addr.String(),
			Port: uint32(meta.APIPort),
		})
	}
}

// NotifyUpdate is called when a node is updated
func (s *Server) NotifyUpdate(node *memberlist.Node) {
	log.Printf("Node updated: %s\n", node.Name)
}

// addMgmtEndpoint adds a management endpoint
func (s *Server) addMgmtEndpoint(endpoint MgmtEndpoint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already exists
	for _, ep := range s.mgmtEndpoints {
		if ep.Host == endpoint.Host && ep.Port == endpoint.Port {
			return
		}
	}

	s.mgmtEndpoints = append(s.mgmtEndpoints, endpoint)

	// Recreate client with new endpoints
	if s.mgmtClient != nil {
		s.mgmtClient.Close()
	}
	s.mgmtClient = NewManagementClient(s.mgmtEndpoints)
}

// removeMgmtEndpoint removes a management endpoint
func (s *Server) removeMgmtEndpoint(endpoint MgmtEndpoint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, ep := range s.mgmtEndpoints {
		if ep.Host == endpoint.Host && ep.Port == endpoint.Port {
			s.mgmtEndpoints = append(s.mgmtEndpoints[:i], s.mgmtEndpoints[i+1:]...)

			// Recreate client with remaining endpoints
			if s.mgmtClient != nil {
				s.mgmtClient.Close()
			}
			if len(s.mgmtEndpoints) > 0 {
				s.mgmtClient = NewManagementClient(s.mgmtEndpoints)
			}
			return
		}
	}
}

// getMgmtClient returns the management client
func (s *Server) getMgmtClient() *ManagementClient {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mgmtClient
}
