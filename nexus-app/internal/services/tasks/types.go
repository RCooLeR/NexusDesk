package tasks

import "time"

type Task struct {
	ID      string
	Kind    string
	Label   string
	Command string
	Cwd     string
	Source  string
}

type Summary struct {
	Tasks       []Task
	Message     string
	GeneratedAt time.Time
}

type RunResult struct {
	Task        Task
	Status      string
	ExitCode    int
	Stdout      string
	Stderr      string
	StartedAt   time.Time
	CompletedAt time.Time
	Duration    time.Duration
	Message     string
}

type TerminalRequest struct {
	Command        string
	Args           []string
	Cwd            string
	TimeoutSeconds int
}

type TerminalResult struct {
	Command     string
	Args        []string
	Cwd         string
	Status      string
	ExitCode    int
	Stdout      string
	Stderr      string
	StartedAt   time.Time
	CompletedAt time.Time
	Duration    time.Duration
	Message     string
}
