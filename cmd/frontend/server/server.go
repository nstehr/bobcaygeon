package server

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/nstehr/bobcaygeon/cluster"
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
}

// New creates a new frontend server
func New(webPort, apiPort int, initialEndpoints []MgmtEndpoint, ml *memberlist.Memberlist) *Server {
	s := &Server{
		webPort:       webPort,
		apiPort:       apiPort,
		mgmtEndpoints: initialEndpoints,
		memberlist:    ml,
		router:        http.NewServeMux(),
	}

	// Initialize management client with initial endpoints
	if len(initialEndpoints) > 0 {
		s.mgmtClient = NewManagementClient(initialEndpoints)
	}

	// Set up routes
	s.setupRoutes()

	return s
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
