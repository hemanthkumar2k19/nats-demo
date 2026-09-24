package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"services/messaging"
	"services/natsclient"
	"syscall"
	"time"
)

func main() {
	nc, err := natsclient.ConnectWithOptions()
	if err != nil {
		log.Fatalf("Failed to initialize NATS Client: %v", err)
	}

	// Subscribing
	sub, err := messaging.Subscribe(nc, "orders.*")
	if err != nil {
		log.Fatalf("Failed to subscribe to NATS: %v", err)
	}
	defer sub.Unsubscribe()

	// Publishing
	messaging.Publish(nc, "orders.placed", []byte("Order 1"))
	messaging.Publish(nc, "orders.shipped", []byte("Order 2"))
	messaging.Publish(nc, "orders.cancelled.test", []byte("Order 3"))

	// Request Reply
	resp, err := messaging.Request(nc, "orders.payment", []byte("Order 4"), 2*time.Second)
	if err != nil {
		log.Fatalf("Failed to request: %v", err)
	}
	fmt.Printf("Received response: %s\n", resp.Data)

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
