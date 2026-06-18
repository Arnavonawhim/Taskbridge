package executor

import (
	"context"
	"fmt"
	"os"

	"taskbridge/internal/model"
)

type FileExists struct{}

func (e *FileExists) Type() model.JobType {
	return model.JobFileExists
}

func (e *FileExists) Execute(ctx context.Context, job model.Job) Result {
	path, ok := job.Payload["path"].(string)
	if !ok || path == "" {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"missing or invalid 'path' in payload"},
			Error:  "path is required",
		}
	}

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{
				Status: model.JobFailed,
				Logs:   []string{fmt.Sprintf("file does not exist: %s", path)},
				Error:  "file not found",
			}
		}
		return Result{
			Status: model.JobFailed,
			Logs:   []string{fmt.Sprintf("failed to check file: %v", err)},
			Error:  err.Error(),
		}
	}

	return Result{
		Status: model.JobSuccess,
		Logs:   []string{fmt.Sprintf("file exists: %s", path)},
	}
}
