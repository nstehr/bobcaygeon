package cluster

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/hashicorp/memberlist"
	"github.com/nstehr/bobcaygeon/raop"
	"github.com/nstehr/bobcaygeon/rtsp"
)

// ForwardingPlayer will forward data packets to member nodes
type ForwardingPlayer struct {
	sessions *sessionMap
}

type sessionMap struct {
	sync.RWMutex
	sessions map[string]*rtsp.Session
}

func newSessionMap() *sessionMap {
	return &sessionMap{sessions: make(map[string]*rtsp.Session)}
}

func (sm *sessionMap) addSession(name string, session *rtsp.Session) {
	sm.Lock()
	defer sm.Unlock()
	sm.sessions[name] = session
}

func (sm *sessionMap) removeSession(name string) {
	sm.Lock()
	defer sm.Unlock()
	delete(sm.sessions, name)
}

func (sm *sessionMap) getSessions() []*rtsp.Session {
	sm.RLock()
	defer sm.RUnlock()
	sessions := make([]*rtsp.Session, 0, len(sm.sessions))

	for _, value := range sm.sessions {
		sessions = append(sessions, value)
	}
	return sessions
}

// NewForwardingPlayer instantiates a new ForwardingPlayer
func NewForwardingPlayer() *ForwardingPlayer {
	return &ForwardingPlayer{sessions: newSessionMap()}
}

// NotifyJoin is invoked when a node is detected to have joined.
// The Node argument must not be modified.
func (p *ForwardingPlayer) NotifyJoin(node *memberlist.Node) {
	log.Println("Node Joined " + node.Name)
	dec := gob.NewDecoder(bytes.NewReader(node.Meta))
	var meta NodeMeta
	dec.Decode(&meta)
	go p.initSession(node.Name, node.Addr, meta.RtspPort)
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

// Play will play the packets received on the specified session
// and forward the packets on
func (p *ForwardingPlayer) Play(session *rtsp.Session) {
	//go player.PlayStream(session)

	go func() {
		for d := range session.DataChan {
			sessions := p.sessions.getSessions()
			for _, s := range sessions {
				s.DataChan <- d
			}
		}
	}()

}

func (p *ForwardingPlayer) initSession(nodeName string, ip net.IP, port int) {

	session, err := raop.EstablishSession(ip.String(), port)

	if err != nil {
		log.Println(fmt.Sprintf("Error connecting to RTSP server: %s:%d", ip.String(), port), err)
		return
	}

	log.Println(fmt.Sprintf("Session established for %s (%s:%d).", nodeName, ip.String(), port))

	session.StartSending()
	p.sessions.addSession(nodeName, session)

}
