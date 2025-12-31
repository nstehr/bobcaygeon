package server

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/nstehr/bobcaygeon/cmd/frontend/templates"
	"github.com/nstehr/bobcaygeon/cmd/frontend/templates/components"
)

// handleIndex serves the main page
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Fetch speakers server-side for initial render
	mgmtClient := s.getMgmtClient()
	if mgmtClient == nil {
		// No management client available, show empty list
		templates.Index(nil).Render(r.Context(), w)
		return
	}

	speakers, err := mgmtClient.GetSpeakers(r.Context())
	if err != nil {
		log.Printf("Error getting speakers for index: %v", err)
		// Show empty list on error
		templates.Index(nil).Render(r.Context(), w)
		return
	}

	templates.Index(speakers.Speakers).Render(r.Context(), w)
}

// handleHealth returns the health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	mgmtClient := s.getMgmtClient()
	status := "healthy"
	if mgmtClient == nil {
		status = "degraded - no management servers"
	}

	response := map[string]string{
		"status": status,
		"node":   s.memberlist.LocalNode().Name,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetSpeakers returns all speakers
func (s *Server) handleGetSpeakers(w http.ResponseWriter, r *http.Request) {
	mgmtClient := s.getMgmtClient()
	if mgmtClient == nil {
		http.Error(w, "No management servers available", http.StatusServiceUnavailable)
		return
	}

	speakers, err := mgmtClient.GetSpeakers(r.Context())
	if err != nil {
		log.Printf("Error getting speakers: %v", err)
		http.Error(w, "Failed to get speakers", http.StatusInternalServerError)
		return
	}

	// For HTMX requests, return HTML fragment
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		components.SpeakersList(speakers.Speakers).Render(r.Context(), w)
		return
	}

	// For regular requests, return JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(speakers)
}

// handleGetZones returns all zones
func (s *Server) handleGetZones(w http.ResponseWriter, r *http.Request) {
	mgmtClient := s.getMgmtClient()
	if mgmtClient == nil {
		http.Error(w, "No management servers available", http.StatusServiceUnavailable)
		return
	}

	zones, err := mgmtClient.GetZones(r.Context())
	if err != nil {
		log.Printf("Error getting zones: %v", err)
		http.Error(w, "Failed to get zones", http.StatusInternalServerError)
		return
	}

	// For HTMX requests, return HTML fragment
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		components.ZonesList(zones.Zones).Render(r.Context(), w)
		return
	}

	// For regular requests, return JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(zones)
}

// handleSpeakerOperations handles various speaker-specific operations
func (s *Server) handleSpeakerOperations(w http.ResponseWriter, r *http.Request) {
	// Extract speaker ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/speaker/")
	parts := strings.Split(path, "/")

	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, "Speaker ID required", http.StatusBadRequest)
		return
	}

	// Handle different operations
	if len(parts) == 2 {
		switch parts[1] {
		case "mute":
			s.handleMuteOperations(w, r)
		case "name":
			s.handleNameUpdate(w, r)
		case "details":
			s.handleSpeakerDetails(w, r)
		default:
			http.Error(w, "Unknown operation", http.StatusNotFound)
		}
	} else {
		// Return speaker info
		s.handleGetSpeaker(w, r)
	}
}

