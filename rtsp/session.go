package rtsp

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

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
	dataConn    net.Conn
	DataChan    chan []byte
	stopChan    chan (struct{})
}

// NewSession instantiates a new Session
func NewSession(description *sdp.SessionDescription, decrypter Decrypter) *Session {
	return &Session{Description: description, decrypter: decrypter, DataChan: make(chan []byte, 1000)}
}

// InitReceive initializes the session to for receiving
func (s *Session) InitReceive() error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", 0))
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	// keep track of the actual connection so we close it later
	s.dataConn = conn
	localAddr := strings.Split(conn.LocalAddr().String(), ":")
	port := localAddr[len(localAddr)-1]
	s.LocalPorts.Data, _ = strconv.Atoi(port)
	return nil
}

// Close closes a session
func (s *Session) Close(closeDone chan struct{}) {
	log.Println("closing session")
	s.stopChan = closeDone
	if s.dataConn != nil {
		s.dataConn.Close()
	} else {
		log.Println("Currently no data connection...")
		s.stopChan <- struct{}{}
	}
}

// StartReceiving starts a session for listening for data
func (s *Session) StartReceiving() error {
	// start listening for audio data
	log.Println("Session started.  Listening for audio packets")
	go func(conn *net.UDPConn) {
		buf := make([]byte, readBuffer)
		for {
			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				log.Println("Error reading data from socket: " + err.Error())
				close(s.DataChan)
				conn = nil
				break
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
		log.Println("Signalling Session is closed")
		if s.stopChan != nil {
			s.stopChan <- struct{}{}
		}
	}(s.dataConn.(*net.UDPConn))
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
