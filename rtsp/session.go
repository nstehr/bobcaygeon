package rtsp

import (
	"fmt"
	"log"
	"net"

	"github.com/nstehr/bobcaygeon/sdp"
)

const (
	readBuffer = 1024 * 16
)

// Decrypter decrypts a received packet
type Decrypter interface {
	Decode([]byte) ([]byte, error)
}

// PortSet wraps the ports needed for an RTSP stream
type PortSet struct {
	Address string
	Control int
	Timing  int
	Data    int
}

// Session a streaming session
type Session struct {
	Description *sdp.SessionDescription
	decrypter   Decrypter
	RemotePorts PortSet
	LocalPorts  PortSet
	dataConn    net.Conn // even though we have all ports, will only open up the data connection to start
	DataChan    chan []byte
}

// NewSession instantiates a new Session
func NewSession(description *sdp.SessionDescription, decrypter Decrypter) *Session {
	return &Session{Description: description, decrypter: decrypter, DataChan: make(chan []byte, 1000)}
}

// Close closes a session
func (s *Session) Close() {
	s.dataConn.Close()
	close(s.DataChan)
}

// StartReceiving starts a session for listening for data
func (s *Session) StartReceiving() error {
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
			d := packet
			if s.decrypter != nil {
				d, err = s.decrypter.Decode(packet)
			}
			if err != nil {
				log.Println("Problem decoding packet", err)
				continue
			}
			// once decoded, we can pass it along to be played
			send := make([]byte, len(d))
			copy(send, d)
			s.DataChan <- send
		}
	}()
	return nil
}

// StartSending starts a session for sending data
func (s *Session) StartSending() error {

	conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", s.RemotePorts.Address, s.RemotePorts.Data))
	if err != nil {
		return err
	}
	// keep track of the actual connection so we close it later
	s.dataConn = conn
	// start listening for audio data
	log.Println("Session started.  Will start sending packets")
	go func() {
		for pkt := range s.DataChan {
			conn.Write(pkt)
		}
	}()
	return nil
}
