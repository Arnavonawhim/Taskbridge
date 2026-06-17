package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"taskbridge/internal/executor"
	"taskbridge/internal/model"
)

type Agent struct {
	serverURL    string
	id           string
	capabilities []string
	pollInterval time.Duration
	registry     *executor.Registry
	client       *http.Client
}

func New(serverURL, id string, capabilities []string, pollInterval time.Duration, registry *executor.Registry) *Agent {
	return &Agent{
		serverURL:    serverURL,
		id:           id,
		capabilities: capabilities,
		pollInterval: pollInterval,
		registry:     registry,
		client:       &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *Agent) Run() error {
	if err := a.register(); err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}
	log.Printf("registered with server %s", a.serverURL)

	a.startHeartbeat()
	a.pollLoop()
	return nil
}

func (a *Agent) startHeartbeat() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			_, _ = a.post("/agents/"+a.id+"/heartbeat", nil)
		}
	}()
}

func (a *Agent) register() error {
	req := model.RegisterAgentRequest{
		ID:           a.id,
		Hostname:     getHostname(),
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		Version:      "0.1.0",
		Capabilities: a.capabilities,
	}
	_, err := a.post("/agents/register", req)
	return err
}

func (a *Agent) pollLoop() {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	log.Printf("polling for jobs every %s", a.pollInterval)
	for range ticker.C {
		a.pollOnce()
	}
}

func (a *Agent) pollOnce() {
	data, err := a.post("/agents/"+a.id+"/next-job", map[string]any{
		"capabilities": a.capabilities,
	})
	if err != nil {
		log.Printf("poll error: %v", err)
		return
	}
	if len(data) == 0 {
		return
	}

	var job model.Job
	if err := json.Unmarshal(data, &job); err != nil || job.ID == "" {
		return
	}

	log.Printf("received job %s (%s) type=%s", job.ID, job.Name, job.Type)
	a.executeAndReport(job)
}

func (a *Agent) executeAndReport(job model.Job) {
	ex, ok := a.registry.Get(job.Type)
	if !ok {
		a.reportResult(job.ID, executor.Result{
			Status: model.JobFailed,
			Logs:   []string{fmt.Sprintf("unsupported job type: %s", job.Type)},
			Error:  "no executor available",
		})
		return
	}

	ctx := context.Background()
	if job.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(job.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	result := ex.Execute(ctx, job)
	log.Printf("job %s completed: %s", job.ID, result.Status)
	a.reportResult(job.ID, result)
}

func (a *Agent) reportResult(jobID string, result executor.Result) {
	body := model.JobResultRequest{
		Status: string(result.Status),
		Logs:   result.Logs,
		Result: result.Result,
		Error:  result.Error,
	}
	_, err := a.post("/jobs/"+jobID+"/result", body)
	if err != nil {
		log.Printf("failed to report result for job %s: %v", jobID, err)
	}
}

func (a *Agent) post(path string, body any) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	resp, err := a.client.Post(a.serverURL+path, "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func getHostname() string {
	h, _ := os.Hostname()
	return h
}
