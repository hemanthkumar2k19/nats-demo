package main

import (
	"log"
	"os"
	"os/signal"
	"services/natsclient"
	"syscall"
)

func main() {
	nc, err := natsclient.ConnectWithOptions()
	if err != nil {
		log.Fatalf("Failed to initialize NATS Client: %v", err)
	}

	// Print full connection health checks, server metadata, and statistics
	natsclient.PrintConnectionDetails(nc)


	// Wait for OS shutdown signal (Ctrl+C / SIGINT / SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Received shutdown signal. Draining NATS connection...")
	if err := nc.Drain(); err != nil {
		log.Printf("Error during NATS connection drain: %v", err)
	} else {
		log.Println("NATS connection successfully drained and closed.")
	}
}
