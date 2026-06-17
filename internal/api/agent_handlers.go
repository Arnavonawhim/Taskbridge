package api

import (
	"encoding/json"
	"log"
	"net/http"

	"taskbridge/internal/model"
)

func (h *Handler) RegisterAgent(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	caps := make([]model.JobType, len(req.Capabilities))
	for i, c := range req.Capabilities {
		caps[i] = model.JobType(c)
	}

	agent := model.Agent{
		ID:           req.ID,
		Hostname:     req.Hostname,
		OS:           req.OS,
		Arch:         req.Arch,
		Version:      req.Version,
		Capabilities: caps,
	}

	registered, err := h.store.RegisterAgent(agent)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("[API] Agent registered: %s", registered.ID)
	writeJSON(w, http.StatusOK, registered)
}

func (h *Handler) AgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agentId")

	if err := h.store.Heartbeat(agentID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) NextJob(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agentId")

	var req struct {
		Capabilities []string `json:"capabilities"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	caps := make([]model.JobType, len(req.Capabilities))
	for i, c := range req.Capabilities {
		caps[i] = model.JobType(c)
	}

	job, found, err := h.store.AssignNextJob(agentID, caps)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !found {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	log.Printf("[API] Assigned job %s to agent %s", job.ID, agentID)
	writeJSON(w, http.StatusOK, job)
}

func (h *Handler) SubmitResult(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("jobId")

	var req model.JobResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	status := model.JobStatus(req.Status)
	if status != model.JobSuccess && status != model.JobFailed {
		writeError(w, http.StatusBadRequest, "status must be SUCCESS or FAILED")
		return
	}

	err := h.store.CompleteJob(jobID, status, req.Logs, req.Result, req.Error)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("[API] Job %s completed: %s", jobID, status)
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (h *Handler) ListAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := h.store.ListAgents()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, agents)
}
