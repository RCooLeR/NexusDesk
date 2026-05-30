package agent

import (
	"strings"
	"testing"
)

func TestRunStatePacksHistoryByApproximateTokens(t *testing.T) {
	state := runState{}
	for index := 0; index < 12; index++ {
		state.appendHistory("Observation", strings.Repeat("older ", 900)+string(rune('a'+index)))
	}
	if !state.truncated {
		t.Fatal("expected packed history to mark truncation")
	}
	joined := strings.Join(state.history, "\n")
	if !strings.Contains(joined, "History packing: omitted") {
		t.Fatalf("expected packing note, got %#v", state.history)
	}
	if approxTokenCount(joined) > maxHistoryTokens+128 {
		t.Fatalf("history exceeded budget by too much: tokens=%d history=%d item(s)", approxTokenCount(joined), len(state.history))
	}
	if !strings.Contains(joined, "older ") {
		t.Fatalf("expected newest observations to be retained: %#v", state.history)
	}
}

func TestRunStateTruncatesOversizedNewestHistoryItem(t *testing.T) {
	state := runState{}
	state.appendHistory("Observation", strings.Repeat("x", maxHistoryTokens*approxCharsPerToken*2))
	if len(state.history) != 1 || !state.truncated {
		t.Fatalf("expected one truncated history item, got %#v truncated=%v", state.history, state.truncated)
	}
	if approxTokenCount(state.history[0]) > maxHistoryTokens+16 {
		t.Fatalf("expected oversized item to be bounded, got %d tokens", approxTokenCount(state.history[0]))
	}
}
