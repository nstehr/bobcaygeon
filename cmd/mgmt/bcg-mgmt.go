package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/grandcat/zeroconf"
	"github.com/hashicorp/memberlist"
	"github.com/nstehr/bobcaygeon/cluster"
	toml "github.com/pelletier/go-toml"
)

var (
	configPath = flag.String("config", "bcg-mgmt.toml", "Path to the config file for the node")
)

const (
	serviceType = "_bobcaygeon._tcp"
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

	nodeName := petname.Generate(2, "-")
	log.Printf("Starting management API node: %s\n", nodeName)
	metaData := &cluster.NodeMeta{NodeType: cluster.Mgmt}
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
			found = true
		case <-ctx.Done():
			log.Println("cluster search timeout, no cluster to join")
		}
	}

	log.Println("Joining cluster")
	_, err = list.Join([]string{fmt.Sprintf("%s:%d", entry.AddrIPv4[0].String(), entry.Port)})
	if err != nil {
		panic("Failed to join cluster: " + err.Error())
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
