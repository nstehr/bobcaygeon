package sdp

type Origin struct {
	Username       string
	SessionID      string
	SessionVersion string
	NetType        string
	AddrType       string
	UnicastAddress string
}

type ConnectData struct {
	NetType           string
	AddrType          string
	ConnectionAddress string
}

type Timing struct {
	StartTime int
	Stopime   int
}

type MediaDescription struct {
	Media string
	Port  string // keeping string for now (parse later, fmt: <port>/<number of ports>)
	Proto string
	Fmt   string
}

type SessionDescription struct {
	Version          int
	Origin           Origin
	SessionName      string
	Information      string
	ConnectData      ConnectData
	Timing           Timing
	MediaDescription []MediaDescription
	Attributes       map[string]string
}

func NewSessionDescription() *SessionDescription {
	var mediaDescription []MediaDescription
	sdp := SessionDescription{Attributes: make(map[string]string), MediaDescription: mediaDescription}
	return &sdp
}
