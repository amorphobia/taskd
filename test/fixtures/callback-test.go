//go:build ignore

package main

import (
	"fmt"
	"time"
	"taskd/internal/task"
)

func main() {
	// Test task with callback
	config := &task.Config{
		Name:       "callback-test",
		Executable: "ping",
		Args:       []string{"127.0.0.1", "-n", "2"},
		InheritEnv: true,
	}
	
	// Create task
	t := task.NewTask(config)
	
	// Set callback
	callbackCalled := false
	t.SetExitCallback(func(taskName string) {
		fmt.Printf("Callback called for task: %s\n", taskName)
		callbackCalled = true
	})
	
	fmt.Printf("Starting task: %s\n", config.Name)
	
	// Start task
	if err := t.Start(); err != nil {
		fmt.Printf("Error starting task: %v\n", err)
		return
	}
	
	fmt.Printf("Task started, waiting for completion...\n")
	
	// Wait for task to complete
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		info := t.GetInfo()
		fmt.Printf("Status: %s, PID: %d\n", info.Status, info.PID)
		
		if info.Status == "stopped" {
			break
		}
	}
	
	// Wait a bit more for callback
	time.Sleep(1 * time.Second)
	
	fmt.Printf("Final status: %s\n", t.GetInfo().Status)
	fmt.Printf("Callback called: %v\n", callbackCalled)
}