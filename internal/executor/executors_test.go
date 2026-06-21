package executor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"taskbridge/internal/model"
)

func TestHTTPCheckExecutor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ex := &HTTPCheck{}
	job := model.Job{
		Payload: map[string]any{
			"url":             server.URL,
			"expected_status": float64(200),
		},
	}

	res := ex.Execute(context.Background(), job)
	if res.Status != model.JobSuccess {
		t.Errorf("expected job success, got %s (err: %s)", res.Status, res.Error)
	}

	failJob := model.Job{
		Payload: map[string]any{
			"url":             server.URL,
			"expected_status": float64(404),
		},
	}

	resFail := ex.Execute(context.Background(), failJob)
	if resFail.Status != model.JobFailed {
		t.Errorf("expected job failure, got %s", resFail.Status)
	}
}

func TestTCPCheckExecutor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	addr := server.Listener.Addr().String()

	ex := &TCPCheck{}
	job := model.Job{
		Payload: map[string]any{
			"address":         addr,
			"timeout_seconds": float64(2),
		},
	}

	res := ex.Execute(context.Background(), job)
	if res.Status != model.JobSuccess {
		t.Errorf("expected TCP connection success, got %s (err: %s)", res.Status, res.Error)
	}

	badJob := model.Job{
		Payload: map[string]any{
			"address": "localhost:9999",
		},
	}

	resBad := ex.Execute(context.Background(), badJob)
	if resBad.Status != model.JobFailed {
		t.Errorf("expected TCP connection failure, got %s", resBad.Status)
	}
}

func TestFileExistsExecutor(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "taskbridge-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	ex := &FileExists{}
	job := model.Job{
		Payload: map[string]any{
			"path": tmpFile,
		},
	}

	res := ex.Execute(context.Background(), job)
	if res.Status != model.JobSuccess {
		t.Errorf("expected success for existing file, got %s", res.Status)
	}

	badJob := model.Job{
		Payload: map[string]any{
			"path": filepath.Join(tmpDir, "nonexistent.txt"),
		},
	}

	resBad := ex.Execute(context.Background(), badJob)
	if resBad.Status != model.JobFailed {
		t.Errorf("expected failure for nonexistent file, got %s", resBad.Status)
	}
}

func TestWriteFileAndChecksum(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "taskbridge-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	targetPath := filepath.Join(tmpDir, "out.txt")

	writeEx := &WriteFile{}
	writeJob := model.Job{
		Payload: map[string]any{
			"path":    targetPath,
			"content": "taskbridge testing",
		},
	}

	writeRes := writeEx.Execute(context.Background(), writeJob)
	if writeRes.Status != model.JobSuccess {
		t.Fatalf("failed to write file: %s (err: %s)", writeRes.Status, writeRes.Error)
	}

	checkEx := &Checksum{}
	checkJob := model.Job{
		Payload: map[string]any{
			"path":      targetPath,
			"algorithm": "md5",
		},
	}

	checkRes := checkEx.Execute(context.Background(), checkJob)
	if checkRes.Status != model.JobSuccess {
		t.Fatalf("failed checksum: %s (err: %s)", checkRes.Status, checkRes.Error)
	}

	md5Hash, ok := checkRes.Result["checksum"].(string)
	if !ok || md5Hash != "044850ef22585eba15c929197ac41437" {
		t.Errorf("unexpected md5 checksum: %s", md5Hash)
	}
}

func TestWaitExecutor(t *testing.T) {
	ex := &Wait{}
	job := model.Job{
		Payload: map[string]any{
			"seconds": float64(0.1),
		},
	}

	start := time.Now()
	res := ex.Execute(context.Background(), job)
	duration := time.Since(start)

	if res.Status != model.JobSuccess {
		t.Errorf("expected success, got %s (err: %s)", res.Status, res.Error)
	}

	if duration < 100*time.Millisecond {
		t.Errorf("expected wait duration at least 100ms, got %v", duration)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cancelJob := model.Job{
		Payload: map[string]any{
			"seconds": float64(2),
		},
	}

	resCancel := ex.Execute(ctx, cancelJob)
	if resCancel.Status != model.JobFailed {
		t.Errorf("expected failure due to context cancellation, got %s", resCancel.Status)
	}
}

func TestCopyFileExecutor(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "taskbridge-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst.txt")

	if err := os.WriteFile(src, []byte("hello copy"), 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	ex := &CopyFile{}
	job := model.Job{
		Payload: map[string]any{
			"src": src,
			"dst": dst,
		},
	}

	res := ex.Execute(context.Background(), job)
	if res.Status != model.JobSuccess {
		t.Errorf("expected success, got %s (err: %s)", res.Status, res.Error)
	}

	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read destination file: %v", err)
	}

	if string(content) != "hello copy" {
		t.Errorf("unexpected content: %s", string(content))
	}
}
