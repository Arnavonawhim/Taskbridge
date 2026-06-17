package executor

import (
	"context"
	"fmt"
	"net"
	"time"

	"taskbridge/internal/model"
)

type TCPCheck struct{}

func (e *TCPCheck) Type() model.JobType {
	return model.JobTCPCheck
}

func (e *TCPCheck) Execute(ctx context.Context, job model.Job) Result {
	address, ok := job.Payload["address"].(string)
	if !ok || address == "" {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"missing or invalid 'address' in payload"},
			Error:  "address is required",
		}
	}

	timeout := 5 * time.Second
	if val, ok := job.Payload["timeout_seconds"].(float64); ok {
		timeout = time.Duration(val) * time.Second
	}

	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{fmt.Sprintf("failed to connect to %s: %v", address, err)},
			Error:  err.Error(),
		}
	}
	conn.Close()

	return Result{
		Status: model.JobSuccess,
		Logs:   []string{fmt.Sprintf("successfully connected to %s", address)},
	}
}
