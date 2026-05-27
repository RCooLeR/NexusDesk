package agent

import "strings"

func normalizePlan(steps []PlanStep) []PlanStep {
	normalized := make([]PlanStep, 0, len(steps))
	inProgressSeen := false
	for _, step := range steps {
		label := strings.TrimSpace(step.Step)
		if label == "" {
			continue
		}
		status := normalizeStatus(step.Status)
		if status == "in_progress" {
			if inProgressSeen {
				status = "pending"
			}
			inProgressSeen = true
		}
		normalized = append(normalized, PlanStep{Step: label, Status: status})
	}
	return normalized
}

func normalizeStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed":
		return "completed"
	case "in_progress", "in progress":
		return "in_progress"
	default:
		return "pending"
	}
}

func finishPlan(steps []PlanStep) []PlanStep {
	if len(steps) == 0 {
		return nil
	}
	finished := make([]PlanStep, len(steps))
	copy(finished, steps)
	for index := range finished {
		if finished[index].Status == "in_progress" {
			finished[index].Status = "completed"
		}
	}
	return finished
}
