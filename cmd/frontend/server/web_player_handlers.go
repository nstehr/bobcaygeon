package server

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/memberlist"
	speakerAPI "github.com/nstehr/bobcaygeon/api"
	"github.com/nstehr/bobcaygeon/cluster"
	"github.com/nstehr/bobcaygeon/cmd/frontend/templates/components"
	api "github.com/nstehr/bobcaygeon/cmd/mgmt/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// handleWebPlayer renders the web player component
func (s *Server) handleWebPlayer(w http.ResponseWriter, r *http.Request) {
	state := s.getWebPlayerState(r)

	component := components.WebPlayer(state)
	component.Render(r.Context(), w)
}

// getWebPlayerState builds the current state for the web player
func (s *Server) getWebPlayerState(r *http.Request) components.WebPlayerState {
	// Check for existing session cookie
	cookie, _ := r.Cookie("web_player_session")

	state := components.WebPlayerState{
		IsEnabled: false,
	}

	if cookie != nil && s.streamManager != nil {
		if session, exists := s.streamManager.GetSession(cookie.Value); exists {
			state.IsEnabled = true
			state.SessionID = session.SessionID
			state.PlaylistURL = session.PlaylistURL
			state.SelectedZone = session.ZoneID
			state.Mode = session.Mode
		}
	}

	// Get zones and speakers
	zones := s.getZonesFromMgmt()
	speakers := s.getSpeakersFromCluster()

	state.Zones = zones
	state.Speakers = speakers

	// Determine mode based on zones
	if len(zones) == 0 {
		state.Mode = "default"
	} else if len(zones) == 1 {
		state.Mode = "single-zone"
	} else {
		state.Mode = "multi-zone"
	}

	return state
}

// handleEnableWebPlayback enables web playback for a zone or default mode
func (s *Server) handleEnableWebPlayback(w http.ResponseWriter, r *http.Request) {
	if s.streamManager == nil {
		http.Error(w, "HLS streaming not configured", http.StatusServiceUnavailable)
		return
	}

	r.ParseForm()

	mode := r.FormValue("mode")
	var speakerID string
	var zoneID string
	var zoneName string

	if mode == "default" {
		// No zones - pick a random speaker
		speakers := s.getSpeakersFromCluster()
		if len(speakers) > 0 {
			speaker := speakers[rand.Intn(len(speakers))]
			speakerID = speaker.Id
		}
		zoneID = "default"
		zoneName = "All Speakers"
	} else {
		// Zone mode - pick random speaker from zone
		zoneID = r.FormValue("zoneId")
		zone := s.getZoneFromMgmt(zoneID)
		if zone != nil && len(zone.Speakers) > 0 {
			speaker := zone.Speakers[rand.Intn(len(zone.Speakers))]
			speakerID = speaker.Id
			zoneName = zone.DisplayName
		}
	}

	if speakerID == "" {
		state := s.getWebPlayerState(r)
		state.ErrorMessage = "No speaker available"
		components.WebPlayer(state).Render(r.Context(), w)
		return
	}

	// Create streaming session
	log.Printf("[WebPlayer] Creating session for speaker %s, zone %s (%s), mode %s", speakerID, zoneName, zoneID, mode)
	session, stream, isNewStream, err := s.streamManager.CreateSession(speakerID, zoneID, mode)
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		state := s.getWebPlayerState(r)
		state.ErrorMessage = fmt.Sprintf("Failed to create session: %v", err)
		components.WebPlayer(state).Render(r.Context(), w)
		return
	}

	log.Printf("[WebPlayer] Session created successfully")
	log.Printf("[WebPlayer]   Session ID: %s", session.SessionID)
	log.Printf("[WebPlayer]   Playlist URL: %s", session.PlaylistURL)
	log.Printf("[WebPlayer]   Is new stream: %v", isNewStream)
	log.Printf("[WebPlayer]   RTSP port: %d", stream.RtspPort)

	// If this is a new stream, activate virtual speaker and tell the speaker to forward to us
	if isNewStream {
		log.Printf("[WebPlayer] This is a new stream, activating virtual speaker...")

		// Check if virtual speaker is available
		if s.virtualSpeaker == nil {
			log.Printf("[WebPlayer] ERROR: Virtual speaker not available")
			s.streamManager.RemoveSession(session.SessionID)
			state := s.getWebPlayerState(r)
			state.ErrorMessage = "Virtual speaker not available"
			components.WebPlayer(state).Render(r.Context(), w)
			return
		}

		// Start virtual speaker if not already active
		if !s.virtualSpeaker.IsActive() {
			virtualName := fmt.Sprintf("WebPlayer-%s", s.memberlist.LocalNode().Name)
			err = s.virtualSpeaker.Start(virtualName)
			if err != nil {
				log.Printf("Failed to start virtual speaker: %v", err)
				s.streamManager.RemoveSession(session.SessionID)
				state := s.getWebPlayerState(r)
				state.ErrorMessage = fmt.Sprintf("Failed to start virtual speaker: %v", err)
				components.WebPlayer(state).Render(r.Context(), w)
				return
			}
			log.Printf("[WebPlayer] Virtual speaker started successfully")

			// Wait a moment for virtual speaker to join cluster
			log.Printf("[WebPlayer] Waiting for virtual speaker to join cluster...")
			time.Sleep(2 * time.Second)
			log.Printf("[WebPlayer] Virtual speaker should be in cluster now")
		}

		// Now tell the original speaker to forward to the virtual speaker
		err = s.enableSpeakerForwardingToVirtual(speakerID)
		if err != nil {
			log.Printf("Failed to enable forwarding: %v", err)
			s.streamManager.RemoveSession(session.SessionID)
			state := s.getWebPlayerState(r)
			state.ErrorMessage = fmt.Sprintf("Failed to enable forwarding: %v", err)
			components.WebPlayer(state).Render(r.Context(), w)
			return
		}
		log.Printf("[WebPlayer] Forwarding enabled successfully")

		// Wait a moment to verify the connection is established
		log.Printf("[WebPlayer] Waiting for audio connection to establish...")
		time.Sleep(1 * time.Second)

		// Verify that HLS segments are being generated
		log.Printf("[WebPlayer] Checking for HLS segment generation...")
		if !s.waitForHLSSegments(speakerID, 10*time.Second) {
			log.Printf("[WebPlayer] WARNING: No HLS segments generated after 10 seconds. Audio may not be flowing.")
			log.Printf("[WebPlayer] This usually means the speaker is not forwarding audio to the virtual speaker.")
			// Don't fail here - let the user try anyway, but log the warning
		} else {
			log.Printf("[WebPlayer] HLS segments confirmed - audio is flowing!")
		}
	} else {
		log.Printf("[WebPlayer] Reusing existing stream for speaker %s (ref count: %d)", speakerID, stream.RefCount)
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "web_player_session",
		Value:    session.SessionID,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400, // 24 hours
	})

	// Return updated component
	state := components.WebPlayerState{
		IsEnabled:       true,
		SessionID:       session.SessionID,
		PlaylistURL:     session.PlaylistURL,
		SelectedZone:    zoneName,
		SelectedSpeaker: speakerID,
		Mode:            mode,
	}

	components.WebPlayer(state).Render(r.Context(), w)
}

