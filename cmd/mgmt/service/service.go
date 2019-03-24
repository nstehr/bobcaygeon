package service

// MgmtService interface for handling management capabilities
type MgmtService interface {
	GetSpeakers() []*Speaker
	SetDisplayName(ID string, displayName string) error
	CreateZone(displayName string, speakerIDs []string) (string, error)
	GetZones() []*Zone
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
