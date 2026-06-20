package store

import (
	"testing"
	"time"

	"taskbridge/internal/model"
)

func TestJobLifecycle(t *testing.T) {
	s := NewMemoryStore()

	job, err := s.CreateJob(model.Job{
		Name:       "test-job",
		Type:       model.JobHTTPCheck,
		MaxRetries: 2,
	})
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	if job.ID == "" {
		t.Error("expected non-empty job ID")
	}
	if job.Status != model.JobPending {
		t.Errorf("expected job status PENDING, got %s", job.Status)
	}

	foundJob, found, err := s.GetJob(job.ID)
	if err != nil || !found {
		t.Fatalf("failed to get job: %v", err)
	}
	if foundJob.Name != "test-job" {
		t.Errorf("expected name 'test-job', got %s", foundJob.Name)
	}

	jobs, err := s.ListJobs()
	if err != nil || len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
}

func TestAssignAndRetry(t *testing.T) {
	s := NewMemoryStore()

	_, _ = s.CreateJob(model.Job{
		Name:       "test-job",
		Type:       model.JobHTTPCheck,
		MaxRetries: 2,
	})

	job, found, err := s.AssignNextJob("agent-1", []model.JobType{model.JobHTTPCheck})
	if err != nil || !found {
		t.Fatalf("expected job to be assigned")
	}
	if job.Status != model.JobRunning {
		t.Errorf("expected status RUNNING, got %s", job.Status)
	}
	if job.AttemptCount != 1 {
		t.Errorf("expected attempt count 1, got %d", job.AttemptCount)
	}

	err = s.CompleteJob(job.ID, model.JobFailed, []string{"failure log"}, nil, "some error")
	if err != nil {
		t.Fatalf("failed to complete job: %v", err)
	}

	retryJob, found, err := s.GetJob(job.ID)
	if err != nil || !found {
		t.Fatalf("failed to get job")
	}
	if retryJob.Status != model.JobRetrying {
		t.Errorf("expected status RETRYING, got %s", retryJob.Status)
	}

	assignedJob, found, err := s.AssignNextJob("agent-1", []model.JobType{model.JobHTTPCheck})
	if err != nil || !found {
		t.Fatalf("expected job to be assigned on retry")
	}
	if assignedJob.AttemptCount != 2 {
		t.Errorf("expected attempt count 2, got %d", assignedJob.AttemptCount)
	}

	err = s.CompleteJob(assignedJob.ID, model.JobFailed, []string{"final failure log"}, nil, "final error")
	if err != nil {
		t.Fatalf("failed to complete job second time: %v", err)
	}

	finalJob, found, err := s.GetJob(assignedJob.ID)
	if err != nil || !found {
		t.Fatalf("failed to get job")
	}
	if finalJob.Status != model.JobFailed {
		t.Errorf("expected status FAILED, got %s", finalJob.Status)
	}
}

func TestAgentCleanup(t *testing.T) {
	s := NewMemoryStore()

	agent, err := s.RegisterAgent(model.Agent{
		ID:           "agent-1",
		Capabilities: []model.JobType{model.JobHTTPCheck},
	})
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	if agent.Status != "online" {
		t.Errorf("expected status 'online', got %s", agent.Status)
	}

	s.agents["agent-1"] = model.Agent{
		ID:       "agent-1",
		Status:   "online",
		LastSeen: time.Now().Add(-20 * time.Second),
	}

	err = s.CleanStaleAgents(15 * time.Second)
	if err != nil {
		t.Fatalf("failed to clean stale agents: %v", err)
	}

	cleaned, err := s.ListAgents()
	if err != nil || len(cleaned) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(cleaned))
	}

	if cleaned[0].Status != "offline" {
		t.Errorf("expected status 'offline', got %s", cleaned[0].Status)
	}
}
