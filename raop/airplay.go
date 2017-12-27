package raop

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/grandcat/zeroconf"
	"github.com/nstehr/bobcaygeon/rtsp"
	"github.com/nstehr/bobcaygeon/sdp"
)

// sets up the properties needed to make us discoverable as a airtunes service
// https://github.com/fgp/AirReceiver/blob/STABLE_1_X/src/main/java/org/phlo/AirReceiver/AirReceiver.java#L88
// https://nto.github.io/AirPlay.html#audio
const (
	airTunesServiceType = "_raop._tcp"
	domain              = "local."
	localDataPort       = 6000
	localTimingPort     = 6002
	localControlPort    = 6001
)

var airtunesServiceProperties = []string{"txtvers=1",
	"tp=UDP",
	"ch=2",
	"ss=16",
	"sr=44100",
	"pw=false",
	"sm=false",
	"sv=false",
	"ek=1",
	"et=0,1",
	"cn=0,1",
	"vn=3"}

type AirplayServer struct {
	port          int
	name          string
	rtspServer    *rtsp.RtspServer
	zerconfServer *zeroconf.Server
	session       *rtsp.Session
}

func NewAirplayServer(port int, name string) *AirplayServer {
	as := AirplayServer{port: port, name: name}
	return &as
}

func (a *AirplayServer) Start(verbose bool) {
	rtspServer := rtsp.NewRtspServer(a.port)
	// as per the protocol, the mac address makes up part of the service name
	macAddr := getMacAddr().String()
	macAddr = strings.Replace(macAddr, ":", "", -1)

	serviceName := fmt.Sprintf("%s@%s", macAddr, a.name)

	server, err := zeroconf.Register(serviceName, airTunesServiceType, domain, a.port, airtunesServiceProperties, nil)
	if err != nil {
		log.Fatal("couldn't start zeroconf: ", err)
	}

	log.Println("Published service:")
	log.Println("- Name:", serviceName)
	log.Println("- Type:", airTunesServiceType)
	log.Println("- Domain:", domain)
	log.Println("- Port:", a.port)

	a.zerconfServer = server
	a.rtspServer = rtspServer

	rtspServer.AddHandler(rtsp.Options, handleOptions)
	rtspServer.AddHandler(rtsp.Announce, a.handleAnnounce)
	rtspServer.AddHandler(rtsp.Setup, a.handleSetup)
	rtspServer.AddHandler(rtsp.Record, a.handleRecord)
	rtspServer.AddHandler(rtsp.Set_Parameter, handlSetParameter)
	rtspServer.Start(verbose)

}

func handleOptions(req *rtsp.Request, resp *rtsp.Response, localAddress string, remoteAddress string) {
	resp.Status = rtsp.Ok
	resp.Headers["Public"] = strings.Join(rtsp.GetMethods(), " ")
	appleChallenge, exists := req.Headers["Apple-Challenge"]
	if !exists {
		return
	}
	log.Println(fmt.Sprintf("Apple Challenge detected: %s", appleChallenge))
	challengResponse, err := generateChallengeResponse(appleChallenge, getMacAddr(), localAddress)
	if err != nil {
		log.Println("Error generating challenge response: ", err.Error())
	}
	resp.Headers["Apple-Response"] = challengResponse

}

func (a *AirplayServer) handleAnnounce(req *rtsp.Request, resp *rtsp.Response, localAddress string, remoteAddress string) {
	if req.Headers["Content-Type"] == "application/sdp" {
		description, err := sdp.Parse(strings.NewReader(req.Body))
		if err != nil {
			log.Println("error parsing SDP payload: ", err)
			resp.Status = rtsp.BadRequest
		}

		// right now, we only maintain one audio session, so close any existing one
		if a.session != nil {
			a.session.Close()
		}
		var decoder rtsp.Decoder
		rtpmap := description.Attributes["rtpmap"]

		if strings.Contains(rtpmap, "AppleLossless") {
			aesKey := description.Attributes["rsaaeskey"]
			aesIv := description.Attributes["aesiv"]
			decoder = NewDecryptingAlacDecoder(aesKey, aesIv)
		}
		a.session = rtsp.NewSession(description, decoder)
	}
	resp.Status = rtsp.Ok
}

func (a *AirplayServer) handleSetup(req *rtsp.Request, resp *rtsp.Response, localAddress string, remoteAddress string) {
	transport := req.Headers["Transport"]
	transportParts := strings.Split(transport, ";")
	var controlPort int
	var timingPort int
	for _, part := range transportParts {
		if strings.Contains(part, "control_port") {
			controlPort, _ = strconv.Atoi(strings.Split(part, "=")[1])
		}
		if strings.Contains(part, "timing_port") {
			timingPort, _ = strconv.Atoi(strings.Split(part, "=")[1])
		}
	}
	a.session.RemotePorts.Control = controlPort
	a.session.RemotePorts.Timing = timingPort

	// hardcode our listening ports for now
	a.session.LocalPorts.Control = localControlPort
	a.session.LocalPorts.Timing = localTimingPort
	a.session.LocalPorts.Data = localDataPort

	resp.Headers["Transport"] = fmt.Sprintf("RTP/AVP/UDP;unicast;mode=record;server_port=%d;control_port=%d;timing_port=%d", localDataPort, localControlPort, localTimingPort)
	resp.Headers["Session"] = "1"
	resp.Headers["Audio-Jack-Status"] = "connected"
	resp.Status = rtsp.Ok
}

func (a *AirplayServer) handleRecord(req *rtsp.Request, resp *rtsp.Response, localAddress string, remoteAddress string) {
	err := a.session.Start()
	if err != nil {
		log.Println("could not start streaming session: ", err)
		resp.Status = rtsp.InternalServerError
	}
	resp.Headers["Audio-Latency"] = "2205"
	resp.Status = rtsp.Ok

}

func handlSetParameter(req *rtsp.Request, resp *rtsp.Response, localAddress string, remoteAddress string) {
	resp.Status = rtsp.Ok
}

func (a *AirplayServer) Stop() {
	a.rtspServer.Stop()
	a.zerconfServer.Shutdown()
}

// getMacAddr gets the MAC hardware
// address of the host machine: https://gist.github.com/rucuriousyet/ab2ab3dc1a339de612e162512be39283
func getMacAddr() (addr net.HardwareAddr) {
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, i := range interfaces {
			if i.Flags&net.FlagUp != 0 && bytes.Compare(i.HardwareAddr, nil) != 0 {
				// Don't use random as we have a real address
				addr = i.HardwareAddr
				break
			}
		}
	}
	return
}
