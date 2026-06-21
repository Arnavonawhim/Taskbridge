package store_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"taskbridge/internal/agent"
	"taskbridge/internal/api"
	"taskbridge/internal/executor"
	"taskbridge/internal/model"
	"taskbridge/internal/store"
)

func TestAPIIntegrationFlow(t *testing.T) {
	memStore := store.NewMemoryStore()
	handler := api.NewHandler(memStore)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux, handler)

	server := httptest.NewServer(mux)
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	registry := executor.NewRegistry()
	registry.Register(&executor.HTTPCheck{})
	registry.Register(&executor.TCPCheck{})
	registry.Register(&executor.FileExists{})
	registry.Register(&executor.Checksum{})
	registry.Register(&executor.CopyFile{})
	registry.Register(&executor.WriteFile{})
	registry.Register(&executor.Wait{})

	a := agent.New(server.URL, "agent-integration-test", []string{"http_check", "tcp_check", "file_exists", "checksum", "copy_file", "write_file", "wait"}, 50*time.Millisecond, registry)
	go func() {
		if err := a.Run(); err != nil {
			t.Errorf("agent run failed: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	agentsResp, err := client.Get(server.URL + "/agents")
	if err != nil {
		t.Fatalf("failed to list agents: %v", err)
	}
	defer agentsResp.Body.Close()
	var agents []model.Agent
	if err := json.NewDecoder(agentsResp.Body).Decode(&agents); err != nil {
		t.Fatalf("failed to decode agents: %v", err)
	}
	if len(agents) != 1 || agents[0].ID != "agent-integration-test" {
		t.Fatalf("expected agent-integration-test in list, got: %+v", agents)
	}

	tmpDir, err := os.MkdirTemp("", "taskbridge-integration")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fpath := filepath.Join(tmpDir, "file.txt")
	copypath := filepath.Join(tmpDir, "copy.txt")

	jobsToCreate := []model.CreateJobRequest{
		{
			Name: "http-check-test",
			Type: "http_check",
			Payload: map[string]any{
				"url":             server.URL + "/health",
				"expected_status": 200,
			},
		},
		{
			Name: "write-file-test",
			Type: "write_file",
			Payload: map[string]any{
				"path":    fpath,
				"content": "hello taskbridge",
			},
		},
		{
			Name: "file-exists-test",
			Type: "file_exists",
			Payload: map[string]any{
				"path": fpath,
			},
		},
		{
			Name: "checksum-test",
			Type: "checksum",
			Payload: map[string]any{
				"path":      fpath,
				"algorithm": "md5",
			},
		},
		{
			Name: "copy-file-test",
			Type: "copy_file",
			Payload: map[string]any{
				"src": fpath,
				"dst": copypath,
			},
		},
		{
			Name: "wait-test",
			Type: "wait",
			Payload: map[string]any{
				"seconds": 0.1,
			},
		},
	}

	for _, req := range jobsToCreate {
		body, _ := json.Marshal(req)
		postResp, err := client.Post(server.URL+"/jobs", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("failed to create job %s: %v", req.Name, err)
		}
		if postResp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d", postResp.StatusCode)
		}
		var created model.Job
		_ = json.NewDecoder(postResp.Body).Decode(&created)
		postResp.Body.Close()

		time.Sleep(200 * time.Millisecond)

		getResp, err := client.Get(server.URL + "/jobs/" + created.ID)
		if err != nil {
			t.Fatalf("failed to fetch job: %v", err)
		}
		var fetched model.Job
		_ = json.NewDecoder(getResp.Body).Decode(&fetched)
		getResp.Body.Close()

		if fetched.Status != model.JobSuccess {
			t.Errorf("job %s status is %s, expected SUCCESS (error: %s, logs: %v)", req.Name, fetched.Status, fetched.Error, fetched.Logs)
		}
	}

	listJobsResp, err := client.Get(server.URL + "/jobs")
	if err != nil {
		t.Fatalf("failed to list jobs: %v", err)
	}
	defer listJobsResp.Body.Close()
	var jobs []model.Job
	if err := json.NewDecoder(listJobsResp.Body).Decode(&jobs); err != nil {
		t.Fatalf("failed to decode jobs: %v", err)
	}
	if len(jobs) != len(jobsToCreate) {
		t.Errorf("expected %d jobs, got %d", len(jobsToCreate), len(jobs))
	}

	heartbeatResp, err := client.Post(server.URL+"/agents/agent-integration-test/heartbeat", "application/json", nil)
	if err != nil {
		t.Fatalf("failed to send heartbeat: %v", err)
	}
	defer heartbeatResp.Body.Close()
	if heartbeatResp.StatusCode != http.StatusOK {
		t.Errorf("expected heartbeat status 200, got %d", heartbeatResp.StatusCode)
	}
}
