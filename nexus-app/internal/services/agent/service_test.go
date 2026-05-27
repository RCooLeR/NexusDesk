package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"nexusdesk/internal/services/llm"
	settingssvc "nexusdesk/internal/services/settings"
)

func TestParseJSONActionKeepsNestedArguments(t *testing.T) {
	call, ok := parseAction(`Thought: inspect.
Action: query_dataset({"relPath":"data.csv","filter":{"column":"channel","value":"Search"},"columns":["spend","conversions"]})`)
	if !ok {
		t.Fatal("expected action")
	}
	if call.Name != "query_dataset" {
		t.Fatalf("unexpected tool name: %q", call.Name)
	}
	if call.Args["filter"] != `{"column":"channel","value":"Search"}` {
		t.Fatalf("nested filter = %q", call.Args["filter"])
	}
	if call.Args["columns"] != `["spend","conversions"]` {
		t.Fatalf("columns = %q", call.Args["columns"])
	}
}

func TestParsePlanUpdateNormalizesOneInProgress(t *testing.T) {
	steps, ok := parsePlanUpdate(`Action: update_plan({"steps":[{"step":"Inspect","status":"in_progress"},{"step":"Patch","status":"in_progress"}]})`)
	if !ok {
		t.Fatal("expected plan update")
	}
	if steps[0].Status != "in_progress" || steps[1].Status != "pending" {
		t.Fatalf("unexpected plan: %#v", steps)
	}
}

func TestRunExecutesToolThenFinalAnswer(t *testing.T) {
	model := &fakeChatClient{messages: []string{
		`Thought: inspect.
Action: read_context({"relPath":"README.md"})`,
		`Final Answer: README says hello.`,
	}}
	executor := ToolExecutorFunc(func(ctx context.Context, call ToolCall, request Request) (ToolResult, error) {
		if call.Name != "read_context" || call.Args["relPath"] != "README.md" {
			t.Fatalf("unexpected call: %#v", call)
		}
		return ToolResult{Name: call.Name, Args: call.Args, Observation: "hello from README", Risk: "low"}, nil
	})
	service := New(fakeSettingsStore{}, model, executor)
	events := []Event{}
	result, err := service.Run(context.Background(), Request{ID: "run-1", Prompt: "Check README"}, func(event Event) {
		events = append(events, event)
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Message != "README says hello." || len(result.ToolCalls) != 1 || result.Iterations != 2 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(events) == 0 || events[len(events)-1].Type != "final" {
		t.Fatalf("expected final event, got %#v", events)
	}
	if !strings.Contains(model.prompts[1], "hello from README") {
		t.Fatalf("second prompt did not include observation:\n%s", model.prompts[1])
	}
}

func TestRunPromptIncludesRegisteredToolDescriptors(t *testing.T) {
	model := &fakeChatClient{messages: []string{`Final Answer: Done.`}}
	service := New(fakeSettingsStore{}, model, fakeDescribingExecutor{})
	_, err := service.Run(context.Background(), Request{Prompt: "Use tools"}, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(model.prompts[0], "Registered deterministic tools") ||
		!strings.Contains(model.prompts[0], "read_context") {
		t.Fatalf("prompt missing descriptors:\n%s", model.prompts[0])
	}
}

func TestRunHandlesUpdatePlanWithoutExecutor(t *testing.T) {
	model := &fakeChatClient{messages: []string{
		`Action: update_plan({"steps":[{"step":"Inspect","status":"completed"},{"step":"Answer","status":"in_progress"}]})`,
		`Final Answer: Done.`,
	}}
	service := New(fakeSettingsStore{}, model, ToolExecutorFunc(func(ctx context.Context, call ToolCall, request Request) (ToolResult, error) {
		t.Fatalf("update_plan should not reach executor")
		return ToolResult{}, nil
	}))
	result, err := service.Run(context.Background(), Request{Prompt: "Plan"}, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(result.Plan) != 2 || result.Plan[1].Status != "completed" {
		t.Fatalf("unexpected final plan: %#v", result.Plan)
	}
}

func TestRunAddsVerificationNoteForUnobservedMutationClaim(t *testing.T) {
	model := &fakeChatClient{messages: []string{`Final Answer: I created gemma-findings.md.`}}
	service := New(fakeSettingsStore{}, model, ToolExecutorFunc(func(ctx context.Context, call ToolCall, request Request) (ToolResult, error) {
		return ToolResult{}, errors.New("not used")
	}))
	result, err := service.Run(context.Background(), Request{Prompt: "Create file"}, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(result.Message, "Verification note") {
		t.Fatalf("expected verification note, got %q", result.Message)
	}
}

type fakeSettingsStore struct{}

func (fakeSettingsStore) Load() (settingssvc.Settings, error) {
	return settingssvc.Settings{Provider: "test", BaseURL: "http://localhost/v1", Model: "test-model", ContextTokens: 4096, ResponseReserveTokens: 512}, nil
}

type fakeChatClient struct {
	messages []string
	prompts  []string
}

func (c *fakeChatClient) Chat(ctx context.Context, config llm.Config, request llm.ChatRequest) (llm.ChatResult, error) {
	c.prompts = append(c.prompts, request.Prompt)
	if len(c.messages) == 0 {
		return llm.ChatResult{}, errors.New("no fake messages left")
	}
	message := c.messages[0]
	c.messages = c.messages[1:]
	return llm.ChatResult{Message: message, Model: config.Model}, nil
}

type fakeDescribingExecutor struct{}

func (fakeDescribingExecutor) ExecuteTool(ctx context.Context, call ToolCall, request Request) (ToolResult, error) {
	return ToolResult{}, nil
}

func (fakeDescribingExecutor) ToolDescriptors() []ToolDescriptor {
	return []ToolDescriptor{{Name: "read_context", Description: "Read context", Risk: "low", Inputs: "relPath"}}
}
