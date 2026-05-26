package jobs

import "time"

type Status string

const (
	StatusRunning  Status = "running"
	StatusSuccess  Status = "success"
	StatusFailed   Status = "failed"
	StatusCanceled Status = "canceled"
	StatusTimedOut Status = "timeout"
)

type Job struct {
	ID          string
	Kind        string
	Label       string
	Status      Status
	Message     string
	Error       string
	LogTail     []string
	StartedAt   time.Time
	CompletedAt time.Time
}
