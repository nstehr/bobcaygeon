package cluster

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/hajimehoshi/oto"
	"github.com/hashicorp/memberlist"
	"github.com/nstehr/bobcaygeon/player"
	"github.com/nstehr/bobcaygeon/raop"
	"github.com/nstehr/bobcaygeon/rtsp"
)

// ForwardingPlayer will forward data packets to member nodes
type ForwardingPlayer struct {
	volLock  sync.RWMutex
	volume   float64
	sessions *sessionMap
}

// represents what a client calling an RTSP
// server would want for a session; the actual
// session for data transfer, as well the port
// for making RTSP calls for control

//TODO: should this be promoted to the RTSP package and
// returned by raop.EstablishSession ??
type clientSession struct {
	*rtsp.Session
	rtspPort int
}

type sessionMap struct {
	sync.RWMutex
	sessions map[string]*clientSession
}

func newSessionMap() *sessionMap {
	return &sessionMap{sessions: make(map[string]*clientSession)}
}

func (sm *sessionMap) addSession(name string, session *clientSession) {
	sm.Lock()
	defer sm.Unlock()
	sm.sessions[name] = session
}

func (sm *sessionMap) removeSession(name string) {
	sm.Lock()
	defer sm.Unlock()
	delete(sm.sessions, name)
}

func (sm *sessionMap) getSessions() []*clientSession {
	sm.RLock()
	defer sm.RUnlock()
	sessions := make([]*clientSession, 0, len(sm.sessions))

	for _, value := range sm.sessions {
		sessions = append(sessions, value)
	}
	return sessions
}

// NewForwardingPlayer instantiates a new ForwardingPlayer
func NewForwardingPlayer() *ForwardingPlayer {
	return &ForwardingPlayer{sessions: newSessionMap(), volume: 1}
}

// NotifyJoin is invoked when a node is detected to have joined.
// The Node argument must not be modified.
func (p *ForwardingPlayer) NotifyJoin(node *memberlist.Node) {
	log.Println("Node Joined " + node.Name)
	meta := DecodeNodeMeta(node.Meta)
	if meta.NodeType == Music {
	    go p.initSession(node.Name, node.Addr, meta.RtspPort)
	}
	
}

// NotifyLeave is invoked when a node is detected to have left.
// The Node argument must not be modified.
func (p *ForwardingPlayer) NotifyLeave(node *memberlist.Node) {
	log.Println("Node Left" + node.Name)
	p.sessions.removeSession(node.Name)
}

// NotifyUpdate is invoked when a node is detected to have
// updated, usually involving the meta data. The Node argument
// must not be modified.
func (*ForwardingPlayer) NotifyUpdate(node *memberlist.Node) {
	log.Println("Node updated" + node.Name)

}

// SetVolume accepts a float between 0 (mute) and 1 (full volume)
func (p *ForwardingPlayer) SetVolume(volume float64) {
	p.volLock.Lock()
	defer p.volLock.Unlock()
	p.volume = volume
	// as a first pass all down stream clients will have the same
	// volume; adjusting the volume of the forwarding player will
	// forward the volume settings
	go func() {
		for _, s := range p.sessions.getSessions() {
			client, err := rtsp.NewClient(s.RemotePorts.Address, s.rtspPort)
			if err != nil {
				log.Println("Error establishing RTSP connection", err)
				continue
			}
			req := rtsp.NewRequest()
			req.Method = rtsp.Set_Parameter
			sessionID := strconv.FormatInt(time.Now().Unix(), 10)
			localAddress := client.LocalAddress()
			req.RequestURI = fmt.Sprintf("rtsp://%s/%s", localAddress, sessionID)
			req.Headers["Content-Type"] = "text/parameters"
			body := fmt.Sprintf("volume: %f", prepareVolume(volume))
			req.Body = []byte(body)
			client.Send(req)
		}
	}()
}

// Play will play the packets received on the specified session
// and forward the packets on
func (p *ForwardingPlayer) Play(session *rtsp.Session) {
	// TODO: refactor so both we don't need to init oto player here too
	ap, err := oto.NewPlayer(44100, 2, 2, 10000)
	if err != nil {
		log.Println("error initializing player", err)
		return
	}
	decoder := player.GetCodec(session)

	go func() {
		for d := range session.DataChan {
			p.volLock.RLock()
			vol := p.volume
			p.volLock.RUnlock()
			// will play the audio
			decoded, err := decoder(d)
			if err != nil {
				log.Println("Problem decoding packet")
			}
			ap.Write(player.AdjustAudio(decoded, vol))

			// will forward the audio to other clients
			go func(pkt []byte) {
				sessions := p.sessions.getSessions()
				for _, s := range sessions {
					s.DataChan <- pkt
				}
			}(d)

		}
	}()

}

func (p *ForwardingPlayer) initSession(nodeName string, ip net.IP, port int) {

	session, err := raop.EstablishSession(ip.String(), port)

	// do retry if we can't establish a session.  We may get
	// the node join event before the node as fully started
	// the rtsp server, so we try a few times
	for i := 0; i < 3; i++ {
		if session != nil {
			break
		}
		if err != nil {
			log.Printf("Error connecting to RTSP server: %s:%d. Retrying\n", ip.String(), port)
		}
		time.Sleep(3 * time.Second)
		session, err = raop.EstablishSession(ip.String(), port)
	}

	if err != nil {
		log.Println(fmt.Sprintf("Error connecting to RTSP server: %s:%d", ip.String(), port), err)
		return
	}

	log.Printf("Session established for %s (%s:%d).\n", nodeName, ip.String(), port)

	session.StartSending()
	cSession := &clientSession{session, port}
	p.sessions.addSession(nodeName, cSession)

}

// airplay server will apply a normalization,
// we have the raw volume on a scale of 0 to 1,
// so we build the proper format
func prepareVolume(vol float64) float64 {
	// 0 volume means mute, airplay servers understands
	// mute as -144
	if vol == 0 {
		return -144
	}

	// 1 is full volume, for airplay servers
	// this means 0, as in 0 volume adjustment needed
	if vol == 1 {
		return 0
	}

	// the remaining values needs to be between -30 and 0,
	adjusted := (vol * 30) - 30

	return adjusted
}
