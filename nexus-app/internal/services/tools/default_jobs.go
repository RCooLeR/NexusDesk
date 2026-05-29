package tools

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"nexusdesk/internal/services/agent"
	jobsSvc "nexusdesk/internal/services/jobs"
)

const (
	defaultJobListLimit = 20
	maxJobListLimit     = 100
	defaultJobLogLines  = 12
	maxJobLogLines      = 50
	defaultJobLogBytes  = 16 * 1024
	maxJobLogBytes      = 64 * 1024
)

func (h defaultHandlers) listJobs(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	statusFilter := strings.ToLower(strings.TrimSpace(firstArg(call, "status")))
	limit := boundedJobInt(call, "limit", defaultJobListLimit, maxJobListLimit)
	jobs := h.deps.Jobs.List()
	lines := []string{fmt.Sprintf("Recent jobs: showing up to %d job(s).", limit)}
	shown := 0
	for _, job := range jobs {
		if statusFilter != "" && strings.ToLower(string(job.Status)) != statusFilter {
			continue
		}
		lines = append(lines, formatJobSummary(job))
		shown++
		if shown >= limit {
			break
		}
	}
	if shown == 0 {
		if statusFilter == "" {
			lines = append(lines, "No jobs are recorded.")
		} else {
			lines = append(lines, fmt.Sprintf("No jobs match status %q.", statusFilter))
		}
	}
	return toolOK(call, "low", strings.Join(lines, "\n")), nil
}

func (h defaultHandlers) readJobLogs(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	jobID := firstArg(call, "jobId", "id")
	if jobID == "" {
		err := errors.New("jobId is required")
		return toolError(call, "low", err), err
	}
	job, ok := h.deps.Jobs.Get(jobID)
	if !ok {
		err := fmt.Errorf("job %q was not found", jobID)
		return toolError(call, "low", err), err
	}
	tailLines := boundedJobInt(call, "tailLines", defaultJobLogLines, maxJobLogLines)
	tailBytes := boundedJobInt(call, "tailBytes", defaultJobLogBytes, maxJobLogBytes)
	logLines := tailJobLines(job.LogTail, tailLines)
	logText := redactJobText(strings.Join(logLines, "\n"))
	logText = tailUTF8Bytes(logText, tailBytes)
	if strings.TrimSpace(logText) == "" {
		logText = "No log lines are available for this job."
	}
	lines := []string{
		formatJobSummary(job),
		fmt.Sprintf("Log tail: %d line(s), capped at %d byte(s).", len(logLines), tailBytes),
		"",
		logText,
	}
	return toolOK(call, "low", strings.Join(lines, "\n")), nil
}

func (h defaultHandlers) cancelJob(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	if !request.ApproveWrites {
		err := errors.New("approval is required before canceling jobs")
		return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "high", Observation: err.Error(), Error: err.Error()}, err
	}
	jobID := firstArg(call, "jobId", "id")
	if jobID == "" {
		err := errors.New("jobId is required")
		return toolError(call, "high", err), err
	}
	job, ok := h.deps.Jobs.Get(jobID)
	if !ok {
		err := fmt.Errorf("job %q was not found", jobID)
		return toolError(call, "high", err), err
	}
	if job.Status != jobsSvc.StatusRunning {
		err := fmt.Errorf("job %s is %s and cannot be canceled", job.ID, job.Status)
		return toolError(call, "high", err), err
	}
	if !h.deps.Jobs.Cancel(jobID) {
		err := fmt.Errorf("job %s could not be canceled", jobID)
		return toolError(call, "high", err), err
	}
	canceled, _ := h.deps.Jobs.Get(jobID)
	return agent.ToolResult{
		Name:        call.Name,
		Args:        call.Args,
		Risk:        "high",
		Mutated:     true,
		Observation: "Cancel requested for durable job.\n" + formatJobSummary(canceled),
	}, nil
}

func formatJobSummary(job jobsSvc.Job) string {
	fields := []string{
		fmt.Sprintf("- %s [%s/%s] %s", job.ID, job.Kind, job.Status, redactJobText(job.Label)),
	}
	if job.Message != "" {
		fields = append(fields, "message="+redactJobText(job.Message))
	}
	if job.Error != "" {
		fields = append(fields, "error="+redactJobText(job.Error))
	}
	if !job.StartedAt.IsZero() {
		fields = append(fields, "started="+formatJobTime(job.StartedAt))
	}
	if !job.CompletedAt.IsZero() {
		fields = append(fields, "completed="+formatJobTime(job.CompletedAt))
	}
	return strings.Join(fields, " ")
}

func formatJobTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func tailJobLines(lines []string, limit int) []string {
	if limit <= 0 || len(lines) <= limit {
		return append([]string(nil), lines...)
	}
	return append([]string(nil), lines[len(lines)-limit:]...)
}

func boundedJobInt(call agent.ToolCall, key string, fallback int, maxValue int) int {
	value := intArg(call, key, fallback)
	if value <= 0 {
		return fallback
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func tailUTF8Bytes(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	start := len(value) - limit
	for start < len(value) && !utf8.RuneStart(value[start]) {
		start++
	}
	return "[log truncated]\n" + value[start:]
}

func redactJobText(value string) string {
	result := value
	for _, pattern := range jobSecretPatterns {
		result = pattern.re.ReplaceAllString(result, pattern.replacement)
	}
	return result
}

var jobSecretPatterns = []struct {
	re          *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9._~+/=-]+`), "Bearer [redacted]"},
	{regexp.MustCompile(`sk-[A-Za-z0-9._-]+`), "sk-[redacted]"},
	{regexp.MustCompile(`(?i)(api[_-]?key\s*[:=]\s*)[^\s,"']+`), `${1}[redacted]`},
	{regexp.MustCompile(`(?i)(token\s*[:=]\s*)[^\s,"']+`), `${1}[redacted]`},
	{regexp.MustCompile(`(?i)(password\s*[:=]\s*)[^\s,"']+`), `${1}[redacted]`},
	{regexp.MustCompile(`(?i)(secret\s*[:=]\s*)[^\s,"']+`), `${1}[redacted]`},
	{regexp.MustCompile(`(?i)("(?:api[_-]?key|token|access_token|auth_token|password|secret)"\s*:\s*)"[^"]*"`), `${1}"[redacted]"`},
	{regexp.MustCompile(`(?i)\b(api[_-]?key|token|access_token|auth_token|password|secret)\s*:\s*('[^']*'|"[^"]*"|[^\s,;]+)`), `${1}: [redacted]`},
	{regexp.MustCompile(`(?i)(Authorization\s*[:=]\s*)[^\n\r]+`), `${1}[redacted]`},
	{regexp.MustCompile(`(?i)(postgres|mysql|sqlserver)://([^:\s/@]+):([^@\s]+)@`), `${1}://${2}:[redacted]@`},
}
