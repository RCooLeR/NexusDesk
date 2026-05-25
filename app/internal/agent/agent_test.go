package agent

import (
	"strings"
	"testing"
)

func TestParseJSONAction(t *testing.T) {
	call, ok := parseAction(`Thought: inspect file.
Action: read_file({"relPath":"docs/README.md"})`)
	if !ok {
		t.Fatal("expected action")
	}
	if call.Name != "read_file" {
		t.Fatalf("tool name = %q", call.Name)
	}
	if call.Arguments["relPath"] != "docs/README.md" {
		t.Fatalf("relPath = %q", call.Arguments["relPath"])
	}
}

func TestParseKeyValueAction(t *testing.T) {
	call, ok := parseAction(`Action: search_files(query="agent plan")`)
	if !ok {
		t.Fatal("expected action")
	}
	if call.Name != "search_files" || call.Arguments["query"] != "agent plan" {
		t.Fatalf("unexpected call: %#v", call)
	}
}

func TestParseFinalAnswer(t *testing.T) {
	answer := parseFinalAnswer("Thought: done\nFinal Answer: The workspace is ready.")
	if answer != "The workspace is ready." {
		t.Fatalf("answer = %q", answer)
	}
}

func TestParsePlanUpdate(t *testing.T) {
	steps, ok := parsePlanUpdate(`Action: update_plan({"steps":[{"step":"Inspect","status":"completed"},{"step":"Patch","status":"in_progress"},{"step":"Verify","status":"pending"}]})`)
	if !ok {
		t.Fatal("expected plan update")
	}
	if len(steps) != 3 || steps[1].Status != "in_progress" {
		t.Fatalf("unexpected steps: %#v", steps)
	}
}

func TestNormalizePlanAllowsOneInProgress(t *testing.T) {
	steps := normalizePlan([]PlanStep{
		{Step: "A", Status: "in_progress"},
		{Step: "B", Status: "in_progress"},
	})
	if steps[0].Status != "in_progress" || steps[1].Status != "pending" {
		t.Fatalf("unexpected statuses: %#v", steps)
	}
}

func TestFinalizationPromptForbidsMoreActions(t *testing.T) {
	state := runState{
		userPrompt: "Check the project",
		plan: []PlanStep{
			{Step: "Inspect", Status: "completed"},
			{Step: "Summarize", Status: "in_progress"},
		},
		toolCalls: []ToolCall{{Name: "list_directory", Observation: "README.md\napp/"}},
		history:   []string{"Observation: README.md"},
	}

	prompt := state.finalizationPrompt()
	if !strings.Contains(prompt, "Do not request more tools") || !strings.Contains(prompt, "Final Answer:") {
		t.Fatalf("finalization prompt does not force a final answer:\n%s", prompt)
	}
	if !strings.Contains(prompt, "list_directory") || !strings.Contains(prompt, "Check the project") {
		t.Fatalf("finalization prompt is missing run context:\n%s", prompt)
	}
}

func TestStoppedRunMessageIncludesToolCount(t *testing.T) {
	message := stoppedRunMessage(&runState{toolCalls: []ToolCall{{Name: "read_file"}}})
	if !strings.Contains(message, "1 tool call") || strings.Contains(message, "1 tool calls") {
		t.Fatalf("unexpected stopped run message: %q", message)
	}
}
