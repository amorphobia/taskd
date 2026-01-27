package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Quick exit program started")
	fmt.Printf("Current time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("Doing some work...")
	
	// Simulate some work
	time.Sleep(2 * time.Second)
	
	fmt.Println("Work completed, exiting...")
	fmt.Printf("Exit time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
}