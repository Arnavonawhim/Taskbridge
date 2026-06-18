package executor

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"taskbridge/internal/model"
)

type Checksum struct{}

func (e *Checksum) Type() model.JobType {
	return model.JobChecksum
}

func (e *Checksum) Execute(ctx context.Context, job model.Job) Result {
	path, ok := job.Payload["path"].(string)
	if !ok || path == "" {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"missing or invalid 'path' in payload"},
			Error:  "path is required",
		}
	}

	algo, ok := job.Payload["algorithm"].(string)
	if !ok || algo == "" {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{"missing or invalid 'algorithm' in payload"},
			Error:  "algorithm is required",
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{fmt.Sprintf("failed to open file: %v", err)},
			Error:  err.Error(),
		}
	}
	defer f.Close()

	var h io.Writer
	var sumFunc func() string

	switch algo {
	case "md5":
		hasher := md5.New()
		h = hasher
		sumFunc = func() string {
			return hex.EncodeToString(hasher.Sum(nil))
		}
	case "sha256":
		hasher := sha256.New()
		h = hasher
		sumFunc = func() string {
			return hex.EncodeToString(hasher.Sum(nil))
		}
	default:
		return Result{
			Status: model.JobFailed,
			Logs:   []string{fmt.Sprintf("unsupported algorithm: %s", algo)},
			Error:  "unsupported algorithm",
		}
	}

	if _, err := io.Copy(h, f); err != nil {
		return Result{
			Status: model.JobFailed,
			Logs:   []string{fmt.Sprintf("failed to read file: %v", err)},
			Error:  err.Error(),
		}
	}

	checksum := sumFunc()
	return Result{
		Status: model.JobSuccess,
		Logs:   []string{fmt.Sprintf("%s checksum of %s is %s", algo, path, checksum)},
		Result: map[string]any{"checksum": checksum},
	}
}
