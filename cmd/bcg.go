package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/hashicorp/memberlist"
	"github.com/nstehr/bobcaygeon/player"
	"github.com/nstehr/bobcaygeon/raop"

	petname "github.com/dustinkirkland/golang-petname"
)

var (
	name        = flag.String("name", "Bobcaygeon", "The name for the service.")
	port        = flag.Int("port", 5000, "Set the port the service is listening to.")
	verbose     = flag.Bool("verbose", false, "Verbose logging; logs requests and responses")
	clusterPort = flag.Int("clusterPort", 7676, "Port to listen for cluster events")
)

const (
	serviceType = "_bobcaygeon._tcp"
)

// EventDelegate handles the delgate functions from the memberlist
type EventDelegate struct {
	// keep a list of delegates so that we can have more than one
	// interested party for the membership events
	eventDelegates []memberlist.EventDelegate
}

// NodeMeta is metadata passed to other members about this node
type NodeMeta struct {
	RtspPort int
}

type Delegate struct{}

// NodeMeta is used to retrieve meta-data about the current node
// when broadcasting an alive message.
func (d Delegate) NodeMeta(limit int) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	metaData := NodeMeta{RtspPort: *port}

	err := enc.Encode(metaData)
	if err != nil {
		log.Println("Error encoding node metadata", err)
	}

	return buf.Bytes()
}

// GetBroadcasts is called when user data messages can be broadcast.
func (Delegate) GetBroadcasts(overhead, limit int) [][]byte {
	return make([][]byte, 0)
}

// LocalState is used for a TCP Push/Pull. This is sent to
// the remote side in addition to the membership information.
func (Delegate) LocalState(join bool) []byte {
	return make([]byte, 0)
}

// MergeRemoteState is invoked after a TCP Push/Pull.
func (Delegate) MergeRemoteState(buf []byte, join bool) {}

// NotifyMsg is called when a user-data message is received.
func (Delegate) NotifyMsg([]byte) {}

func newEventDelegate(d []memberlist.EventDelegate) *EventDelegate {
	return &EventDelegate{eventDelegates: d}
}

// NotifyJoin is invoked when a node is detected to have joined.
// The Node argument must not be modified.
func (ed *EventDelegate) NotifyJoin(node *memberlist.Node) {
	for _, delegate := range ed.eventDelegates {
		delegate.NotifyJoin(node)
	}
}

// NotifyLeave is invoked when a node is detected to have left.
// The Node argument must not be modified.
func (ed *EventDelegate) NotifyLeave(node *memberlist.Node) {
	for _, delegate := range ed.eventDelegates {
		delegate.NotifyLeave(node)
	}
}

// NotifyUpdate is invoked when a node is detected to have
// updated, usually involving the meta data. The Node argument
// must not be modified.
func (ed *EventDelegate) NotifyUpdate(node *memberlist.Node) {
	for _, delegate := range ed.eventDelegates {
		delegate.NotifyUpdate(node)
	}
}

func main() {
	flag.Parse()
	// generate a name for this node and initialize the distributed member list
	nodeName := petname.Generate(2, "-")
	log.Println(fmt.Sprintf("Starting node: %s", nodeName))
	c := memberlist.DefaultLocalConfig()
	c.Name = nodeName
	c.BindPort = *clusterPort
	c.AdvertisePort = *clusterPort
	c.Delegate = Delegate{}

	list, err := memberlist.Create(c)
	if err != nil {
		panic("Failed to create memberlist: " + err.Error())
	}

	var delegates []memberlist.EventDelegate
	// next we use mdns to try to find a cluster to join.
	// the curent leader (and receiving airplay server)
	// will be broadcasting a service to join
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}

	entries := make(chan *zeroconf.ServiceEntry)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(5))
	defer cancel()
	err = resolver.Browse(ctx, serviceType, "local", entries)
	if err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}
	log.Println("searching for cluster to join")
	var entry *zeroconf.ServiceEntry
	select {
	// this should be ok, since we only expect one service of the _bobcaygeon_ type to be found
	case entry = <-entries:
		log.Println("Found cluster to join")
	case <-ctx.Done():
		log.Println("cluster search timeout, no cluster to join")
	}
	// if the entry is nil, then we didn't find a cluster to join, so assume leadership
	if entry == nil {
		log.Println("starting cluster, I am now initial leader")
		log.Println("broadcasting my join info")
		// start broadcasting the service
		server, err := zeroconf.Register(nodeName, serviceType, "local.", *clusterPort, []string{"txtv=0", "lo=1", "la=2"}, nil)
		if err != nil {
			log.Println("Error starting zeroconf service", err)
		}
		// since we are the leader, we will start the airplay server to accept the packets
		// and eventually forward to other members
		player := player.NewForwardingPlayer()
		delegates = append(delegates, player)

		nd := newEventDelegate(delegates)
		c.Events = nd

		airplayServer := raop.NewAirplayServer(*port, *name, player)
		go airplayServer.Start(*verbose)
		defer airplayServer.Stop()
		defer server.Shutdown()
	} else {
		log.Println("Joining cluster")
		_, err = list.Join([]string{fmt.Sprintf("%s:%d", entry.AddrIPv4[0].String(), entry.Port)})
		if err != nil {
			panic("Failed to join cluster: " + err.Error())
		}
	}

	for _, member := range list.Members() {
		log.Println(fmt.Sprintf("Member: %s %s\n", member.Name, member.Addr))
		dec := gob.NewDecoder(bytes.NewReader(member.Meta))
		var meta NodeMeta
		dec.Decode(&meta)
		log.Println(meta)
	}

	// Clean exit.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sig:
		// Exit by user
		log.Println("Ctrl-c detected, shutting down")
	}

	log.Println("Goodbye.")
}
