package startup

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	statusRunning = "running"
	statusClosed  = "closed"

	sessionRelativePath = "NexusDesk/startup-session.json"
)

type Session struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	AppName   string `json:"appName"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	PID       int    `json:"pid"`
	StartedAt string `json:"startedAt"`
	ClosedAt  string `json:"closedAt,omitempty"`
}

type Status struct {
	Path              string
	CurrentID         string
	CurrentStartedAt  time.Time
	Previous          Session
	PreviousUnclean   bool
	PreviousStartedAt time.Time
	Message           string
}

type Options struct {
	AppName string
	Version string
	Commit  string
	Now     time.Time
	PID     int
}

type Store struct {
	path string
}

func NewStore() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return NewFileStore(filepath.Join(dir, sessionRelativePath)), nil
}

func NewFileStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *Store) Begin(options Options) (Status, error) {
	if s == nil || strings.TrimSpace(s.path) == "" {
		return Status{}, errors.New("startup session store path is required")
	}
	now := options.Now
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	pid := options.PID
	if pid == 0 {
		pid = os.Getpid()
	}
	previous, _ := s.read()
	current := Session{
		ID:        sessionID(now, pid),
		Status:    statusRunning,
		AppName:   strings.TrimSpace(options.AppName),
		Version:   strings.TrimSpace(options.Version),
		Commit:    strings.TrimSpace(options.Commit),
		PID:       pid,
		StartedAt: formatTime(now),
	}
	if err := s.write(current); err != nil {
		return Status{}, err
	}
	status := Status{
		Path:             s.path,
		CurrentID:        current.ID,
		CurrentStartedAt: now,
		Previous:         previous,
		Message:          "Startup session marker recorded.",
	}
	status.PreviousStartedAt = parseTime(previous.StartedAt)
	status.PreviousUnclean = strings.EqualFold(strings.TrimSpace(previous.Status), statusRunning)
	if status.PreviousUnclean {
		status.Message = "Previous NexusDesk run did not record a clean exit. Check Diagnostics, Jobs, Agent Audit, metadata health, and issue-report export before retrying long work."
	}
	return status, nil
}

func (s *Store) MarkClean(id string, at time.Time) error {
	if s == nil || strings.TrimSpace(s.path) == "" {
		return errors.New("startup session store path is required")
	}
	session, err := s.read()
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if strings.TrimSpace(id) != "" && strings.TrimSpace(session.ID) != strings.TrimSpace(id) {
		return nil
	}
	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}
	session.Status = statusClosed
	session.ClosedAt = formatTime(at)
	return s.write(session)
}

func (s *Store) read() (Session, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return Session{}, err
	}
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Store) write(session Session) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tempPath, s.path)
}

func sessionID(now time.Time, pid int) string {
	return now.UTC().Format("20060102T150405.000000000Z") + "-" + strconv.Itoa(pid)
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}
