package rtsp

import (
	"fmt"
	"log"
	"net"

	"github.com/nstehr/bobcaygeon/sdp"
)

const (
	readBuffer = 100000
)

type Decoder interface {
	Decode([]byte) []byte
}

type PortSet struct {
	Control int
	Timing  int
	Data    int
}

type Session struct {
	description *sdp.SessionDescription
	decoder     Decoder
	RemotePorts PortSet
	LocalPorts  PortSet
	dataConn    net.Conn // even though we have all ports, will only open up the data connection to start
}

func NewSession(description *sdp.SessionDescription, decoder Decoder) *Session {
	return &Session{description: description, decoder: decoder}
}

func (s *Session) Close() {
	s.dataConn.Close()
}

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
			s.decoder.Decode(packet)
			// once decoded, we can pass it along to be played
		}
	}()
	return nil
}
