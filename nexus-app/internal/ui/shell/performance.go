package shell

import (
	"fmt"
	"strings"
	"time"

	perfSvc "nexusdesk/internal/services/perf"
)

func (v *View) RecordStartupReady(started time.Time, detail string) {
	v.recordPerformanceTiming(perfSvc.TimingStartupReady, started, perfSvc.StartupReadyBudget, detail)
}

func (v *View) recordPerformanceTiming(name string, started time.Time, budget time.Duration, detail string) perfSvc.TimingRecord {
	if v == nil || v.performanceRecorder == nil {
		return perfSvc.TimingRecord{}
	}
	return v.performanceRecorder.Record(name, started, time.Now().UTC(), budget, detail)
}

func (v *View) performanceTimings(limit int) []perfSvc.TimingRecord {
	if v == nil || v.performanceRecorder == nil {
		return nil
	}
	return v.performanceRecorder.Snapshot(limit)
}

func formatPerformanceTiming(record perfSvc.TimingRecord) string {
	status := "ok"
	if !record.WithinBudget {
		status = "over budget"
	}
	detail := strings.TrimSpace(record.Detail)
	if detail == "" {
		detail = "no detail"
	}
	if record.Budget <= 0 {
		return fmt.Sprintf("%s: %s (%s) - %s", record.Name, roundTimingDuration(record.Duration), status, detail)
	}
	return fmt.Sprintf("%s: %s of %s budget (%s) - %s", record.Name, roundTimingDuration(record.Duration), roundTimingDuration(record.Budget), status, detail)
}

func performanceTimingWarning(record perfSvc.TimingRecord) string {
	return fmt.Sprintf("Performance timing over budget: %s took %s (budget %s).", record.Name, roundTimingDuration(record.Duration), roundTimingDuration(record.Budget))
}

func hasOverBudgetPerformanceTiming(records []perfSvc.TimingRecord) bool {
	for _, record := range records {
		if !record.WithinBudget {
			return true
		}
	}
	return false
}

func roundTimingDuration(value time.Duration) time.Duration {
	if value <= 0 {
		return 0
	}
	if value < time.Millisecond {
		return value
	}
	return value.Round(time.Millisecond)
}
