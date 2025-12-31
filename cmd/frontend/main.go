package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/grandcat/zeroconf"
	"github.com/hashicorp/memberlist"
	"github.com/nstehr/bobcaygeon/cluster"
	"github.com/nstehr/bobcaygeon/cmd/frontend/server"
	"github.com/pelletier/go-toml"
)

var (
	configPath = flag.String("config", "bcg-frontend.toml", "Path to the config file for the node")
)

type nodeConfig struct {
	APIPort       int    `toml:"api-port"`
	ClusterPort   int    `toml:"cluster-port"`
	Name          string `toml:"name"`
	WebServerPort int    `toml:"web-server-port"`
}

type conf struct {
	Node nodeConfig `toml:"node"`
}

func main() {
	flag.Parse()

	// Load configuration
	configData, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatal("Could not open config file: ", err)
	}

	config := conf{}
	err = toml.Unmarshal(configData, &config)
	if err != nil {
		log.Fatal("Could not parse config file: ", err)
	}

	// Generate node name if not set
	if config.Node.Name == "" {
		log.Println("Generating node name")
		config.Node.Name = petname.Generate(2, "-")
		updated, err := toml.Marshal(config)
		if err != nil {
			log.Fatal("Could not update config")
		}
		os.WriteFile(*configPath, updated, 0644)
	}

	nodeName := config.Node.Name
	log.Printf("Starting frontend node: %s\n", nodeName)

	// Set up cluster membership
	metaData := &cluster.NodeMeta{NodeType: cluster.Frontend, APIPort: config.Node.APIPort}
	c := memberlist.DefaultLANConfig()
	c.Name = nodeName
	c.BindPort = config.Node.ClusterPort
	c.AdvertisePort = config.Node.ClusterPort
	c.Delegate = cluster.Delegate{MetaData: metaData}

	list, err := memberlist.Create(c)
	if err != nil {
		panic("Failed to create memberlist: " + err.Error())
	}

	// Discover and join cluster
	var entry *zeroconf.ServiceEntry
	found := false

	// Loop until we find at least one bcg node to join
	retryCount := 0
	for !found {
		log.Println("Searching for cluster to join...")

		// Try our improved discovery first
		entry = searchForClusterWithDebug()

		// If that fails after a few tries, fall back to the original method
		if entry == nil && retryCount > 2 {
			log.Println("Trying alternate discovery method...")
			entry = cluster.SearchForCluster()
		}

		if entry != nil && len(entry.AddrIPv4) > 0 {
			found = true
			log.Printf("Found service: %s at %s:%d\n", entry.Instance, entry.AddrIPv4[0].String(), entry.Port)
		} else if entry != nil {
			log.Printf("Found service: %s but no IPv4 address available\n", entry.Instance)
		} else {
			log.Println("No cluster found yet, will retry in 2 seconds...")
			time.Sleep(2 * time.Second)
			retryCount++
		}
	}

	log.Printf("Attempting to join cluster at %s:%d\n", entry.AddrIPv4[0].String(), entry.Port)
	_, err = list.Join([]string{fmt.Sprintf("%s:%d", entry.AddrIPv4[0].String(), entry.Port)})
	if err != nil {
		panic("Failed to join cluster: " + err.Error())
	}
	log.Println("Successfully joined cluster")

	// Start broadcasting our own service so other nodes can find us
	log.Println("Broadcasting my join info")
	zeroconfServer, err := zeroconf.Register(nodeName, cluster.ServiceType, "local.", config.Node.ClusterPort, []string{"txtv=0", "lo=1", "la=2"}, nil)
	if err != nil {
		log.Println("Error starting zeroconf service", err)
	}
	defer zeroconfServer.Shutdown()

	// Find management endpoints
	mgmtEndpoints := findManagementEndpoints(list)
	log.Printf("Found %d management endpoints initially\n", len(mgmtEndpoints))
	for _, ep := range mgmtEndpoints {
		log.Printf("  - Management node at %s:%d\n", ep.Host, ep.Port)
	}

	// Create and start the HTTP server
	httpServer := server.New(config.Node.WebServerPort, config.Node.APIPort, mgmtEndpoints, list)

	// Set up member event handler to track mgmt nodes joining/leaving
	c.Events = cluster.NewEventDelegate([]memberlist.EventDelegate{httpServer})

	// Start the server
	go httpServer.Start()

	// Clean exit handling
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sig:
		log.Println("Shutting down...")
	}

	log.Println("Goodbye.")
}

func searchForClusterWithDebug() *zeroconf.ServiceEntry {
	log.Println("Starting mDNS discovery for _bobcaygeon._tcp...")

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Printf("Failed to initialize resolver: %v\n", err)
		return nil
	}

	entries := make(chan *zeroconf.ServiceEntry)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err = resolver.Browse(ctx, cluster.ServiceType, "local", entries)
	if err != nil {
		log.Printf("Failed to browse: %v\n", err)
		return nil
	}

	var foundEntry *zeroconf.ServiceEntry
	foundChan := make(chan *zeroconf.ServiceEntry, 1)

	go func() {
		for entry := range entries {
			log.Printf("Discovered service: %s at %s:%d (IPv4 count: %d)\n",
				entry.Instance, entry.HostName, entry.Port, len(entry.AddrIPv4))

			// Sometimes we get entries without IPv4 addresses initially
			// Keep looking until we find one with an IPv4 address
			if len(entry.AddrIPv4) > 0 && foundEntry == nil {
				log.Printf("  IPv4 address available: %s\n", entry.AddrIPv4[0].String())
				foundEntry = entry
				select {
				case foundChan <- entry:
				default:
				}
			} else if len(entry.AddrIPv4) == 0 {
				log.Printf("  No IPv4 address yet for %s, waiting for update...\n", entry.Instance)
			}
		}
	}()

	// Wait for either timeout or finding a suitable entry
	select {
	case <-ctx.Done():
		log.Println("Discovery timeout reached")
	case entry := <-foundChan:
		log.Printf("Found suitable entry: %s\n", entry.Instance)
		foundEntry = entry
		cancel() // Stop searching
	}

	// Give a small amount of time for any final updates
	time.Sleep(100 * time.Millisecond)

	if foundEntry != nil && len(foundEntry.AddrIPv4) > 0 {
		log.Printf("Selected service: %s at %s:%d\n",
			foundEntry.Instance, foundEntry.AddrIPv4[0].String(), foundEntry.Port)
	} else {
		log.Println("No suitable service found with IPv4 address")
	}

	return foundEntry
}

func findManagementEndpoints(list *memberlist.Memberlist) []server.MgmtEndpoint {
	log.Printf("Current cluster members: %d\n", list.NumMembers())
	for _, member := range list.Members() {
		meta := cluster.DecodeNodeMeta(member.Meta)
		nodeType := "Unknown"
		switch meta.NodeType {
		case cluster.Music:
			nodeType = "Music"
		case cluster.Mgmt:
			nodeType = "Mgmt"
		case cluster.Frontend:
			nodeType = "Frontend"
		}
		log.Printf("  - %s (%s) at %s, API port: %d\n", member.Name, nodeType, member.Addr.String(), meta.APIPort)
	}

	var endpoints []server.MgmtEndpoint
	for _, member := range cluster.FilterMembers(cluster.Mgmt, list) {
		meta := cluster.DecodeNodeMeta(member.Meta)
		ep := server.MgmtEndpoint{
			Host: member.Addr.String(),
			Port: uint32(meta.APIPort),
		}
		endpoints = append(endpoints, ep)
	}
	return endpoints
}
