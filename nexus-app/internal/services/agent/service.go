package agent

import (
	"context"
	"errors"
	"strings"
	"time"

	"nexusdesk/internal/services/llm"
	settingssvc "nexusdesk/internal/services/settings"
)

type Service struct {
	settingsStore SettingsStore
	client        ChatClient
	executor      ToolExecutor
}

func New(settingsStore SettingsStore, client ChatClient, executor ToolExecutor) *Service {
	return &Service{settingsStore: settingsStore, client: client, executor: executor}
}

func (s *Service) Run(ctx context.Context, request Request, observe Observer) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithTimeout(ctx, EffectiveRunTimeout(request))
	defer cancel()
	request.Prompt = strings.TrimSpace(request.Prompt)
	if request.Prompt == "" {
		return Result{}, errors.New("agent prompt is required")
	}
	if s.settingsStore == nil {
		return Result{}, errors.New("agent settings store is required")
	}
	if s.client == nil {
		return Result{}, errors.New("agent LLM client is required")
	}
	if s.executor == nil {
		return Result{}, errors.New("agent tool executor is required")
	}
	settings, err := s.settingsStore.Load()
	if err != nil {
		return Result{}, err
	}
	settings, routeInfo := settingsForAgentRequest(settings, request)
	config := llm.ConfigFromSettings(settings)
	tools := s.toolDescriptors()
	state := runState{plan: []PlanStep{{Step: "Understand the request", Status: "in_progress"}}}
	s.emit(observe, request, eventWithRoute(Event{
		Type:    "start",
		Message: agentStartMessage(routeInfo),
		Model:   config.Model,
		Plan:    state.plan,
	}, routeInfo))

	for iteration := 1; ; iteration++ {
		if deadlineExceeded(runCtx) {
			return s.timeoutResult(config, request, state, routeInfo, observe, iteration-1), nil
		}
		if iteration > backendEmergencyGuard {
			return s.wrapUpStoppedRun(runCtx, config, request, state, routeInfo, observe, iteration-1)
		}
		s.emit(observe, request, Event{Type: "model_request", Iteration: iteration, Message: "Asking model for the next step.", Plan: state.plan})
		result, err := s.client.Chat(runCtx, config, llm.ChatRequest{
			SystemPrompt:   systemPrompt(),
			Prompt:         runtimePrompt(request, state, tools),
			Conversation:   request.Conversation,
			ContextRelPath: request.ContextRelPath,
			ContextContent: request.ContextContent,
			SourcePaths:    request.SourcePaths,
		})
		if err != nil {
			if deadlineExceeded(runCtx) {
				return s.timeoutResult(config, request, state, routeInfo, observe, iteration), nil
			}
			s.emit(observe, request, Event{Type: "error", Iteration: iteration, Message: "Model request failed.", Error: err.Error()})
			return Result{}, err
		}
		message := strings.TrimSpace(result.Message)
		s.emit(observe, request, Event{Type: "model_response", Iteration: iteration, Message: limitText(message, maxEventBytes), Model: result.Model})

		if steps, ok := parsePlanUpdate(message); ok {
			state.plan = steps
			state.appendHistory("Plan", "Plan updated.")
			s.emit(observe, request, Event{Type: "plan_update", Iteration: iteration, Message: "Plan updated.", Plan: state.plan})
			continue
		}
		if final := parseFinalAnswer(message); final != "" {
			state.plan = finishPlan(state.plan)
			final = appendMutationVerification(final, state)
			s.emit(observe, request, Event{Type: "final", Iteration: iteration, Message: limitText(final, maxEventBytes), Plan: state.plan})
			return resultWithRoute(Result{Message: final, Model: result.Model, Plan: state.plan, ToolCalls: state.toolCalls, Iterations: iteration, Truncated: state.truncated}, routeInfo), nil
		}
		call, ok := parseAction(message)
		if !ok {
			state.plan = finishPlan(state.plan)
			message = appendMutationVerification(message, state)
			s.emit(observe, request, Event{Type: "final", Iteration: iteration, Message: limitText(message, maxEventBytes), Plan: state.plan})
			return resultWithRoute(Result{Message: message, Model: result.Model, Plan: state.plan, ToolCalls: state.toolCalls, Iterations: iteration, Truncated: state.truncated}, routeInfo), nil
		}
		completed := s.executeTool(runCtx, request, call, iteration, observe)
		state.toolCalls = append(state.toolCalls, completed)
		observation := completed.Observation
		if completed.Error != "" {
			observation = "ERROR: " + completed.Error
		}
		state.appendHistory("Observation", observation)
		if deadlineExceeded(runCtx) {
			return s.timeoutResult(config, request, state, routeInfo, observe, iteration), nil
		}
	}
}

func (s *Service) toolDescriptors() []ToolDescriptor {
	describer, ok := s.executor.(ToolDescriber)
	if !ok {
		return nil
	}
	descriptors := describer.ToolDescriptors()
	return append([]ToolDescriptor{}, descriptors...)
}

