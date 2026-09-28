package main

import (
	"log"
	"os"
	"os/signal"
	"services/natsclient"
	"syscall"
)

func main() {
	// nc, err := natsclient.AuthUsernamePasswordConnect("admin", "pwd", "localhost:4222")
	nc, err := natsclient.AuthTokenConnect("88eab73c32486334dbad5bad67f1a6adabdc5e8eecbf38c4b9e3b6dc3cda6fe0", "nats://localhost:4222")
	if err != nil {
		log.Fatalf("Failed to initialize NATS Client: %v", err)
	}

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