// handleDisableWebPlayback disables web playback
func (s *Server) handleDisableWebPlayback(w http.ResponseWriter, r *http.Request) {
	if s.streamManager == nil {
		http.Error(w, "HLS streaming not configured", http.StatusServiceUnavailable)
		return
	}

	// Parse form data (HTMX sends form-encoded data)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	sessionID := r.FormValue("sessionId")
	if sessionID == "" {
		http.Error(w, "Missing sessionId", http.StatusBadRequest)
		return
	}

	session, exists := s.streamManager.GetSession(sessionID)
	if !exists {
		state := s.getWebPlayerState(r)
		state.ErrorMessage = "Session not found"
		components.WebPlayer(state).Render(r.Context(), w)
		return
	}

	speakerID := session.SpeakerID

	// Remove session
	s.streamManager.RemoveSession(sessionID)

	// Check if this was the last session for this speaker
	if _, exists := s.streamManager.GetStream(speakerID); !exists {
		// No more users, disable forwarding from the speaker
		s.disableSpeakerForwardingFromVirtual(speakerID)

		// Check if there are any remaining active streams at all
		if s.streamManager.GetActiveStreamCount() == 0 {
			// No more streams, stop the virtual speaker
			if s.virtualSpeaker != nil && s.virtualSpeaker.IsActive() {
				log.Printf("No more active streams, stopping virtual speaker")
				s.virtualSpeaker.Stop()
			}
		}
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "web_player_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	// Return fresh component
	state := s.getWebPlayerState(r)
	components.WebPlayer(state).Render(r.Context(), w)
}

// handleStreamStatus provides Server-Sent Events for track updates
func (s *Server) handleStreamStatus(w http.ResponseWriter, r *http.Request) {
	if s.streamManager == nil {
		http.Error(w, "HLS streaming not configured", http.StatusServiceUnavailable)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "Missing sessionId", http.StatusBadRequest)
		return
	}

	session, exists := s.streamManager.GetSession(sessionID)
	if !exists {
		// Return proper SSE response even when session doesn't exist
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		fmt.Fprintf(w, "event: error\n")
		fmt.Fprintf(w, "data: session_not_found\n\n")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering if present

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial ping to establish connection
	fmt.Fprintf(w, "event: ping\n")
	fmt.Fprintf(w, "data: connected\n\n")
	flusher.Flush()

	// Send track updates every 2 seconds
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if session still exists
			if _, exists := s.streamManager.GetSession(sessionID); !exists {
				// Session was removed, close SSE connection gracefully
				fmt.Fprintf(w, "event: close\n")
				fmt.Fprintf(w, "data: session_ended\n\n")
				flusher.Flush()
				return
			}

			// Get current track from stream
			if track, exists := s.streamManager.GetTrackForSpeaker(session.SpeakerID); exists {
				// Render track info component
				trackData := components.TrackData{
					Title:   track.Title,
					Artist:  track.Artist,
					Album:   track.Album,
					Artwork: track.Artwork,
				}

				// Create a buffer for the rendered HTML
				var buf bytes.Buffer
				if err := components.TrackInfo(trackData).Render(r.Context(), &buf); err != nil {
					log.Printf("Error rendering track info: %v", err)
					continue
				}

				// Send SSE message with rendered HTML
				// Each line of data must be prefixed with "data: "
				lines := strings.Split(buf.String(), "\n")
				_, err := fmt.Fprintf(w, "event: message\n")
				if err != nil {
					// Connection closed by client
					return
				}
				for _, line := range lines {
					if line != "" {
						fmt.Fprintf(w, "data: %s\n", line)
					}
				}
				fmt.Fprintf(w, "\n")
				flusher.Flush()
			}

		case <-r.Context().Done():
			return
		}
	}
}

