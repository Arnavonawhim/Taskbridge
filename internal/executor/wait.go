package executor

import (
	"context"
	"fmt"
	"time"

	"taskbridge/internal/model"
)

type Wait struct{}

func (e *Wait) Type() model.JobType {
	return model.JobWait
}

func (e *Wait) Execute(ctx context.Context, job model.Job) Result {
	seconds, ok := job.Payload["seconds"].(float64)
	if !ok || seconds < 0 {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"missing or invalid 'seconds' in payload"},
			Error:  "seconds is required and must be non-negative",
		}
	}

	d := time.Duration(seconds * float64(time.Second))
	select {
	case <-time.After(d):
		return Result{
			Status: model.JobSuccess,
			Logs:   []string{fmt.Sprintf("waited for %v seconds", seconds)},
		}
	case <-ctx.Done():
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"wait was canceled or timed out"},
			Error:  ctx.Err().Error(),
		}
	}
}
