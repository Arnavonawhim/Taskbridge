package store

import "taskbridge/internal/model"

// Store defines the required persistence operations.
// Candidate should first implement an in-memory store, then optionally add SQLite.
type Store interface {
	CreateJob(job model.Job) (model.Job, error)
	ListJobs() ([]model.Job, error)
	GetJob(jobID string) (model.Job, bool, error)
	CancelJob(jobID string) error

	RegisterAgent(agent model.Agent) (model.Agent, error)
	Heartbeat(agentID string) error
	ListAgents() ([]model.Agent, error)

	AssignNextJob(agentID string, capabilities []model.JobType) (model.Job, bool, error)
	CompleteJob(jobID string, status model.JobStatus, logs []string, result map[string]any, errMsg string) error
}

// TODO: Candidate should implement MemoryStore with mutex-safe maps.
// TODO: Stretch goal: SQLite-backed Store implementation.