// handleHLSFiles serves HLS playlist and segment files
func (s *Server) handleHLSFiles(w http.ResponseWriter, r *http.Request) {
	if s.streamManager == nil {
		http.Error(w, "HLS streaming not configured", http.StatusServiceUnavailable)
		return
	}

	// Path format: /hls/{speakerID}/stream.m3u8 or /hls/{speakerID}/segment_XXX.ts
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		log.Printf("[HLS] Invalid path format: %s (parts: %v)", r.URL.Path, parts)
		http.NotFound(w, r)
		return
	}

	speakerID := parts[2]
	fileName := parts[3]

	// Verify stream exists
	stream, exists := s.streamManager.GetStream(speakerID)
	if !exists {
		log.Printf("[HLS] Stream not found for speaker: %s (requested file: %s)", speakerID, fileName)
		log.Printf("[HLS] Available streams: %d", s.streamManager.GetActiveStreamCount())
		http.NotFound(w, r)
		return
	}

	log.Printf("[HLS] Serving %s for speaker %s (stream active, ref count: %d)", fileName, speakerID, stream.RefCount)

	// Serve files from speaker-specific directory
	speakerDir := filepath.Join(s.hlsDir, speakerID)
	http.StripPrefix(fmt.Sprintf("/hls/%s/", speakerID),
		http.FileServer(http.Dir(speakerDir))).ServeHTTP(w, r)
}

