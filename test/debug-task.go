package main

import (
	"fmt"
	"taskd/internal/task"
)

func main() {
	// Test task creation and management
	config := &task.Config{
		Name:       "debug-test",
		Executable: "ping",
		Args:       []string{"127.0.0.1", "-n", "5"},
		InheritEnv: true,
	}
	
	// Create task
	t := task.NewTask(config)
	
	fmt.Printf("Task created: %s\n", t.GetInfo().Name)
	fmt.Printf("Initial status: %s\n", t.GetInfo().Status)
	
	// Start task
	fmt.Println("Starting task...")
	if err := t.Start(); err != nil {
		fmt.Printf("Error starting task: %v\n", err)
		return
	}
	
	fmt.Printf("Task started, status: %s\n", t.GetInfo().Status)
	fmt.Printf("Process ID: %d\n", t.GetInfo().PID)
	fmt.Printf("Start time: %s\n", t.GetInfo().StartTime)
	
	// Wait a bit and check status again
	fmt.Println("Waiting 2 seconds...")
	// time.Sleep(2 * time.Second)
	
	fmt.Printf("Status after wait: %s\n", t.GetInfo().Status)
	fmt.Printf("Process ID: %d\n", t.GetInfo().PID)
}