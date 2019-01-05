package service

// MgmtService interface for handling management capabilities
type MgmtService interface {
	GetSpeakers() []*Speaker
	SetDisplayName(ID string, displayName string) error
}

// Speaker speaker instance
type Speaker struct {
	ID          string
	DisplayName string
}
