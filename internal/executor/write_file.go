package executor

import (
	"context"
	"fmt"
	"os"

	"taskbridge/internal/model"
)

type WriteFile struct{}

func (e *WriteFile) Type() model.JobType {
	return model.JobWriteFile
}

func (e *WriteFile) Execute(ctx context.Context, job model.Job) Result {
	path, ok := job.Payload["path"].(string)
	if !ok || path == "" {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"missing or invalid 'path' in payload"},
			Error:  "path is required",
		}
	}

	content, ok := job.Payload["content"].(string)
	if !ok {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"missing or invalid 'content' in payload"},
			Error:  "content is required",
		}
	}

	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{fmt.Sprintf("failed to write file: %v", err)},
			Error:  err.Error(),
		}
	}

	return Result{
		Status: model.JobSuccess,
		Logs:   []string{fmt.Sprintf("successfully wrote file to %s", path)},
	}
}
