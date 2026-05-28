package perf

import (
	"strings"
	"sync"
	"time"
)

const (
	TimingStartupReady      = "startup-ready"
	TimingWorkspaceOpen     = "workspace-open"
	TimingWorkspaceMetadata = "workspace-metadata-open"

	StartupReadyBudget      = 2 * time.Second
	WorkspaceOpenBudget     = 2 * time.Second
	WorkspaceMetadataBudget = 750 * time.Millisecond
)

type TimingRecord struct {
	Name         string
	StartedAt    time.Time
	CompletedAt  time.Time
	Duration     time.Duration
	Budget       time.Duration
	Detail       string
	WithinBudget bool
}

type Recorder struct {
	mu      sync.Mutex
	limit   int
	records []TimingRecord
}

func NewRecorder(limit int) *Recorder {
	if limit <= 0 {
		limit = 64
	}
	return &Recorder{limit: limit}
}

func (r *Recorder) Record(name string, started time.Time, completed time.Time, budget time.Duration, detail string) TimingRecord {
	completed = normalizeCompletedAt(completed)
	if started.IsZero() {
		started = completed
	}
	if completed.Before(started) {
		completed = started
	}
	record := TimingRecord{
		Name:         firstTimingValue(name, "timing"),
		StartedAt:    started.UTC(),
		CompletedAt:  completed.UTC(),
		Duration:     completed.Sub(started),
		Budget:       budget,
		Detail:       strings.TrimSpace(detail),
		WithinBudget: budget <= 0 || completed.Sub(started) <= budget,
	}
	if r == nil {
		return record
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, record)
	if len(r.records) > r.limit {
		r.records = append([]TimingRecord(nil), r.records[len(r.records)-r.limit:]...)
	}
	return record
}

func (r *Recorder) Snapshot(limit int) []TimingRecord {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	records := r.records
	if limit > 0 && len(records) > limit {
		records = records[len(records)-limit:]
	}
	return append([]TimingRecord(nil), records...)
}

func normalizeCompletedAt(completed time.Time) time.Time {
	if completed.IsZero() {
		return time.Now().UTC()
	}
	return completed.UTC()
}

func firstTimingValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
