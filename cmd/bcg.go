package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nstehr/bobcaygeon/raop"
)

var (
	name = flag.String("name", "Bobcaygeon", "The name for the service.")
	port = flag.Int("port", 5000, "Set the port the service is listening to.")
)

func main() {
	flag.Parse()

	airplayServer := raop.NewAirplayServer(*port, *name)
	go airplayServer.Start(true)

	// Clean exit.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sig:
		// Exit by user
		airplayServer.Stop()

	}

	log.Println("Shutting down.")
}
