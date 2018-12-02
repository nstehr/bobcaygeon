package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/hashicorp/memberlist"
	"github.com/nstehr/bobcaygeon/api"
	"github.com/nstehr/bobcaygeon/cluster"
	"github.com/nstehr/bobcaygeon/player"
	"github.com/nstehr/bobcaygeon/raop"
	"google.golang.org/grpc"

	petname "github.com/dustinkirkland/golang-petname"
)

var (
	name          = flag.String("name", "Bobcaygeon", "The name for the service.")
	port          = flag.Int("port", 5000, "Set the port the service is listening to.")
	dataPort      = flag.Int("dataPort", 6000, "The port to listen for streaming data")
	verbose       = flag.Bool("verbose", false, "Verbose logging; logs requests and responses")
	clusterPort   = flag.Int("clusterPort", 7676, "Port to listen for cluster events")
	apiServerPort = flag.Int("apiServerPort", 7777, "Port to listen for API server")
)

const (
	serviceType = "_bobcaygeon._tcp"
)

func main() {
	flag.Parse()
	// generate a name for this node and initialize the distributed member list
	nodeName := petname.Generate(2, "-")
	log.Printf("Starting node: %s\n", nodeName)
	metaData := &cluster.NodeMeta{RtspPort: *port, NodeType: cluster.Music}
	c := memberlist.DefaultLocalConfig()
	c.Name = nodeName
	c.BindPort = *clusterPort
	c.AdvertisePort = *clusterPort
	c.Delegate = cluster.Delegate{MetaData: metaData}

	list, err := memberlist.Create(c)
	if err != nil {
		panic("Failed to create memberlist: " + err.Error())
	}

	var delegates []memberlist.EventDelegate
	// we pass a player to the airplay server.  In our case
	// it will be a forwarding player or a regular player
	var streamPlayer player.Player
	// we use our airplay server to handle both scenarios
	// the "leader" and the "follower".  If we are a follower
	// we don't advertise as an airplay server
	advertise := false
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
	foundEntry := make(chan *zeroconf.ServiceEntry)
	// what we do is spin of a goroutine that will process the entries registered in
	// mDNS for our service.  As soon as we detect there is one with an IP4 address
	// we send it off and cancel to stop the searching.
	// there is an issue, https://github.com/grandcat/zeroconf/issues/27 where we
	// could get an entry back without an IP4 addr, it will come in later as an update
	// so we wait until we find the addr, or timeout
	go func(results <-chan *zeroconf.ServiceEntry, foundEntry chan *zeroconf.ServiceEntry) {
		for e := range results {
			if (len(e.AddrIPv4)) > 0 {
				foundEntry <- e
				cancel()
			}
		}
	}(entries, foundEntry)

	select {
	// this should be ok, since we only expect one service of the _bobcaygeon_ type to be found
	case entry = <-foundEntry:
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
		forwardingPlayer := cluster.NewForwardingPlayer()
		streamPlayer = forwardingPlayer
		delegates = append(delegates, forwardingPlayer)

		nd := cluster.NewEventDelegate(delegates)
		c.Events = nd

		defer server.Shutdown()
		advertise = true
	} else {
		log.Println("Joining cluster")
		_, err = list.Join([]string{fmt.Sprintf("%s:%d", entry.AddrIPv4[0].String(), entry.Port)})
		if err != nil {
			panic("Failed to join cluster: " + err.Error())
		}
		streamPlayer = player.NewLocalPlayer()
	}

	airplayServer := raop.NewAirplayServer(*port, *dataPort, *name, streamPlayer)
	go airplayServer.Start(*verbose, advertise)
	defer airplayServer.Stop()

	// start the API server
	go startAPIServer(*apiServerPort, airplayServer)

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

func startAPIServer(apiServerPort int, airplayServer *raop.AirplayServer) {
	// create a listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", apiServerPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	// create a server instance
	s := api.NewServer(airplayServer)
	// create a gRPC server object
	grpcServer := grpc.NewServer()
	// attach the Ping service to the server
	api.RegisterManagementServer(grpcServer, s)
	log.Printf("Starting API server on port: %d", apiServerPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}
