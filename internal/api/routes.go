package api

import "net/http"

func RegisterRoutes(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("GET /health", h.Health)

	mux.HandleFunc("POST /jobs", h.CreateJob)
	mux.HandleFunc("GET /jobs", h.ListJobs)
	mux.HandleFunc("GET /jobs/{jobId}", h.GetJob)
	mux.HandleFunc("POST /jobs/{jobId}/result", h.SubmitResult)
	mux.HandleFunc("POST /agents/register", h.RegisterAgent)
	mux.HandleFunc("POST /agents/{agentId}/heartbeat", h.AgentHeartbeat)
	mux.HandleFunc("POST /agents/{agentId}/next-job", h.NextJob)
	mux.HandleFunc("GET /agents", h.ListAgents)
}
