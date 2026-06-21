package executor

import (
	"context"
	"fmt"
	"io"
	"os"

	"taskbridge/internal/model"
)

type CopyFile struct{}

func (e *CopyFile) Type() model.JobType {
	return model.JobCopyFile
}

func (e *CopyFile) Execute(ctx context.Context, job model.Job) Result {
	src, ok := job.Payload["src"].(string)
	if !ok || src == "" {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"missing or invalid 'src' in payload"},
			Error:  "src is required",
		}
	}

	dst, ok := job.Payload["dst"].(string)
	if !ok || dst == "" {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"missing or invalid 'dst' in payload"},
			Error:  "dst is required",
		}
	}

	sf, err := os.Open(src)
	if err != nil {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{fmt.Sprintf("failed to open source file: %v", err)},
			Error:  err.Error(),
		}
	}
	defer sf.Close()

	df, err := os.Create(dst)
	if err != nil {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{fmt.Sprintf("failed to create destination file: %v", err)},
			Error:  err.Error(),
		}
	}
	defer df.Close()

	if _, err := io.Copy(df, sf); err != nil {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{fmt.Sprintf("failed to copy content: %v", err)},
			Error:  err.Error(),
		}
	}

	return Result{
		Status: model.JobSuccess,
		Logs:   []string{fmt.Sprintf("copied file from %s to %s", src, dst)},
	}
}
