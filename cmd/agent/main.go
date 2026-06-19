package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"taskbridge/internal/agent"
	"taskbridge/internal/executor"
)

func main() {
	serverURL := flag.String("server", "http://localhost:8080", "TaskBridge server URL")
	agentID := flag.String("id", "agent-dev-1", "agent identifier")
	caps := flag.String("capabilities", "http_check", "comma-separated job capabilities")
	pollInterval := flag.Duration("poll-interval", 3*time.Second, "job polling interval")
	flag.Parse()

	registry := executor.NewRegistry()
	registry.Register(&executor.HTTPCheck{})
	registry.Register(&executor.TCPCheck{})
	registry.Register(&executor.FileExists{})
	registry.Register(&executor.Checksum{})
	registry.Register(&executor.WriteFile{})
	registry.Register(&executor.Wait{})

	capabilities := strings.Split(*caps, ",")

	a := agent.New(*serverURL, *agentID, capabilities, *pollInterval, registry)

	log.Printf("starting agent %s (server=%s, caps=%v)", *agentID, *serverURL, capabilities)
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
