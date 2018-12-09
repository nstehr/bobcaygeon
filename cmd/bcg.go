package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/grandcat/zeroconf"
	"github.com/hashicorp/memberlist"
	"github.com/nstehr/bobcaygeon/api"
	"github.com/nstehr/bobcaygeon/cluster"
	"github.com/nstehr/bobcaygeon/player"
	"github.com/nstehr/bobcaygeon/raop"
	"github.com/pelletier/go-toml"
	"google.golang.org/grpc"

	petname "github.com/dustinkirkland/golang-petname"
)

var (
	verbose    = flag.Bool("verbose", false, "Verbose logging; logs requests and responses")
	configPath = flag.String("config", "bcg.toml", "Path to the config file for the node")
)

const (
	serviceType = "_bobcaygeon._tcp"
)

type rtspConfig struct {
	Name     string `toml:"name"`
	Port     int    `toml:"port"`
	DataPort int    `toml:"data-port"`
}

type nodeConfig struct {
	APIPort     int    `toml:"api-port"`
	ClusterPort int    `toml:"cluster-port"`
	Name        string `toml:"name"`
}

type conf struct {
	Node nodeConfig `toml:"node"`
	Rtsp rtspConfig `toml:"rtsp"`
}

func main() {
	flag.Parse()
	// generate a name for this node and initialize the distributed member list
	configFile, err := ioutil.ReadFile(*configPath)
	// if we os.Open returns an error then handle it
	if err != nil {
		log.Fatal("Could not open config file: ", err)
	}

	config := conf{}
	err = toml.Unmarshal(configFile, &config)
	if err != nil {
		log.Fatal("Could parse open config file: ", err)
	}

	if config.Node.Name == "" {
		log.Println("Generating node name")
		config.Node.Name = petname.Generate(2, "-")
		updated, err := toml.Marshal(config)
		if err != nil {
			log.Fatal("Could not update config")
		}
		ioutil.WriteFile(*configPath, updated, 0644)
	}
	nodeName := config.Node.Name
	log.Printf("Starting node: %s\n", nodeName)
	metaData := &cluster.NodeMeta{RtspPort: config.Rtsp.Port, NodeType: cluster.Music}
	c := memberlist.DefaultLocalConfig()
	c.Name = nodeName
	c.BindPort = config.Node.ClusterPort
	c.AdvertisePort = config.Node.ClusterPort
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
	entry := cluster.SearchForCluster()

	// if the entry is nil, then we didn't find a cluster to join, so assume leadership
	if entry == nil {
		log.Println("starting cluster, I am now initial leader")
		log.Println("broadcasting my join info")
		// start broadcasting the service
		server, err := zeroconf.Register(nodeName, serviceType, "local.", config.Node.ClusterPort, []string{"txtv=0", "lo=1", "la=2"}, nil)
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

	airplayServer := raop.NewAirplayServer(config.Rtsp.Port, config.Rtsp.DataPort, config.Rtsp.Name, streamPlayer)
	go airplayServer.Start(*verbose, advertise)
	defer airplayServer.Stop()

	// start the API server
	go startAPIServer(config.Node.APIPort, airplayServer)

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
	api.RegisterAirPlayManagementServer(grpcServer, s)
	log.Printf("Starting API server on port: %d", apiServerPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}
