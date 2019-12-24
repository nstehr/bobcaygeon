package service

// MgmtService interface for handling management capabilities
type MgmtService interface {
	GetSpeakers() []*Speaker
	SetDisplayName(ID string, displayName string, updateBroadcast bool) error
	CreateZone(displayName string, speakerIDs []string) (string, error)
	AddSpeakersToZone(zoneID string, speakerIDs []string) error
	RemoveSpeakersFromZone(zoneID string, speakerIDs []string) error
	DeleteZone(zoneID string) error
	ChangeZoneName(zoneID string, newName string) error
	GetZones() []*Zone
	GetTrackForZone(zoneID string) (*Track, error)
	GetTrackForSpeaker(speakerID string) (*Track, error)
}

// Speaker speaker instance
type Speaker struct {
	ID          string
	DisplayName string
}

// Zone zone instance
type Zone struct {
	ID          string
	DisplayName string
	Speakers    []*Speaker
}

// Track represents a track
type Track struct {
	Artist  string
	Album   string
	Title   string
	Artwork []byte
}
