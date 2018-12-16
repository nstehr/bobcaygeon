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

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/grandcat/zeroconf"
	"github.com/hashicorp/memberlist"
	"github.com/nstehr/bobcaygeon/cluster"
	"github.com/nstehr/bobcaygeon/cmd/mgmt/api"
	toml "github.com/pelletier/go-toml"
	"google.golang.org/grpc"
)

var (
	configPath = flag.String("config", "bcg-mgmt.toml", "Path to the config file for the node")
)

type nodeConfig struct {
	APIPort     int    `toml:"api-port"`
	ClusterPort int    `toml:"cluster-port"`
	Name        string `toml:"name"`
}

type conf struct {
	Node nodeConfig `toml:"node"`
}

func main() {
	flag.Parse()
	log.Println(*configPath)
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
	log.Printf("Starting management API node: %s\n", nodeName)
	metaData := &cluster.NodeMeta{NodeType: cluster.Mgmt, APIPort: config.Node.APIPort}
	c := memberlist.DefaultLocalConfig()
	c.Name = nodeName
	c.BindPort = config.Node.ClusterPort
	c.AdvertisePort = config.Node.ClusterPort
	c.Delegate = cluster.Delegate{MetaData: metaData}

	list, err := memberlist.Create(c)

	var entry *zeroconf.ServiceEntry
	found := false

	// since we are a management node, we are an 'add on' so we will loop
	// until we know that there is atleast one bcg music playing node
	for found != true {
		entry = cluster.SearchForCluster()
		if entry != nil {
			found = true
		}
	}

	log.Println("Joining cluster")
	_, err = list.Join([]string{fmt.Sprintf("%s:%d", entry.AddrIPv4[0].String(), entry.Port)})
	if err != nil {
		panic("Failed to join cluster: " + err.Error())
	}
	go startAPIServer(config.Node.APIPort, list)
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

func startAPIServer(apiServerPort int, list *memberlist.Memberlist) {
	// create a listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", apiServerPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	// create a server instance
	s := api.NewServer(list)
	// create a gRPC server object
	grpcServer := grpc.NewServer()
	api.RegisterBobcaygeonManagementServer(grpcServer, s)
	log.Printf("Starting API server on port: %d", apiServerPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}
