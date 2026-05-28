package agent

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxObservationBytes     = 24_000
	maxEventBytes           = 2_000
	maxHistoryItems         = 10
	backendEmergencyGuard   = 64
	stopReasonSafetyGuard   = "safety_guard"
	stopReasonSafetyWrapped = "safety_guard_finalized"
)

type runState struct {
	plan      []PlanStep
	toolCalls []ToolResult
	history   []string
	truncated bool
}

func (s *runState) appendHistory(label string, content string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}
	value, truncated := truncateUTF8(label+": "+content, maxObservationBytes, s.truncated)
	s.truncated = truncated
	s.history = append(s.history, value)
	if len(s.history) > maxHistoryItems {
		s.history = s.history[len(s.history)-maxHistoryItems:]
	}
}

func truncateUTF8(value string, maxBytes int, alreadyTruncated bool) (string, bool) {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value, alreadyTruncated
	}
	cut := value[:maxBytes]
	for !utf8.ValidString(cut) && len(cut) > 0 {
		cut = cut[:len(cut)-1]
	}
	return cut + "\n[truncated]", true
}

func limitText(value string, maxBytes int) string {
	limited, _ := truncateUTF8(strings.TrimSpace(value), maxBytes, false)
	return limited
}

func appendMutationVerification(message string, state runState) string {
	message = strings.TrimSpace(message)
	note := mutationVerificationNote(state)
	if note == "" {
		return message
	}
	if message == "" {
		return note
	}
	return message + "\n\n" + note
}

func mutationVerificationNote(state runState) string {
	successes := []string{}
	attempts := 0
	for _, call := range state.toolCalls {
		if call.Mutated {
			successes = append(successes, mutationToolSummary(call))
			continue
		}
		if isMutationTool(call.Name) {
			attempts++
		}
	}
	if len(successes) > 0 {
		total := len(successes)
		if total > 5 {
			successes = append(successes[:5], fmt.Sprintf("+%d more", total-5))
		}
		return fmt.Sprintf("Mutation verification: verified %d successful workspace mutation(s) from tool observation(s): %s.", total, strings.Join(successes, "; "))
	}
	if attempts > 0 {
		return fmt.Sprintf("Mutation verification: no successful workspace mutation was observed; %d mutation-capable tool attempt(s) failed or were blocked.", attempts)
	}
	if len(state.toolCalls) > 0 {
		return "Mutation verification: no workspace mutation was reported by the completed tool observations."
	}
	return "Mutation verification: no tool observation was recorded, so no workspace mutation is verified."
}

func isMutationTool(name string) bool {
	switch strings.TrimSpace(name) {
	case "write_file", "append_file", "copy_file", "move_file", "delete_file", "apply_patch", "rollback_file_mutation":
		return true
	default:
		return false
	}
}

func mutationToolSummary(call ToolResult) string {
	target := firstMutationArg(call.Args, "relPath", "targetRelPath", "sourceRelPath", "id", "path", "target", "source")
	if target == "" {
		return call.Name
	}
	return call.Name + " " + target
}

func firstMutationArg(args map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(args[key]); value != "" {
			return value
		}
	}
	return ""
}
