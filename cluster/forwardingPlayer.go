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
	volLock      sync.RWMutex
	trackLock    sync.RWMutex
	volume       float64
	sessions     *sessionMap
	ap           *oto.Player
	currentTrack player.Track
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

func (sm *sessionMap) removeAll() {
	sm.Lock()
	defer sm.Unlock()
	sm.sessions = make(map[string]*clientSession)
}

func (sm *sessionMap) sessionExists(name string) bool {
	sm.RLock()
	defer sm.RUnlock()
	_, present := sm.sessions[name]
	return present
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
func NewForwardingPlayer() (*ForwardingPlayer, error) {
	ap, err := oto.NewPlayer(44100, 2, 2, 10000)
	if err != nil {
		return nil, err
	}
	return &ForwardingPlayer{sessions: newSessionMap(), volume: 1, ap: ap}, nil
}

// NotifyJoin is invoked when a node is detected to have joined.
// The Node argument must not be modified.
func (p *ForwardingPlayer) NotifyJoin(node *memberlist.Node) {
	log.Println("Node Joined " + node.Name)
	p.AddSessionForNode(node)

}

// NotifyLeave is invoked when a node is detected to have left.
// The Node argument must not be modified.
func (p *ForwardingPlayer) NotifyLeave(node *memberlist.Node) {
	log.Println("Node Left" + node.Name)
	p.RemoveSessionForNode(node)
}

// NotifyUpdate is invoked when a node is detected to have
// updated, usually involving the meta data. The Node argument
// must not be modified.
func (*ForwardingPlayer) NotifyUpdate(node *memberlist.Node) {
	log.Println("Node updated" + node.Name)

}

// AddSessionForNode will create a session to the given node
func (p *ForwardingPlayer) AddSessionForNode(node *memberlist.Node) {
	log.Println("Adding session for node: " + node.Name)
	meta := DecodeNodeMeta(node.Meta)
	if meta.NodeType == Music {
		go p.initSession(node.Name, node.Addr, meta.RtspPort)
	}
}

// RemoveSessionForNode will remove the session for the given node
func (p *ForwardingPlayer) RemoveSessionForNode(node *memberlist.Node) {
	log.Println("Removing session for node: " + node.Name)
	meta := DecodeNodeMeta(node.Meta)
	// TODO: should probably explicitly close the session.
	// next connection to node will do that, so it should be ok
	// for now
	if meta.NodeType == Music {
		p.sessions.removeSession(node.Name)
	}
}

// RemoveAllSessions will remove all the active forwarding sessions
func (p *ForwardingPlayer) RemoveAllSessions() {
	log.Println("Removing all forwarding sessions")
	p.sessions.removeAll()
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
	decoder := player.GetCodec(session)

	go func(dc player.CodecHandler) {
		for d := range session.DataChan {
			p.volLock.RLock()
			vol := p.volume
			p.volLock.RUnlock()
			func() {
				defer func() {
					if err := recover(); err != nil {
						fmt.Println(err)
					}
				}()
				// will play the audio
				decoded, err := dc(d)
				if err != nil {
					log.Println("Problem decoding packet")
				}
				p.ap.Write(player.AdjustAudio(decoded, vol))

				// will forward the audio to other clients
				go func(pkt []byte) {
					sessions := p.sessions.getSessions()
					for _, s := range sessions {
						s.DataChan <- pkt
					}
				}(d)
			}()
		}
		log.Println("Session data sending closed")
	}(decoder)

}

// SetTrack sets the track for the player
func (p *ForwardingPlayer) SetTrack(album string, artist string, title string) {
	p.trackLock.Lock()
	defer p.trackLock.Unlock()
	p.currentTrack.Album = album
	p.currentTrack.Artist = artist
	p.currentTrack.Title = title
	// forward the track data downstream
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
			req.Headers["Content-Type"] = "application/x-dmap-tagged"
			input := make(map[string]interface{})

			input["daap.songalbum"] = album
			input["dmap.itemname"] = title
			input["daap.songartist"] = artist
			body, err := raop.EncodeDaap(input)
			if err != nil {
				log.Println("Error encoding song information", err)
				continue
			}
			req.Body = body
			client.Send(req)
		}
	}()
}

// SetAlbumArt sets the album art for the player
func (p *ForwardingPlayer) SetAlbumArt(artwork []byte) {
	p.trackLock.Lock()
	defer p.trackLock.Unlock()
	p.currentTrack.Artwork = artwork
	// forward the album art downstream
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
			req.Headers["Content-Type"] = "image/jpeg"

			req.Body = artwork
			client.Send(req)
		}
	}()
}

// GetTrack returns the track
func (p *ForwardingPlayer) GetTrack() player.Track {
	p.trackLock.RLock()
	defer p.trackLock.RUnlock()
	return p.currentTrack
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
