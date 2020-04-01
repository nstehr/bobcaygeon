//go:generate statik -src=./webui/dist -include=*.jpg,*.png,*.json,*.html,*.css,*.js,*.xml,*.ico

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"net/http"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/grandcat/zeroconf"
	"github.com/hashicorp/memberlist"
	"github.com/nstehr/bobcaygeon/cluster"
	"github.com/nstehr/bobcaygeon/cmd/frontend/control"
	_ "github.com/nstehr/bobcaygeon/cmd/frontend/statik"
	toml "github.com/pelletier/go-toml"
	"github.com/rakyll/statik/fs"
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

type memberHandler struct {
	cp *control.ControlPlane
}

func newMemberHandler(cp *control.ControlPlane) *memberHandler {
	return &memberHandler{cp: cp}
}

// NotifyJoin is invoked when a node is detected to have joined.
// The Node argument must not be modified.
func (m *memberHandler) NotifyJoin(node *memberlist.Node) {
	log.Println("Node Joined " + node.Name)
	meta := cluster.DecodeNodeMeta(node.Meta)
	if meta.NodeType == cluster.Mgmt {
		ep := control.MgmtEndpoint{Host: node.Addr.String(), Port: uint32(meta.APIPort)}
		m.cp.AddEndpoint(ep)
	}

}

// NotifyLeave is invoked when a node is detected to have left.
// The Node argument must not be modified.
func (m *memberHandler) NotifyLeave(node *memberlist.Node) {
	log.Println("Node Left" + node.Name)
	meta := cluster.DecodeNodeMeta(node.Meta)
	if meta.NodeType == cluster.Mgmt {
		ep := control.MgmtEndpoint{Host: node.Addr.String(), Port: uint32(meta.APIPort)}
		m.cp.RemoveEndpoint(ep)
	}
}

// NotifyUpdate is invoked when a node is detected to have
// updated, usually involving the meta data. The Node argument
// must not be modified.
func (*memberHandler) NotifyUpdate(node *memberlist.Node) {
	log.Println("Node updated" + node.Name)

}

func main() {
	flag.Parse()
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

	log.Printf("Starting frontend node: %s\n", nodeName)
	metaData := &cluster.NodeMeta{NodeType: cluster.Frontend, APIPort: config.Node.APIPort}
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

	controlPlane := control.NewControlPlane(config.Node.APIPort)

	// find any mgmt endpoints that are online and we will update our
	// proxy with their connection info
	var endpoints []control.MgmtEndpoint
	for _, member := range cluster.FilterMembers(cluster.Mgmt, list) {
		meta := cluster.DecodeNodeMeta(member.Meta)
		ep := control.MgmtEndpoint{Host: member.Addr.String(), Port: uint32(meta.APIPort)}
		endpoints = append(endpoints, ep)

	}
	if len(endpoints) > 0 {
		controlPlane.UpdateEndpoints(endpoints)
	}

	// sets up the delegate to handle when members join or leave
	c.Events = cluster.NewEventDelegate([]memberlist.EventDelegate{newMemberHandler(controlPlane)})

	go controlPlane.Start()
	go setupWebApp(config.Node.WebServerPort)

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

func setupWebApp(port int) {

	log.Printf("Service Web UI on port: %d\n", port)

	statikFS, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(statikFS)))
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
