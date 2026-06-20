package store

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"taskbridge/internal/model"
)

// ─────────────────────────────────────────────────────────────────────────────
// MemoryStore stores jobs and agents in plain Go maps, protected by a mutex.
//
// WHY A MUTEX?
// The HTTP server handles each request in its own goroutine.  If two requests
// arrive at the same time (e.g. two curl commands), both goroutines might try
// to read/write the same map simultaneously — which causes a data race and
// can crash the program.  The mutex ensures only ONE goroutine touches the
// maps at a time.
//
// Pattern:
//   m.mu.Lock()        ← "I'm about to read/write, everyone else wait"
//   defer m.mu.Unlock() ← "when this function returns, release the lock"
//   ... do map operations ...
// ─────────────────────────────────────────────────────────────────────────────

// MemoryStore implements the Store interface using in-memory maps.
type MemoryStore struct {
	mu     sync.Mutex
	jobs   map[string]model.Job
	agents map[string]model.Agent
}

// NewMemoryStore creates an initialised MemoryStore with empty maps.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		jobs:   make(map[string]model.Job),
		agents: make(map[string]model.Agent),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Job operations
// ─────────────────────────────────────────────────────────────────────────────

// CreateJob stores a new job with status PENDING and a generated ID.
func (m *MemoryStore) CreateJob(job model.Job) (model.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate a unique ID for this job.
	job.ID = generateID("job")
	job.Status = model.JobPending
	job.CreatedAt = time.Now()
	job.AttemptCount = 0

	m.jobs[job.ID] = job
	return job, nil
}

func (m *MemoryStore) ListJobs() ([]model.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]model.Job, 0, len(m.jobs))
	for _, j := range m.jobs {
		result = append(result, j)
	}
	return result, nil
}

// GetJob returns a single job by ID.  The bool indicates whether it was found.
func (m *MemoryStore) GetJob(jobID string) (model.Job, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	return job, ok, nil
}

// CancelJob transitions a PENDING job to CANCELED.
func (m *MemoryStore) CancelJob(jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return fmt.Errorf("job %s not found", jobID)
	}
	if job.Status != model.JobPending {
		return fmt.Errorf("cannot cancel job in %s state", job.Status)
	}

	job.Status = model.JobCanceled
	now := time.Now()
	job.FinishedAt = &now
	m.jobs[jobID] = job
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Agent operations
// ─────────────────────────────────────────────────────────────────────────────

// RegisterAgent adds or updates an agent record.  If the agent ID already
// exists we treat it as a re-registration (update fields, set online).
func (m *MemoryStore) RegisterAgent(agent model.Agent) (model.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if agent.ID == "" {
		agent.ID = generateID("agent")
	}
	agent.LastSeen = time.Now()
	agent.Status = "online"

	m.agents[agent.ID] = agent
	return agent, nil
}

// Heartbeat updates the agent's LastSeen timestamp and sets status to online.
func (m *MemoryStore) Heartbeat(agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, ok := m.agents[agentID]
	if !ok {
		return fmt.Errorf("agent %s not found", agentID)
	}

	agent.LastSeen = time.Now()
	agent.Status = "online"
	m.agents[agentID] = agent
	return nil
}

func (m *MemoryStore) ListAgents() ([]model.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]model.Agent, 0, len(m.agents))
	for _, a := range m.agents {
		result = append(result, a)
	}
	return result, nil
}

func (m *MemoryStore) CleanStaleAgents(timeout time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-timeout)
	for id, agent := range m.agents {
		if agent.LastSeen.Before(cutoff) {
			agent.Status = "offline"
			m.agents[id] = agent
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Job assignment and completion
// ─────────────────────────────────────────────────────────────────────────────

// AssignNextJob finds the first PENDING (or RETRYING) job whose type matches
// one of the agent's capabilities, marks it RUNNING, and returns it.
// If no compatible job is found, returns (zero, false, nil).
func (m *MemoryStore) AssignNextJob(agentID string, capabilities []model.JobType) (model.Job, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Build a lookup set for quick capability checks.
	capSet := make(map[model.JobType]bool, len(capabilities))
	for _, c := range capabilities {
		capSet[c] = true
	}

	for id, job := range m.jobs {
		// Only assign jobs that are waiting to be picked up.
		if job.Status != model.JobPending && job.Status != model.JobRetrying {
			continue
		}
		// Only assign if the agent supports this job type.
		if !capSet[job.Type] {
			continue
		}

		// Transition the job to RUNNING.
		now := time.Now()
		job.Status = model.JobRunning
		job.AssignedAgentID = agentID
		job.StartedAt = &now
		job.AttemptCount++
		m.jobs[id] = job

		return job, true, nil
	}

	return model.Job{}, false, nil
}

// CompleteJob records the final result of a job execution.
// If the job failed and retries remain, it goes to RETRYING instead of FAILED.
func (m *MemoryStore) CompleteJob(jobID string, status model.JobStatus, logs []string, result map[string]any, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return fmt.Errorf("job %s not found", jobID)
	}

	// If the executor reported FAILED but the job still has retries left,
	// put it back into RETRYING so the next AssignNextJob call picks it up.
	if status == model.JobFailed && job.AttemptCount < job.MaxRetries {
		job.Status = model.JobRetrying
		job.Logs = append(job.Logs, logs...)
		job.Error = errMsg
		job.AssignedAgentID = "" // free it for re-assignment
		m.jobs[jobID] = job
		return nil
	}

	// Terminal state — either SUCCESS or final FAILED.
	now := time.Now()
	job.Status = status
	job.FinishedAt = &now
	job.Logs = append(job.Logs, logs...)
	job.Result = result
	job.Error = errMsg
	m.jobs[jobID] = job
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

// generateID creates a short, unique identifier like "job-a1b2c3d4e5f6".
// We use crypto/rand (not math/rand) so IDs are truly unpredictable.
func generateID(prefix string) string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%x", prefix, b)
}
