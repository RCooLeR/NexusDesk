package shell

import (
	"time"

	agentSvc "nexusdesk/internal/services/agent"
	metadataSvc "nexusdesk/internal/services/metadata"
)

func (v *View) persistAgentRun(jobID string, request agentSvc.Request, result agentSvc.Result, status string, message string, startedAt time.Time) {
	if v.metadataStore == nil {
		return
	}
	completedAt := time.Now().UTC()
	record := metadataSvc.AgentRunRecord{
		ID:          request.ID,
		JobID:       jobID,
		Prompt:      request.Prompt,
		Status:      status,
		Message:     message,
		Iterations:  result.Iterations,
		StopReason:  result.StopReason,
		Plan:        agentPlanForMetadata(result.Plan),
		SourcePaths: append([]string{}, request.SourcePaths...),
		StartedAt:   startedAt,
		CompletedAt: completedAt,
		DurationMs:  completedAt.Sub(startedAt).Milliseconds(),
	}
	if err := v.metadataStore.SaveAgentRun(record); err != nil {
		v.addActivity("Could not persist agent run: " + err.Error())
		return
	}
	for index, call := range result.ToolCalls {
		if err := v.metadataStore.SaveToolRun(toolRunForMetadata(record, call, index+1)); err != nil {
			v.addActivity("Could not persist tool run: " + err.Error())
			return
		}
	}
	v.addActivity("Persisted agent audit for " + request.ID + ".")
	v.refreshAgentAudit()
}

func agentPlanForMetadata(plan []agentSvc.PlanStep) []metadataSvc.AgentPlanStep {
	steps := make([]metadataSvc.AgentPlanStep, 0, len(plan))
	for _, step := range plan {
		steps = append(steps, metadataSvc.AgentPlanStep{Step: step.Step, Status: step.Status})
	}
	return steps
}

func toolRunForMetadata(run metadataSvc.AgentRunRecord, call agentSvc.ToolResult, sequence int) metadataSvc.ToolRunRecord {
	return metadataSvc.ToolRunRecord{
		AgentRunID:  run.ID,
		JobID:       run.JobID,
		Sequence:    sequence,
		ToolName:    call.Name,
		Risk:        call.Risk,
		Mutated:     call.Mutated,
		Args:        copyToolArgs(call.Args),
		Observation: call.Observation,
		Error:       call.Error,
		StartedAt:   parseAgentTime(call.StartedAt),
		CompletedAt: parseAgentTime(call.CompletedAt),
	}
}

func copyToolArgs(args map[string]string) map[string]string {
	if len(args) == 0 {
		return nil
	}
	copied := make(map[string]string, len(args))
	for key, value := range args {
		copied[key] = value
	}
	return copied
}

func parseAgentTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}
