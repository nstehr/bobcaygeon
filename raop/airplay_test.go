package raop

import (
	"fmt"
	"testing"

	"github.com/nstehr/bobcaygeon/sdp"

	"github.com/nstehr/bobcaygeon/rtsp"
)

type FakePlayer struct{}

func (FakePlayer) Play(session *rtsp.Session) {}

func TestHandleOptions(t *testing.T) {
	req := rtsp.NewRequest()
	req.Headers["Apple-Challenge"] = "gY3cmhtK9LnECNUlXFb0qg=="
	resp := rtsp.NewResponse()
	localAddress := "192.168.0.15"
	remoteAddress := "10.0.0.0"
	handleOptions(req, resp, localAddress, remoteAddress)
	if resp.Status != rtsp.Ok {
		t.Error(fmt.Sprintf("Expected: %s\r\n Got: %s", rtsp.Ok.String(), resp.Status.String()))
	}
	_, ok := resp.Headers["Public"]
	if !ok {
		t.Error(fmt.Sprintf("Expected to have Public header"))
	}
	// we don't actually care about the generated value (that is tested in another test)
	_, ok = resp.Headers["Apple-Response"]
	if !ok {
		t.Error(fmt.Sprintf("Expected to have Apple-Response header"))
	}
}

func TestHandleSetup(t *testing.T) {
	a := NewAirplayServer(444, 333, "Test", FakePlayer{})
	s := rtsp.NewSession(sdp.NewSessionDescription(), nil)
	a.session = s
	req := rtsp.NewRequest()
	req.Headers["Transport"] = "RTP/AVP/UDP;unicast;interleaved=0-1;mode=record;control_port=8888;timing_port=8889"
	resp := rtsp.NewResponse()
	localAddress := "192.168.0.15"
	remoteAddress := "10.0.0.0"
	a.handleSetup(req, resp, localAddress, remoteAddress)
	if resp.Status != rtsp.Ok {
		t.Error(fmt.Sprintf("Expected: %s\r\n Got: %s", rtsp.Ok.String(), resp.Status.String()))
	}
	if a.session.RemotePorts.Address != remoteAddress {
		t.Error(fmt.Sprintf("Expected: %s\r\n Got: %s", remoteAddress, a.session.RemotePorts.Address))
	}
	if a.session.RemotePorts.Control != 8888 {
		t.Error(fmt.Sprintf("Expected: %d\r\n Got: %d", 8888, a.session.RemotePorts.Control))
	}
	if a.session.RemotePorts.Timing != 8889 {
		t.Error(fmt.Sprintf("Expected: %d\r\n Got: %d", 8889, a.session.RemotePorts.Timing))
	}
	_, ok := resp.Headers["Transport"]
	if !ok {
		t.Error(fmt.Sprintf("Expected to have Transport header"))
	}
	val, ok := resp.Headers["Session"]
	if !ok {
		t.Error(fmt.Sprintf("Expected to have Session header"))
	}
	if val != "1" {
		t.Error(fmt.Sprintf("Expected: %s\r\n Got: %s", "1", val))
	}
	val, ok = resp.Headers["Audio-Jack-Status"]
	if !ok {
		t.Error(fmt.Sprintf("Expected to have Transport header"))
	}
	if val != "connected" {
		t.Error(fmt.Sprintf("Expected: %s\r\n Got: %s", "connected", val))
	}
}
