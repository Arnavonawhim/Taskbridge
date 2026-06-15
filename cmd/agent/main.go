package main

import (
	"flag"
	"fmt"
	"time"
)

func main() {
	serverURL := flag.String("server", "http://localhost:8080", "TaskBridge server URL")
	agentID := flag.String("id", "agent-dev-1", "agent identifier")
	capabilities := flag.String("capabilities", "http_check,tcp_check,file_exists", "comma-separated job capabilities")
	pollInterval := flag.Duration("poll-interval", 3*time.Second, "job polling interval")
	flag.Parse()

	// TODO: Candidate should implement:
	//   1. Register agent with server
	//   2. Send periodic heartbeat
	//   3. Poll server for next job
	//   4. Execute job using internal/executor
	//   5. Report logs/result back to server

	fmt.Println("TaskBridge agent starter")
	fmt.Println("server:", *serverURL)
	fmt.Println("agent_id:", *agentID)
	fmt.Println("capabilities:", *capabilities)
	fmt.Println("poll_interval:", *pollInterval)
}
