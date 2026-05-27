package agent

import (
	"regexp"
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
	mutated   bool
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

func claimsMutation(message string) bool {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return false
	}
	negative := []string{
		"did not create", "didn't create", "could not create", "cannot create", "can't create",
		"did not write", "didn't write", "could not write", "cannot write", "can't write",
		"did not save", "could not save", "cannot save", "can't save",
		"not created", "not written", "not saved",
	}
	for _, phrase := range negative {
		if strings.Contains(normalized, phrase) {
			return false
		}
	}
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\b(i|i have|i've|we|we have|we've)\s+(created|wrote|written|saved|updated|modified|generated|documented|recorded|added)\b`),
		regexp.MustCompile(`\b(file|artifact|document|report)\s+.*\b(created|written|saved|updated|generated)\b`),
	}
	for _, pattern := range patterns {
		if pattern.MatchString(normalized) {
			return true
		}
	}
	return false
}

func guardMutationClaim(message string, mutated bool) string {
	if mutated || !claimsMutation(message) {
		return message
	}
	return strings.TrimSpace(message) + "\n\nVerification note: this agent run did not receive a successful mutating tool observation, so any claimed workspace write should be treated as unverified."
}
