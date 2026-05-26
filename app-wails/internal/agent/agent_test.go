package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"NexusAugenticStudio/internal/llm"
	"NexusAugenticStudio/internal/storage"
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

func TestParseJSONActionKeepsNestedArgumentsAsJSON(t *testing.T) {
	call, ok := parseAction(`Action: dataset.query({"relPath":"data.csv","filter":{"column":"channel","value":"Search"},"columns":["spend","conversions"]})`)
	if !ok {
		t.Fatal("expected action")
	}
	if call.Arguments["filter"] != `{"column":"channel","value":"Search"}` {
		t.Fatalf("expected nested object JSON, got %q", call.Arguments["filter"])
	}
	if call.Arguments["columns"] != `["spend","conversions"]` {
		t.Fatalf("expected nested array JSON, got %q", call.Arguments["columns"])
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

func TestPromptExplainsWriteFileCreateContract(t *testing.T) {
	state := runState{userPrompt: "Create docs/review.md"}
	prompt := state.prompt()
	for _, expected := range []string{
		"write_file: Create or replace a text/code/config/document file",
		"read_context: Build a bounded context pack",
		"read_git_diff: Read git status",
		"read_git_history: Read bounded Git commit history",
		"read_git_blame: Read bounded Git blame attribution",
		"read_problems: Run lightweight workspace diagnostics",
		"list_tasks: List discovered safe workspace tasks",
		"list_artifacts: List generated workspace artifacts",
		"read_artifact: Read a bounded artifact preview",
		"run_task: Run a discovered workspace task",
		"write_binary_file: Create or replace a binary file",
		"apply_patch: Apply a unified diff",
		"copy_file: Copy one workspace file",
		"move_file: Move or rename one workspace file",
		"delete_file: Delete one workspace file",
		"list_rollbacks: List rollback snapshots",
		"rollback_file_mutation: Restore or remove files",
		"To create or replace text/code/config/Markdown/JSON/source files",
		"Never say a file was created",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt missing %q:\n%s", expected, prompt)
		}
	}
}

func TestSystemPromptMentionsWriteTools(t *testing.T) {
	prompt := SystemPrompt()
	if !strings.Contains(prompt, "write_file") ||
		!strings.Contains(prompt, "read_context") ||
		!strings.Contains(prompt, "read_git_diff") ||
		!strings.Contains(prompt, "read_git_history") ||
		!strings.Contains(prompt, "read_git_blame") ||
		!strings.Contains(prompt, "read_problems") ||
		!strings.Contains(prompt, "list_tasks") ||
		!strings.Contains(prompt, "list_artifacts") ||
		!strings.Contains(prompt, "read_artifact") ||
		!strings.Contains(prompt, "run_task") ||
		!strings.Contains(prompt, "write_binary_file") ||
		!strings.Contains(prompt, "apply_patch") ||
		!strings.Contains(prompt, "append_file") ||
		!strings.Contains(prompt, "copy_file") ||
		!strings.Contains(prompt, "move_file") ||
		!strings.Contains(prompt, "delete_file") ||
		!strings.Contains(prompt, "rollback_file_mutation") {
		t.Fatalf("system prompt does not mention write tools:\n%s", prompt)
	}
}

func TestStoppedRunMessageIncludesToolCount(t *testing.T) {
	message := stoppedRunMessage(&runState{toolCalls: []ToolCall{{Name: "read_file"}}})
	if !strings.Contains(message, "1 tool call") || strings.Contains(message, "1 tool calls") {
		t.Fatalf("unexpected stopped run message: %q", message)
	}
	if strings.Contains(strings.ToLower(message), "iteration") {
		t.Fatalf("stopped run message leaks implementation wording: %q", message)
	}
}

func TestFitsModelContextUsesResponseReserve(t *testing.T) {
	settings := storage.LLMSettings{MaxContextTokens: 4096, ResponseReserveTokens: 2048}
	if !fitsModelContext(strings.Repeat("x", 1000), settings) {
		t.Fatal("expected small prompt to fit")
	}
	if fitsModelContext(strings.Repeat("x", 8000), settings) {
		t.Fatal("expected large prompt to exceed model context")
	}
}

func TestGuardUnverifiedSideEffectClaimsWarnsWithoutMutatingTool(t *testing.T) {
	state := runState{toolCalls: []ToolCall{{Name: "read_file", Observation: "README.md"}}}
	message := state.guardUnverifiedSideEffectClaims("I created gemma-findings.md with the review findings.")
	if !strings.Contains(message, "Verification warning") {
		t.Fatalf("expected verification warning, got %q", message)
	}
}

func TestGuardUnverifiedSideEffectClaimsWarnsForDocumentedInNewFile(t *testing.T) {
	state := runState{toolCalls: []ToolCall{{Name: "search_files", Observation: "No matches"}}}
	message := state.guardUnverifiedSideEffectClaims("All findings have been documented in the new file gemma-findings.md.")
	if !strings.Contains(message, "Verification warning") {
		t.Fatalf("expected verification warning, got %q", message)
	}
}

func TestGuardUnverifiedSideEffectClaimsAllowsSuccessfulWriteTool(t *testing.T) {
	state := runState{toolCalls: []ToolCall{{Name: "write_file", Observation: "Wrote docs/review.md"}}}
	message := state.guardUnverifiedSideEffectClaims("I created docs/review.md with the review findings.")
	if strings.Contains(message, "Verification warning") {
		t.Fatalf("did not expect verification warning, got %q", message)
	}
}

func TestGuardUnverifiedSideEffectClaimsIgnoresNegativeClaims(t *testing.T) {
	state := runState{}
	message := state.guardUnverifiedSideEffectClaims("I did not create a file because approval was missing.")
	if strings.Contains(message, "Verification warning") {
		t.Fatalf("did not expect verification warning for negative claim, got %q", message)
	}
}

func TestRequestsWorkspaceWriteDetectsNewMarkdownFile(t *testing.T) {
	prompt := "Review the project. List issues, bugs, gaps you find in the entire project to new file gemma-findings.md"
	if !requestsPersistentWorkspaceChange(prompt) {
		t.Fatalf("expected prompt to request workspace write")
	}
}

func TestRequestsPersistentWorkspaceChangeDetectsCodeChanges(t *testing.T) {
	for _, prompt := range []string{
		"Fix the broken sidebar layout in the frontend",
		"Implement the next tracker item in code",
		"Refactor the backend agent module",
	} {
		if !requestsPersistentWorkspaceChange(prompt) {
			t.Fatalf("expected prompt to request workspace mutation: %q", prompt)
		}
	}
}

func TestNeedsRequestedWriteToolRequiresApprovedWriteResult(t *testing.T) {
	state := runState{
		userPrompt:          "Document the review in gemma-findings.md",
		writeAccessApproved: true,
		toolCalls:           []ToolCall{{Name: "read_file", Observation: "README.md"}},
	}
	if !state.needsRequestedMutationTool() {
		t.Fatal("expected missing requested write to require a write tool")
	}
	state.toolCalls = append(state.toolCalls, ToolCall{Name: "write_file", Observation: "Wrote gemma-findings.md"})
	if state.needsRequestedMutationTool() {
		t.Fatal("did not expect write requirement after successful write_file")
	}
}

func TestNeedsRequestedMutationToolAcceptsBinaryWrite(t *testing.T) {
	state := runState{
		userPrompt:          "Create icon.png",
		writeAccessApproved: true,
		toolCalls:           []ToolCall{{Name: "write_binary_file", Observation: "Wrote icon.png"}},
	}
	if state.needsRequestedMutationTool() {
		t.Fatal("did not expect write requirement after successful write_binary_file")
	}
}

func TestNeedsRequestedMutationToolAcceptsApplyPatch(t *testing.T) {
	state := runState{
		userPrompt:          "Fix the frontend sidebar layout",
		writeAccessApproved: true,
		toolCalls:           []ToolCall{{Name: "apply_patch", Observation: "Applied unified patch to 2 file(s)."}},
	}
	if state.needsRequestedMutationTool() {
		t.Fatal("did not expect write requirement after successful apply_patch")
	}
}

func TestNeedsRequestedMutationToolAcceptsMoveAndDelete(t *testing.T) {
	for _, call := range []ToolCall{
		{Name: "move_file", Observation: "Moved docs/a.md to docs/b.md"},
		{Name: "delete_file", Observation: "Deleted docs/a.md from the workspace."},
	} {
		state := runState{
			userPrompt:          "Move or delete docs/a.md",
			writeAccessApproved: true,
			toolCalls:           []ToolCall{call},
		}
		if state.needsRequestedMutationTool() {
			t.Fatalf("did not expect write requirement after successful %s", call.Name)
		}
	}
}

func TestRunRejectsPrematureFinalAnswerForApprovedWriteRequest(t *testing.T) {
	responses := []string{
		"Final Answer: I have completed the project review. All findings have been documented in the new file gemma-findings.md.",
		`Thought: I need to create the requested file.
Action: write_file({"relPath":"gemma-findings.md","content":"# Findings\n\n- Agent write verification gap."})`,
		"Final Answer: Created gemma-findings.md with the review findings.",
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if len(responses) == 0 {
			t.Fatal("unexpected extra model request")
		}
		response.Header().Set("Content-Type", "application/json")
		payload := map[string]any{
			"choices": []map[string]any{{
				"message": map[string]string{"role": "assistant", "content": responses[0]},
			}},
		}
		responses = responses[1:]
		if err := json.NewEncoder(response).Encode(payload); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	store := storage.NewLLMSettingsStore(t.TempDir() + "/llm-settings.json")
	if _, err := store.Save(storage.LLMSettings{
		ProviderName:          "test",
		BaseURL:               server.URL + "/v1",
		Model:                 "test-model",
		MaxContextTokens:      32768,
		ResponseReserveTokens: 4096,
	}); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	var calls []ToolCall
	runner := New(llm.NewClient(), store)
	result, err := runner.Run(context.Background(), RunRequest{
		Prompt:            "Review the project. List issues, bugs, gaps you find in the entire project to new file gemma-findings.md",
		ApproveHighImpact: true,
	}, func(ctx context.Context, call ToolCall, request RunRequest) (ToolCall, error) {
		calls = append(calls, call)
		call.Observation = "Wrote gemma-findings.md"
		return call, nil
	}, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(calls) != 1 || calls[0].Name != "write_file" {
		t.Fatalf("expected one write_file call after rejected final answer, got %#v", calls)
	}
	if len(responses) != 0 {
		t.Fatalf("expected all model responses to be consumed, remaining %d", len(responses))
	}
	if !strings.Contains(result.Message, "Created gemma-findings.md") {
		t.Fatalf("unexpected final answer: %q", result.Message)
	}
	if strings.Contains(result.Message, "Verification warning") {
		t.Fatalf("did not expect verification warning after write tool: %q", result.Message)
	}
}
