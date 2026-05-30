package agent

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxObservationBytes     = 24_000
	maxEventBytes           = 2_000
	maxHistoryTokens        = 4_096
	approxCharsPerToken     = 4
	backendEmergencyGuard   = 64
	defaultRunTimeout       = 45 * time.Minute
	StopReasonTimeout       = "timeout"
	stopReasonSafetyGuard   = "safety_guard"
	stopReasonSafetyWrapped = "safety_guard_finalized"
)

func DefaultRunTimeout() time.Duration {
	return defaultRunTimeout
}

func EffectiveRunTimeout(request Request) time.Duration {
	if request.RunTimeout > 0 {
		return request.RunTimeout
	}
	return defaultRunTimeout
}

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
	s.packHistory(maxHistoryTokens)
}

func (s *runState) packHistory(tokenBudget int) {
	if tokenBudget <= 0 {
		if len(s.history) > 0 {
			s.truncated = true
		}
		s.history = nil
		return
	}
	remaining := tokenBudget
	packed := []string{}
	dropped := 0
	for index := len(s.history) - 1; index >= 0; index-- {
		item := strings.TrimSpace(s.history[index])
		if item == "" {
			continue
		}
		tokens := approxTokenCount(item)
		if tokens <= remaining {
			packed = append(packed, item)
			remaining -= tokens
			continue
		}
		if len(packed) == 0 {
			limited, truncated := truncateUTF8(item, remaining*approxCharsPerToken, true)
			if strings.TrimSpace(limited) != "" {
				packed = append(packed, limited)
			}
			s.truncated = s.truncated || truncated
			dropped = index
			break
		}
		dropped = index + 1
		break
	}
	reverseStrings(packed)
	if dropped > 0 {
		s.truncated = true
		note := fmt.Sprintf("History packing: omitted %d older observation(s) to fit the approximate %d-token context budget.", dropped, tokenBudget)
		packed = append([]string{note}, packed...)
	}
	s.history = packed
}

func approxTokenCount(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	tokens := len(value) / approxCharsPerToken
	if len(value)%approxCharsPerToken != 0 {
		tokens++
	}
	if tokens < 1 {
		return 1
	}
	return tokens
}

func reverseStrings(values []string) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
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
