package player

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net"

	"github.com/hashicorp/memberlist"
	"github.com/nstehr/bobcaygeon/cluster"
	"github.com/nstehr/bobcaygeon/rtsp"
)

// ForwardingPlayer will forward data packets to member nodes
type ForwardingPlayer struct{}

// NewForwardingPlayer instantiates a new ForwardingPlayer
func NewForwardingPlayer() *ForwardingPlayer {
	return &ForwardingPlayer{}
}

// NotifyJoin is invoked when a node is detected to have joined.
// The Node argument must not be modified.
func (*ForwardingPlayer) NotifyJoin(node *memberlist.Node) {
	log.Println("Node Joined " + node.Name)
	dec := gob.NewDecoder(bytes.NewReader(node.Meta))
	var meta cluster.NodeMeta
	dec.Decode(&meta)
	go handshake(node.Addr, meta.RtspPort)
}

// NotifyLeave is invoked when a node is detected to have left.
// The Node argument must not be modified.
func (*ForwardingPlayer) NotifyLeave(node *memberlist.Node) {
	log.Println("Node Left" + node.Name)
}

// NotifyUpdate is invoked when a node is detected to have
// updated, usually involving the meta data. The Node argument
// must not be modified.
func (*ForwardingPlayer) NotifyUpdate(node *memberlist.Node) {
	log.Println("Node updated" + node.Name)

}

// Play will play the packets received on the specified session
// and forward the packets on
func (*ForwardingPlayer) Play(session *rtsp.Session) {
	go play(session)
}

func handshake(ip net.IP, port int) {
	client, err := rtsp.NewClient(ip.String(), port)
	if err != nil {
		log.Println(fmt.Sprintf("Error connecting to RTSP server: %s:%d", ip.String(), port), err)
	}
	req := rtsp.NewRequest()
	req.Method = rtsp.Options
	req.RequestURI = "*"
	resp, err := client.Send(req)
	if err != nil {
		log.Println("Error sending message to server", err)
	}
	log.Println(resp.String())
}
