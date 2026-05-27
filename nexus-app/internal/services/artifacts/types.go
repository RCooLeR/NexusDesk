// Package artifacts owns deterministic generated files under .nexusdesk/artifacts.
package artifacts

import "time"

type Artifact struct {
	Kind      string
	Title     string
	RelPath   string
	AbsPath   string
	Message   string
	CreatedAt time.Time
}

type TaskRunReport struct {
	ID          string
	JobID       string
	TaskID      string
	Kind        string
	Label       string
	Command     string
	Cwd         string
	Source      string
	Status      string
	ExitCode    int
	Stdout      string
	Stderr      string
	Message     string
	StartedAt   time.Time
	CompletedAt time.Time
	DurationMs  int64
}
