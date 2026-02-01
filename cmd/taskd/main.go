package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"taskd/internal/cli"
)

func main() {
	// Check if running in daemon mode
	if cli.IsDaemonMode() {
		runDaemonMode()
		return
	}
	
	// Normal command mode
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runDaemonMode runs the daemon process
func runDaemonMode() {
	fmt.Println("Starting TaskD daemon...")
	
	// TODO: Initialize task monitor
	// monitor := &TaskMonitor{
	//     checkInterval: 5 * time.Second,
	//     stopChan:      make(chan struct{}),
	// }
	
	// Set up signal handling for graceful shutdown
	setupSignalHandling()
	
	// TODO: Start monitoring
	// monitor.Start()
	
	// For now, just run indefinitely
	select {}
}

// setupSignalHandling sets up signal handling for graceful daemon shutdown
func setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		sig := <-sigChan
		fmt.Printf("Received signal %v, shutting down daemon...\n", sig)
		
		// TODO: Stop monitoring gracefully
		// monitor.Stop()
		
		os.Exit(0)
	}()
}