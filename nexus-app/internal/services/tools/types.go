// Package tools owns the native deterministic tool dispatcher.
package tools

import (
	"context"

	"nexusdesk/internal/services/agent"
)

type Handler func(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error)

type Tool struct {
	Descriptor agent.ToolDescriptor
	Handler    Handler
}
