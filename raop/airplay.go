package raop

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/grandcat/zeroconf"
	"github.com/nstehr/bobcaygeon/rtsp"
)

// sets up the properties needed to make us discoverable as a airtunes service
// https://github.com/fgp/AirReceiver/blob/STABLE_1_X/src/main/java/org/phlo/AirReceiver/AirReceiver.java#L88
// https://nto.github.io/AirPlay.html#audio
const airTunesServiceType = "_raop._tcp"
const domain = "local."

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
	rtspServer.Start(verbose)

}

func handleOptions(req *rtsp.Request, resp *rtsp.Response, localAddress string, remoteAddress string) {
	log.Println("Handling OPTIONS")
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
