// Package agent owns the native backend agent loop.
package agent

import (
	"context"

	"nexusdesk/internal/services/llm"
	settingssvc "nexusdesk/internal/services/settings"
)

type SettingsStore interface {
	Load() (settingssvc.Settings, error)
}

type ChatClient interface {
	Chat(ctx context.Context, config llm.Config, request llm.ChatRequest) (llm.ChatResult, error)
}

type ToolExecutor interface {
	ExecuteTool(ctx context.Context, call ToolCall, request Request) (ToolResult, error)
}

type ToolDescriber interface {
	ToolDescriptors() []ToolDescriptor
}

type ToolExecutorFunc func(ctx context.Context, call ToolCall, request Request) (ToolResult, error)

func (fn ToolExecutorFunc) ExecuteTool(ctx context.Context, call ToolCall, request Request) (ToolResult, error) {
	return fn(ctx, call, request)
}

type PlanStep struct {
	Step   string `json:"step"`
	Status string `json:"status"`
}

type ToolDescriptor struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Risk        string `json:"risk"`
	Inputs      string `json:"inputs"`
}

type ToolCall struct {
	Name      string            `json:"name"`
	Args      map[string]string `json:"args"`
	StartedAt string            `json:"startedAt"`
}

type ToolApprovalRequest struct {
	Name        string
	Args        map[string]string
	Risk        string
	Description string
}

type ToolApprover func(ctx context.Context, request ToolApprovalRequest) bool

type ToolResult struct {
	Name        string            `json:"name"`
	Args        map[string]string `json:"args"`
	Observation string            `json:"observation"`
	Error       string            `json:"error"`
	Risk        string            `json:"risk"`
	Mutated     bool              `json:"mutated"`
	StartedAt   string            `json:"startedAt"`
	CompletedAt string            `json:"completedAt"`
}

type Request struct {
	ID             string
	Prompt         string
	WorkspaceRoot  string
	ApproveWrites  bool
	ApproveShell   bool
	ApproveTool    ToolApprover
	Conversation   []llm.ChatTurn
	ContextRelPath string
	ContextContent string
	SourcePaths    []string
}

type Result struct {
	Message    string
	Plan       []PlanStep
	ToolCalls  []ToolResult
	Iterations int
	Truncated  bool
	StopReason string
}

type Event struct {
	RequestID   string
	Type        string
	Iteration   int
	Message     string
	Model       string
	ToolName    string
	ToolArgs    map[string]string
	Observation string
	Error       string
	Risk        string
	Plan        []PlanStep
}

type Observer func(Event)
