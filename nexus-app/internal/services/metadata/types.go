// Package metadata owns the native SQLite metadata store under .nexusdesk.
package metadata

import "time"

type Status struct {
	Path          string
	SchemaPath    string
	SchemaVersion int
	SchemaHash    string
	Tables        []string
	Message       string
	UpdatedAt     time.Time
}

type TaskRunRecord struct {
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
