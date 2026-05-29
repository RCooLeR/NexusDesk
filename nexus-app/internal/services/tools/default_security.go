package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"nexusdesk/internal/services/agent"
	approvalsSvc "nexusdesk/internal/services/approvals"
)

const redactTextMaxBytes = 256 * 1024

func (h defaultHandlers) redactText(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	_ = ctx
	_ = request
	content := firstArg(call, "content", "text", "value")
	if content == "" {
		err := errors.New("content is required")
		return toolError(call, "low", err), err
	}
	if len(content) > redactTextMaxBytes {
		err := errors.New("content is too large to redact in one tool call")
		return toolError(call, "low", err), err
	}
	redacted := redactSensitiveText(content)
	status := "No sensitive patterns detected."
	if redacted != content {
		status = "Sensitive patterns were redacted."
	}
	return toolOK(call, "low", fmt.Sprintf("%s\nBytes: %d\n\n%s", status, len(redacted), redacted)), nil
}

func (h defaultHandlers) listApprovals(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	_ = ctx
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "low", err), err
	}
	limit := intArg(call, "limit", 20)
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	decision := strings.ToLower(strings.TrimSpace(firstArg(call, "decision", "status")))
	records, err := h.deps.Approvals.List(root)
	if err != nil {
		return toolError(call, "low", err), err
	}
	policy, _ := h.deps.Approvals.LoadPolicy(root)

	lines := []string{formatApprovalPolicyLine(policy)}
	count := 0
	for _, record := range records {
		if decision != "" && strings.ToLower(strings.TrimSpace(record.Decision)) != decision {
			continue
		}
		if count >= limit {
			lines = append(lines, "[approval records truncated]")
			break
		}
		lines = append(lines, formatApprovalRecordLine(record))
		count++
	}
	if count == 0 {
		lines = append(lines, "No approval records matched.")
	}
	return toolOK(call, "low", strings.Join(lines, "\n")), nil
}

func (h defaultHandlers) requestApproval(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	_ = ctx
	root, err := workspaceRoot(request)
	if err != nil {
		return toolError(call, "medium", err), err
	}
	action := firstArg(call, "action", "name")
	if action == "" {
		err := errors.New("action is required")
		return toolError(call, "medium", err), err
	}
	risk := strings.ToLower(strings.TrimSpace(firstArg(call, "risk")))
	if risk == "" {
		risk = "high"
	}
	switch risk {
	case "low", "medium", "high":
	default:
		err := errors.New("risk must be low, medium, or high")
		return toolError(call, "medium", err), err
	}
	target := firstArg(call, "target", "relPath", "path")
	summary := firstArg(call, "summary", "message", "description")
	if summary == "" {
		summary = "Agent requested explicit approval before continuing."
	}
	records, err := h.deps.Approvals.Append(root, approvalsSvc.Record{
		Action:   "tool.approval.request." + action,
		Target:   redactSensitiveText(target),
		Risk:     risk,
		Decision: "requested",
		Message:  redactSensitiveText(summary),
	})
	if err != nil {
		return toolError(call, "medium", err), err
	}
	record := records[0]
	observation := strings.Join([]string{
		"Approval request recorded.",
		"ID: " + record.ID,
		"Action: " + record.Action,
		"Risk: " + record.Risk,
		"Target: " + redactSensitiveText(record.Target),
		"Message: " + redactSensitiveText(record.Message),
	}, "\n")
	return agent.ToolResult{Name: call.Name, Args: call.Args, Risk: "medium", Observation: observation, Mutated: true}, nil
}

func redactSensitiveText(value string) string {
	return redactJobText(value)
}

func formatApprovalPolicyLine(policy approvalsSvc.Policy) string {
	if policy.Active(time.Now().UTC()) {
		return fmt.Sprintf("Policy: full project access active until %s.", policy.ExpiresAt.UTC().Format("2006-01-02 15:04:05Z"))
	}
	if strings.TrimSpace(policy.Message) != "" {
		return "Policy: " + redactSensitiveText(policy.Message)
	}
	return "Policy: guarded per-call approvals."
}

func formatApprovalRecordLine(record approvalsSvc.Record) string {
	created := ""
	if !record.CreatedAt.IsZero() {
		created = " created=" + record.CreatedAt.UTC().Format("2006-01-02 15:04:05Z")
	}
	fields := []string{
		"- " + record.ID,
		"action=" + redactSensitiveText(record.Action),
		"risk=" + record.Risk,
		"decision=" + record.Decision,
	}
	if target := strings.TrimSpace(record.Target); target != "" {
		fields = append(fields, "target="+redactSensitiveText(target))
	}
	if message := strings.TrimSpace(record.Message); message != "" {
		fields = append(fields, "message="+redactSensitiveText(message))
	}
	return strings.Join(fields, " ") + created
}
