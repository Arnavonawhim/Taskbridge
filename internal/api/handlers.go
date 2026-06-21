package api

import (
	"encoding/json"
	"log"
	"net/http"
	"taskbridge/internal/model"
	"taskbridge/internal/store"
)

type Handler struct {
	store store.Store
}

func NewHandler(s store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "taskbridge-server",
	})
}

func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req model.CreateJobRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}

	validTypes := map[model.JobType]bool{
		model.JobHTTPCheck:  true,
		model.JobTCPCheck:   true,
		model.JobFileExists: true,
		model.JobChecksum:   true,
		model.JobCopyFile:   true,
		model.JobWriteFile:  true,
		model.JobWait:       true,
	}
	if !validTypes[model.JobType(req.Type)] {
		writeError(w, http.StatusBadRequest, "unsupported job type: "+req.Type)
		return
	}

	job := model.Job{
		Name:           req.Name,
		Type:           model.JobType(req.Type),
		Payload:        req.Payload,
		MaxRetries:     req.MaxRetries,
		TimeoutSeconds: req.TimeoutSeconds,
	}

	created, err := h.store.CreateJob(job)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("[API] Created job %s (%s) type=%s", created.ID, created.Name, created.Type)
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.store.ListJobs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("jobId")

	job, found, err := h.store.GetJob(jobID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "job not found: "+jobID)
		return
	}

	writeJSON(w, http.StatusOK, job)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[API] failed to write JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, model.ErrorResponse{Error: message})
}
