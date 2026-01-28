package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		showHelp()
		return
	}

	command := os.Args[1]
	
	switch command {
	case "add":
		handleAdd()
	case "list":
		handleList()
	case "start":
		handleStart()
	case "stop":
		handleStop()
	case "status":
		handleStatus()
	case "help", "--help", "-h":
		showHelp()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		showHelp()
	}
}

func showHelp() {
	fmt.Println("TaskD - Task daemon management tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  taskd <command> [arguments]")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  add     Add a new task")
	fmt.Println("  list    List all tasks")
	fmt.Println("  start   Start a task")
	fmt.Println("  stop    Stop a task")
	fmt.Println("  status  Show task status")
	fmt.Println("  help    Show help information")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  taskd add mytask")
	fmt.Println("  taskd start mytask")
	fmt.Println("  taskd list")
}

func handleAdd() {
	fmt.Println("Add task functionality (to be implemented)")
	fmt.Println("Will support:")
	fmt.Println("- Specify executable file and arguments")
	fmt.Println("- Set working directory")
	fmt.Println("- Configure environment variables")
	fmt.Println("- Redirect input/output")
}

func handleList() {
	fmt.Println("Task list:")
	fmt.Println("NAME\tSTATUS\tPID\tSTART_TIME")
	fmt.Println("----\t------\t---\t----------")
	fmt.Println("(no tasks)")
}

func handleStart() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: taskd start <task-name>")
		return
	}
	taskName := os.Args[2]
	fmt.Printf("Starting task: %s (to be implemented)\n", taskName)
}

func handleStop() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: taskd stop <task-name>")
		return
	}
	taskName := os.Args[2]
	fmt.Printf("Stopping task: %s (to be implemented)\n", taskName)
}

func handleStatus() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: taskd status <task-name>")
		return
	}
	taskName := os.Args[2]
	fmt.Printf("Task status: %s (to be implemented)\n", taskName)
}