func (s *Service) executeTool(ctx context.Context, request Request, call ToolCall, iteration int, observe Observer) ToolResult {
	startedAt := time.Now().UTC().Format(time.RFC3339Nano)
	call.StartedAt = startedAt
	s.emit(observe, request, Event{Type: "tool_start", Iteration: iteration, Message: "Tool requested.", ToolName: call.Name, ToolArgs: call.Args})
	result, err := s.executor.ExecuteTool(ctx, call, request)
	if result.Name == "" {
		result.Name = call.Name
	}
	if result.Args == nil {
		result.Args = call.Args
	}
	result.StartedAt = startedAt
	result.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err != nil && result.Error == "" {
		result.Error = err.Error()
	}
	result.Observation, _ = truncateUTF8(result.Observation, maxObservationBytes, false)
	eventType := "tool_done"
	if result.Error != "" {
		eventType = "tool_error"
	}
	s.emit(observe, request, Event{
		Type:        eventType,
		Iteration:   iteration,
		Message:     "Tool completed.",
		ToolName:    result.Name,
		ToolArgs:    result.Args,
		Observation: limitText(result.Observation, maxEventBytes),
		Error:       result.Error,
		Risk:        result.Risk,
	})
	return result
}

func (s *Service) wrapUpStoppedRun(ctx context.Context, config llm.Config, request Request, state runState, routeInfo agentRouteResolution, observe Observer, iterations int) (Result, error) {
	state.plan = finishPlan(state.plan)
	s.emit(observe, request, Event{Type: "finalizing", Iteration: iterations, Message: "Backend safety guard stopped the tool loop; asking for a final answer.", Plan: state.plan})
	result, err := s.client.Chat(ctx, config, llm.ChatRequest{SystemPrompt: systemPrompt(), Prompt: finalizationPrompt(request, state)})
	if err != nil {
		message := "Agent stopped before producing a final answer."
		s.emit(observe, request, Event{Type: "stopped", Iteration: iterations, Message: message, Error: err.Error(), Plan: state.plan})
		return resultWithRoute(Result{Message: message, Model: config.Model, Plan: state.plan, ToolCalls: state.toolCalls, Iterations: iterations, Truncated: state.truncated, StopReason: stopReasonSafetyGuard}, routeInfo), nil
	}
	message := strings.TrimSpace(result.Message)
	if final := parseFinalAnswer(message); final != "" {
		message = final
	}
	message = appendMutationVerification(message, state)
	s.emit(observe, request, Event{Type: "stopped_finalized", Iteration: iterations, Message: limitText(message, maxEventBytes), Model: result.Model, Plan: state.plan})
	return resultWithRoute(Result{Message: message, Model: result.Model, Plan: state.plan, ToolCalls: state.toolCalls, Iterations: iterations, Truncated: state.truncated, StopReason: stopReasonSafetyWrapped}, routeInfo), nil
}

func (s *Service) timeoutResult(config llm.Config, request Request, state runState, routeInfo agentRouteResolution, observe Observer, iterations int) Result {
	state.plan = finishPlan(state.plan)
	timeout := EffectiveRunTimeout(request)
	message := "Agent run timed out."
	if timeout > 0 {
		message = "Agent run timed out after " + timeout.String() + "."
	}
	s.emit(observe, request, Event{Type: "stopped", Iteration: iterations, Message: message, Error: context.DeadlineExceeded.Error(), Plan: state.plan})
	return resultWithRoute(Result{
		Message:    message,
		Model:      config.Model,
		Plan:       state.plan,
		ToolCalls:  state.toolCalls,
		Iterations: iterations,
		Truncated:  state.truncated,
		StopReason: StopReasonTimeout,
	}, routeInfo)
}

func deadlineExceeded(ctx context.Context) bool {
	return ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded)
}

func (s *Service) emit(observe Observer, request Request, event Event) {
	if observe == nil {
		return
	}
	event.RequestID = request.ID
	observe(event)
}

type agentRouteResolution struct {
	ID      string
	Label   string
	Warning string
}

func settingsForAgentRequest(settings settingssvc.Settings, request Request) (settingssvc.Settings, agentRouteResolution) {
	routeID := strings.TrimSpace(request.ModelRouteID)
	if routeID == "" {
		return settings, agentRouteResolution{}
	}
	route, ok := settingssvc.ModelRouteByID(settings, routeID)
	if !ok {
		return settings, agentRouteResolution{
			ID:      routeID,
			Warning: "Model route " + routeID + " was not found; using the global model.",
		}
	}
	routed, ok := settingssvc.SettingsForModelRoute(settings, routeID)
	if !ok {
		return settings, agentRouteResolution{
			ID:      routeID,
			Label:   route.Label,
			Warning: "Model route " + firstNonEmptyString(route.Label, routeID) + " could not be resolved; using the global model.",
		}
	}
	return routed, agentRouteResolution{ID: route.ID, Label: route.Label}
}

func resultWithRoute(result Result, route agentRouteResolution) Result {
	result.ModelRouteID = route.ID
	result.ModelRoute = route.Label
	result.RouteWarning = route.Warning
	return result
}

func eventWithRoute(event Event, route agentRouteResolution) Event {
	event.ModelRouteID = route.ID
	event.ModelRoute = route.Label
	event.RouteWarning = route.Warning
	return event
}

func agentStartMessage(route agentRouteResolution) string {
	if route.Warning != "" {
		return "Agent started. " + route.Warning
	}
	if route.Label != "" {
		return "Agent started with " + route.Label + "."
	}
	return "Agent started."
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
