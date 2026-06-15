package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":8080", "server listen address")
	flag.Parse()

	mux := http.NewServeMux()

	// TODO: Candidate should move route registration into internal/api.
	// Required routes:
	//   GET  /health
	//   POST /jobs
	//   GET  /jobs
	//   GET  /jobs/{jobId}
	//   POST /jobs/{jobId}/cancel
	//   POST /agents/register
	//   POST /agents/{agentId}/heartbeat
	//   POST /agents/{agentId}/next-job
	//   POST /jobs/{jobId}/result
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"taskbridge-server"}`))
	})

	fmt.Printf("TaskBridge server listening on %s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
