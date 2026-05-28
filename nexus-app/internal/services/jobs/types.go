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

type RetentionPolicy struct {
	KeepRecent      int
	MaxAge          time.Duration
	IncludeFailures bool
	Now             time.Time
}

type RetentionResult struct {
	Removed       int
	Kept          int
	RunningKept   int
	FailuresKept  int
	RepositoryIDs []string
}
