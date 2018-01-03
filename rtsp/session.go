package rtsp

import (
	"fmt"
	"log"
	"net"

	"github.com/hajimehoshi/oto"
	"github.com/nstehr/bobcaygeon/sdp"
)

const (
	readBuffer = 1024 * 16
)

// Decoder decodes a received packet
type Decoder interface {
	Decode([]byte) ([]byte, error)
}

// PortSet wraps the ports needed for an RTSP stream
type PortSet struct {
	Control int
	Timing  int
	Data    int
}

// Session a streaming session
type Session struct {
	description *sdp.SessionDescription
	decoder     Decoder
	RemotePorts PortSet
	LocalPorts  PortSet
	dataConn    net.Conn // even though we have all ports, will only open up the data connection to start
}

// NewSession instantiates a new Session
func NewSession(description *sdp.SessionDescription, decoder Decoder) *Session {
	return &Session{description: description, decoder: decoder}
}

// Close closes a session
func (s *Session) Close() {
	s.dataConn.Close()
}

// Start starts a session for listening for data
func (s *Session) Start() error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", s.LocalPorts.Data))
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	// keep track of the actual connection so we close it later
	s.dataConn = conn
	// start listening for audio data
	log.Println("Session started.  Listening for audio packets")
	testChan := make(chan []byte, 1000)
	go func() {
		buf := make([]byte, readBuffer)
		for {
			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				log.Println("Error reading data from socket: " + err.Error())
				return
			}
			packet := buf[:n]
			// send the data to the decoder
			d, err := s.decoder.Decode(packet)
			if err != nil {
				log.Println("Problem decoding packet", err)
				continue
			}
			// once decoded, we can pass it along to be played
			testChan <- d
		}
	}()
	// TODO: abstract this out, maybe refactor this session/player logic all together
	go func() {
		p, err := oto.NewPlayer(44100, 2, 2, 10000)
		if err != nil {
			log.Println("error initializing player", err)
			return
		}
		for d := range testChan {
			p.Write(d)
		}
	}()
	return nil
}
