package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"taskd/internal/cli"
	"taskd/internal/task"
)

func main() {
	// Check command line arguments directly for --daemon flag
	for _, arg := range os.Args[1:] {
		if arg == "--daemon" {
			runDaemonMode()
			return
		}
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
	
	// Initialize task monitor with 5 second check interval
	monitor := task.NewTaskMonitor(5 * time.Second)
	
	// Set up signal handling for graceful shutdown
	setupSignalHandling(monitor)
	
	// Start monitoring
	monitor.Start()
}

// setupSignalHandling sets up signal handling for graceful daemon shutdown
func setupSignalHandling(monitor *task.TaskMonitor) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		sig := <-sigChan
		fmt.Printf("Received signal %v, shutting down daemon...\n", sig)
		
		// Stop monitoring gracefully
		monitor.Stop()
		
		os.Exit(0)
	}()
}