// Helper: Enable forwarding from speaker to virtual speaker
func (s *Server) enableSpeakerForwardingToVirtual(speakerID string) error {
	log.Printf("[WebPlayer] Enabling forwarding from speaker %s to virtual speaker", speakerID)

	// Find speaker node
	filter := func(node *memberlist.Node) bool {
		meta := cluster.DecodeNodeMeta(node.Meta)
		return meta.NodeType == cluster.Music && node.Name == speakerID
	}

	speakers := cluster.FilterMembersByFn(filter, s.memberlist)
	if len(speakers) != 1 {
		log.Printf("[WebPlayer] ERROR: Speaker not found in cluster: %s (found %d matches)", speakerID, len(speakers))
		return fmt.Errorf("speaker not found: %s", speakerID)
	}

	speaker := speakers[0]
	meta := cluster.DecodeNodeMeta(speaker.Meta)

	log.Printf("[WebPlayer] Found speaker %s at %s:%d (API port %d)", speakerID, speaker.Addr.String(), speaker.Port, meta.APIPort)

	// Connect to speaker's gRPC API
	speakerAPIAddress := fmt.Sprintf("%s:%d", speaker.Addr.String(), meta.APIPort)
	log.Printf("[WebPlayer] Connecting to speaker gRPC API at %s...", speakerAPIAddress)
	conn, err := grpc.Dial(speakerAPIAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("[WebPlayer] ERROR: Failed to connect to speaker gRPC at %s: %v", speakerAPIAddress, err)
		return fmt.Errorf("failed to connect to speaker: %w", err)
	}
	defer conn.Close()
	log.Printf("[WebPlayer] Connected to speaker gRPC API")

	client := speakerAPI.NewAirPlayManagementClient(conn)

	// Tell speaker to forward to the virtual speaker (which appears as a Music node)
	virtualNodeName := fmt.Sprintf("WebPlayer-%s", s.memberlist.LocalNode().Name)
	log.Printf("[WebPlayer] Sending ForwardToNodes request: %s -> %s", speakerID, virtualNodeName)

	resp, err := client.ForwardToNodes(context.Background(), &speakerAPI.AddRemoveNodesRequest{
		Ids: []string{virtualNodeName},
	})

	if err != nil {
		log.Printf("[WebPlayer] ERROR: ForwardToNodes RPC failed: %v", err)
		return err
	}

	log.Printf("[WebPlayer] ForwardToNodes response: return code %d", resp.ReturnCode)
	if resp.ReturnCode != 200 {
		log.Printf("[WebPlayer] WARNING: Unexpected return code from ForwardToNodes: %d", resp.ReturnCode)
	}
	log.Printf("[WebPlayer] Successfully enabled forwarding from %s to %s", speakerID, virtualNodeName)

	return nil
}

// Helper: Disable forwarding from speaker to virtual speaker
func (s *Server) disableSpeakerForwardingFromVirtual(speakerID string) error {
	// Find speaker node
	filter := func(node *memberlist.Node) bool {
		meta := cluster.DecodeNodeMeta(node.Meta)
		return meta.NodeType == cluster.Music && node.Name == speakerID
	}

	speakers := cluster.FilterMembersByFn(filter, s.memberlist)
	if len(speakers) != 1 {
		return fmt.Errorf("speaker not found: %s", speakerID)
	}

	speaker := speakers[0]
	meta := cluster.DecodeNodeMeta(speaker.Meta)

	// Connect to speaker's gRPC API
	speakerAPIAddress := fmt.Sprintf("%s:%d", speaker.Addr.String(), meta.APIPort)
	conn, err := grpc.Dial(speakerAPIAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to speaker: %w", err)
	}
	defer conn.Close()

	client := speakerAPI.NewAirPlayManagementClient(conn)

	// Tell speaker to stop forwarding to the virtual speaker
	virtualNodeName := fmt.Sprintf("WebPlayer-%s", s.memberlist.LocalNode().Name)
	_, err = client.RemoveForwardToNodes(context.Background(), &speakerAPI.AddRemoveNodesRequest{
		Ids: []string{virtualNodeName},
	})

	return err
}

// Helper: Get zones from management service
func (s *Server) getZonesFromMgmt() []*api.Zone {
	s.mu.RLock()
	client := s.mgmtClient
	s.mu.RUnlock()

	if client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetZones(ctx)
	if err != nil {
		log.Printf("Failed to get zones: %v", err)
		return nil
	}

	return resp.Zones
}

// Helper: Get a specific zone from management service
func (s *Server) getZoneFromMgmt(zoneID string) *api.Zone {
	zones := s.getZonesFromMgmt()
	for _, zone := range zones {
		if zone.Id == zoneID {
			return zone
		}
	}
	return nil
}

// Helper: Get all speakers from cluster
func (s *Server) getSpeakersFromCluster() []*api.Speaker {
	musicNodes := cluster.FilterMembers(cluster.Music, s.memberlist)

	speakers := make([]*api.Speaker, 0, len(musicNodes))
	for _, node := range musicNodes {
		speakers = append(speakers, &api.Speaker{
			Id:          node.Name,
			DisplayName: node.Name, // Could enhance with stored display names
		})
	}

	return speakers
}

// Helper: Wait for HLS segments to be generated
func (s *Server) waitForHLSSegments(speakerID string, timeout time.Duration) bool {
	hlsDir := filepath.Join(s.hlsDir, speakerID)
	playlistPath := filepath.Join(hlsDir, "stream.m3u8")

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// Check if playlist exists and has content
		if info, err := os.Stat(playlistPath); err == nil && info.Size() > 0 {
			// Check if there are any .ts segment files
			segments, err := filepath.Glob(filepath.Join(hlsDir, "*.ts"))
			if err == nil && len(segments) > 0 {
				log.Printf("HLS segments detected: %d files in %s", len(segments), hlsDir)
				return true
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}
