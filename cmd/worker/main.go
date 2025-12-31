package main

import (
	"log"
	"os"
	"time"
)

func main() {
	log.Println("[Worker] starting up...")

	// Simulate connecting to NATS
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://nats:4222"
	}
	log.Printf("[Worker] connecting to NATS at %s", natsURL)

	// Example infinite worker loop
	for {
		log.Println("[Worker] running background tasks...")
		time.Sleep(10 * time.Second)
	}
}