// handleMuteOperations handles mute get/set for a speaker
func (s *Server) handleMuteOperations(w http.ResponseWriter, r *http.Request) {
	speakerID := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/speaker/"), "/")[0]

	mgmtClient := s.getMgmtClient()
	if mgmtClient == nil {
		http.Error(w, "No management servers available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		muteStatus, err := mgmtClient.GetMuteForSpeaker(r.Context(), speakerID)
		if err != nil {
			log.Printf("Error getting mute status: %v", err)
			http.Error(w, "Failed to get mute status", http.StatusInternalServerError)
			return
		}

		// Return button HTML for HTMX
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			components.MuteButton(speakerID, muteStatus.IsMuted).Render(r.Context(), w)
			return
		}

		// Return JSON for regular requests
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(muteStatus)

	case http.MethodPost:
		// Toggle mute status
		currentStatus, err := mgmtClient.GetMuteForSpeaker(r.Context(), speakerID)
		if err != nil {
			log.Printf("Error getting current mute status: %v", err)
			http.Error(w, "Failed to get current mute status", http.StatusInternalServerError)
			return
		}

		newStatus := !currentStatus.IsMuted
		_, err = mgmtClient.SetMuteForSpeaker(r.Context(), speakerID, newStatus)
		if err != nil {
			log.Printf("Error setting mute status: %v", err)
			http.Error(w, "Failed to set mute status", http.StatusInternalServerError)
			return
		}

		// Return updated button HTML for HTMX
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			components.MuteButton(speakerID, newStatus).Render(r.Context(), w)
			return
		}

		// Return JSON for regular requests
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"isMuted": newStatus})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleNameUpdate handles speaker name updates
func (s *Server) handleNameUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	speakerID := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/speaker/"), "/")[0]

	var request struct {
		DisplayName     string `json:"displayName"`
		UpdateBroadcast bool   `json:"updateBroadcast"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	mgmtClient := s.getMgmtClient()
	if mgmtClient == nil {
		http.Error(w, "No management servers available", http.StatusServiceUnavailable)
		return
	}

	response, err := mgmtClient.SetSpeakerDisplayName(r.Context(), speakerID, request.DisplayName, request.UpdateBroadcast)
	if err != nil {
		log.Printf("Error updating speaker name: %v", err)
		http.Error(w, "Failed to update speaker name", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSpeakerDetails returns detailed speaker information
func (s *Server) handleSpeakerDetails(w http.ResponseWriter, r *http.Request) {
	speakerID := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/speaker/"), "/")[0]

	mgmtClient := s.getMgmtClient()
	if mgmtClient == nil {
		http.Error(w, "No management servers available", http.StatusServiceUnavailable)
		return
	}

	// Get speaker info
	speakers, err := mgmtClient.GetSpeakers(r.Context())
	if err != nil {
		log.Printf("Error getting speakers: %v", err)
		http.Error(w, "Failed to get speaker details", http.StatusInternalServerError)
		return
	}

	// Find the specific speaker
	for _, speaker := range speakers.Speakers {
		if speaker.Id == speakerID {
			// Return HTML for HTMX
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				components.SpeakerDetails(speaker).Render(r.Context(), w)
				return
			}

			// Return JSON for regular requests
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(speaker)
			return
		}
	}

	http.Error(w, "Speaker not found", http.StatusNotFound)
}

// handleGetSpeaker returns info for a specific speaker
func (s *Server) handleGetSpeaker(w http.ResponseWriter, r *http.Request) {
	speakerID := strings.TrimPrefix(r.URL.Path, "/api/speaker/")

	mgmtClient := s.getMgmtClient()
	if mgmtClient == nil {
		http.Error(w, "No management servers available", http.StatusServiceUnavailable)
		return
	}

	speakers, err := mgmtClient.GetSpeakers(r.Context())
	if err != nil {
		log.Printf("Error getting speakers: %v", err)
		http.Error(w, "Failed to get speakers", http.StatusInternalServerError)
		return
	}

	for _, speaker := range speakers.Speakers {
		if speaker.Id == speakerID {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(speaker)
			return
		}
	}

	http.Error(w, "Speaker not found", http.StatusNotFound)
}

// handleNowPlaying returns the current playing track for a speaker
func (s *Server) handleNowPlaying(w http.ResponseWriter, r *http.Request) {
	speakerID := strings.TrimPrefix(r.URL.Path, "/api/now-playing/")

	if speakerID == "" {
		http.Error(w, "Speaker ID required", http.StatusBadRequest)
		return
	}

	mgmtClient := s.getMgmtClient()
	if mgmtClient == nil {
		http.Error(w, "No management servers available", http.StatusServiceUnavailable)
		return
	}

	track, err := mgmtClient.GetCurrentTrack(r.Context(), speakerID)
	if err != nil {
		log.Printf("Error getting current track for speaker %s: %v", speakerID, err)
		// Return empty now playing for HTMX
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			components.NowPlaying(nil).Render(r.Context(), w)
			return
		}
		http.Error(w, "Failed to get current track", http.StatusInternalServerError)
		return
	}

	log.Printf("Got track for speaker %s: %s - %s (artwork size: %d bytes)",
		speakerID, track.Artist, track.Title, len(track.Artwork))

	// For HTMX requests, return HTML fragment
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		components.NowPlaying(track).Render(r.Context(), w)
		return
	}

	// For regular requests, return JSON with base64 encoded artwork
	response := map[string]interface{}{
		"artist":  track.Artist,
		"album":   track.Album,
		"title":   track.Title,
		"artwork": base64.StdEncoding.EncodeToString(track.Artwork),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
