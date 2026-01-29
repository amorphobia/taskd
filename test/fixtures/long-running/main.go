package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	fmt.Println("Long running program started")
	fmt.Printf("Start time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("Press Ctrl+C to stop...")

	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	
	// Register the channel to receive specific signals
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a ticker for periodic output
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	// Main loop
	for {
		select {
		case sig := <-sigChan:
			fmt.Printf("\nReceived signal: %v\n", sig)
			fmt.Printf("Gracefully shutting down at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
			fmt.Println("Cleanup completed, goodbye!")
			return
		case t := <-ticker.C:
			fmt.Printf("Still running... %s\n", t.Format("2006-01-02 15:04:05"))
		}
	}
}