package executor

import (
	"context"
	"fmt"
	"net/http"

	"taskbridge/internal/model"
)

type HTTPCheck struct{}

func (e *HTTPCheck) Type() model.JobType {
	return model.JobHTTPCheck
}

func (e *HTTPCheck) Execute(ctx context.Context, job model.Job) Result {
	url, ok := job.Payload["url"].(string)
	if !ok || url == "" {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"missing or invalid 'url' in payload"},
			Error:  "url is required",
		}
	}

	expectedStatus := 200
	if v, ok := job.Payload["expected_status"].(float64); ok {
		expectedStatus = int(v)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{fmt.Sprintf("failed to create request: %v", err)},
			Error:  err.Error(),
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{fmt.Sprintf("request failed: %v", err)},
			Error:  err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{fmt.Sprintf("expected status %d, got %d", expectedStatus, resp.StatusCode)},
			Error:  fmt.Sprintf("status mismatch: expected %d got %d", expectedStatus, resp.StatusCode),
		}
	}

	return Result{
		Status: model.JobSuccess,
		Logs:   []string{fmt.Sprintf("GET %s returned %d", url, resp.StatusCode)},
		Result: map[string]any{"status_code": resp.StatusCode},
	}
}
