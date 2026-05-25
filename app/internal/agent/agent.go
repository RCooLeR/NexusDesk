package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"NexusAugenticStudio/internal/agenttools"
	"NexusAugenticStudio/internal/llm"
	"NexusAugenticStudio/internal/storage"
)

const (
	DefaultMaxIterations = 6
	maxObservationBytes  = 6000
	maxHistoryItems      = 10
)

type PlanStep struct {
	Step   string `json:"step"`
	Status string `json:"status"`
}

type ToolCall struct {
	Name        string            `json:"name"`
	Arguments   map[string]string `json:"arguments"`
	Observation string            `json:"observation"`
	Error       string            `json:"error"`
	Risk        string            `json:"risk"`
	StartedAt   string            `json:"startedAt"`
	CompletedAt string            `json:"completedAt"`
}

type RunRequest struct {
	Prompt             string `json:"prompt"`
	MaxIterations      int    `json:"maxIterations"`
	ApproveHighImpact  bool   `json:"approveHighImpact"`
	AllowShellCommands bool   `json:"allowShellCommands"`
}

type RunResult struct {
	Message    string     `json:"message"`
	Plan       []PlanStep `json:"plan"`
	ToolCalls  []ToolCall `json:"toolCalls"`
	Iterations int        `json:"iterations"`
	Truncated  bool       `json:"truncated"`
}

type ToolExecutor func(ctx context.Context, call ToolCall, request RunRequest) (ToolCall, error)

type Agent struct {
	llmClient *llm.Client
	llmStore  *storage.LLMSettingsStore
}

func New(client *llm.Client, store *storage.LLMSettingsStore) *Agent {
	return &Agent{llmClient: client, llmStore: store}
}

func (a *Agent) Run(ctx context.Context, request RunRequest, execute ToolExecutor) (RunResult, error) {
	request.Prompt = strings.TrimSpace(request.Prompt)
	if request.Prompt == "" {
		return RunResult{}, fmt.Errorf("agent prompt is required")
	}
	if execute == nil {
		return RunResult{}, fmt.Errorf("agent tool executor is required")
	}

	settings, err := a.llmStore.Get()
	if err != nil {
		return RunResult{}, err
	}

	maxIterations := request.MaxIterations
	if maxIterations <= 0 || maxIterations > 12 {
		maxIterations = DefaultMaxIterations
	}

	state := runState{
		userPrompt: request.Prompt,
		plan:       []PlanStep{{Step: "Understand the user request", Status: "in_progress"}},
		history:    []string{},
	}

	for iteration := 0; iteration < maxIterations; iteration++ {
		prompt := state.prompt()
		result, err := a.llmClient.Chat(ctx, settings, llm.ChatRequest{Prompt: prompt})
		if err != nil {
			return RunResult{}, err
		}

		message := strings.TrimSpace(result.Message)
		state.appendHistory("Assistant", message)

		if steps, ok := parsePlanUpdate(message); ok {
			state.plan = steps
		}

		if final := parseFinalAnswer(message); final != "" {
			state.finishPlan()
			return RunResult{
				Message:    final,
				Plan:       state.plan,
				ToolCalls:  state.toolCalls,
				Iterations: iteration + 1,
				Truncated:  state.truncated,
			}, nil
		}

		call, ok := parseAction(message)
		if !ok {
			state.finishPlan()
			return RunResult{
				Message:    message,
				Plan:       state.plan,
				ToolCalls:  state.toolCalls,
				Iterations: iteration + 1,
				Truncated:  state.truncated,
			}, nil
		}

		call.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
		completed, runErr := execute(ctx, call, request)
		completed.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if runErr != nil && completed.Error == "" {
			completed.Error = runErr.Error()
		}
		completed.Observation, state.truncated = truncateUTF8(completed.Observation, maxObservationBytes, state.truncated)
		state.toolCalls = append(state.toolCalls, completed)

		observation := completed.Observation
		if completed.Error != "" {
			observation = "ERROR: " + completed.Error
		}
		state.appendHistory("Observation", observation)
		state.pruneHistory()
	}

	return RunResult{
		Message:    "Agent reached the iteration limit before producing a final answer.",
		Plan:       state.plan,
		ToolCalls:  state.toolCalls,
		Iterations: maxIterations,
		Truncated:  state.truncated,
	}, nil
}

type runState struct {
	userPrompt string
	plan       []PlanStep
	history    []string
	toolCalls  []ToolCall
	truncated  bool
}

func (s *runState) prompt() string {
	var builder strings.Builder
	builder.WriteString(SystemPrompt())
	builder.WriteString("\n\nAvailable tools:\n")
	for _, descriptor := range agenttools.Registry() {
		builder.WriteString(fmt.Sprintf("- %s: %s Risk=%s Inputs=%s\n", descriptor.Name, descriptor.Description, descriptor.Risk, strings.Join(descriptor.Inputs, ", ")))
	}
	builder.WriteString("- list_directory: List workspace directory entries. Risk=low Inputs=relPath, recursive, maxDepth\n")
	builder.WriteString("- read_file: Read a bounded workspace file preview. Risk=low Inputs=relPath\n")
	builder.WriteString("- search_files: Search workspace file paths and text. Risk=low Inputs=query\n")
	builder.WriteString("- write_file: Preview or apply a text file write. Risk=high Inputs=relPath, content\n")
	builder.WriteString("- append_file: Preview or apply appending text to a file. Risk=high Inputs=relPath, content\n")
	builder.WriteString("- execute_shell_command: Run a shell command inside the workspace. Risk=high Inputs=command\n")
	builder.WriteString("- analyze_csv_excel: Profile/query a dataset. Risk=low Inputs=relPath, query\n")
	builder.WriteString("- generate_artifact: Create a deterministic Markdown artifact. Risk=low Inputs=sourcePath\n")
	builder.WriteString("- update_plan: Replace visible plan steps. Risk=low Inputs=steps\n\n")
	builder.WriteString("Respond with one of these forms only:\n")
	builder.WriteString("Thought: ...\nAction: tool_name({\"key\":\"value\"})\n")
	builder.WriteString("or\nFinal Answer: ...\n\n")
	builder.WriteString("Current plan:\n")
	for _, step := range s.plan {
		builder.WriteString(fmt.Sprintf("- [%s] %s\n", step.Status, step.Step))
	}
	builder.WriteString("\nUser request:\n")
	builder.WriteString(s.userPrompt)
	builder.WriteString("\n\nRecent working memory:\n")
	for _, item := range s.history {
		builder.WriteString(item)
		builder.WriteString("\n")
	}
	return builder.String()
}

func (s *runState) appendHistory(role string, content string) {
	content, s.truncated = truncateUTF8(strings.TrimSpace(content), maxObservationBytes, s.truncated)
	if content == "" {
		return
	}
	s.history = append(s.history, role+": "+content)
}

func (s *runState) pruneHistory() {
	if len(s.history) <= maxHistoryItems {
		return
	}
	s.history = append([]string{"Earlier context summarized: older observations were pruned after tool execution to keep the local model focused."}, s.history[len(s.history)-maxHistoryItems:]...)
	s.truncated = true
}

func (s *runState) finishPlan() {
	for index, step := range s.plan {
		if step.Status == "in_progress" {
			s.plan[index].Status = "completed"
		}
	}
}

func SystemPrompt() string {
	return strings.Join([]string{
		"You are Nexus Agent, the local-first assistant inside Nexus Augentic Studio for code, data, documents, operations, and artifacts.",
		"Work as a permissioned co-pilot: inspect before acting, keep all actions inside the active workspace, and never claim access to files you did not inspect.",
		"Use a ReAct loop with concise Thought, a single Action, the resulting Observation, and then a Final Answer when the task is done.",
		"Use update_plan for multi-step work and keep exactly one step in_progress.",
		"High-impact tools such as writes, shell commands, deletes, moves, and Docker actions require explicit approval. If approval is missing, explain the proposed action instead of pretending it ran.",
		"Prefer existing deterministic tools over free-form guesses. Keep answers grounded in tool observations and source paths.",
	}, "\n")
}

func parseAction(message string) (ToolCall, bool) {
	re := regexp.MustCompile(`(?is)Action:\s*([a-zA-Z0-9_.-]+)\s*\((.*)\)\s*$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(message))
	if len(matches) != 3 {
		return ToolCall{}, false
	}

	args := map[string]string{}
	rawArgs := strings.TrimSpace(matches[2])
	if strings.HasPrefix(rawArgs, "{") {
		decoded := map[string]any{}
		if err := json.Unmarshal([]byte(rawArgs), &decoded); err == nil {
			for key, value := range decoded {
				args[key] = fmt.Sprint(value)
			}
			return ToolCall{Name: strings.TrimSpace(matches[1]), Arguments: args}, true
		}
	}

	for _, pair := range splitArgs(rawArgs) {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		args[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return ToolCall{Name: strings.TrimSpace(matches[1]), Arguments: args}, true
}

func parseFinalAnswer(message string) string {
	re := regexp.MustCompile(`(?is)Final Answer:\s*(.*)$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(message))
	if len(matches) != 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func parsePlanUpdate(message string) ([]PlanStep, bool) {
	re := regexp.MustCompile(`(?is)Action:\s*update_plan\s*\((.*)\)\s*$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(message))
	if len(matches) != 2 {
		return nil, false
	}
	raw := strings.TrimSpace(matches[1])
	payload := struct {
		Steps []PlanStep `json:"steps"`
	}{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, false
	}
	if len(payload.Steps) == 0 {
		return nil, false
	}
	return normalizePlan(payload.Steps), true
}

func normalizePlan(steps []PlanStep) []PlanStep {
	normalized := make([]PlanStep, 0, len(steps))
	inProgressSeen := false
	for _, step := range steps {
		step.Step = strings.TrimSpace(step.Step)
		step.Status = strings.TrimSpace(step.Status)
		if step.Step == "" {
			continue
		}
		switch step.Status {
		case "pending", "completed":
		case "in_progress":
			if inProgressSeen {
				step.Status = "pending"
			}
			inProgressSeen = true
		default:
			step.Status = "pending"
		}
		normalized = append(normalized, step)
	}
	if len(normalized) == 0 {
		return []PlanStep{{Step: "Work on the user request", Status: "in_progress"}}
	}
	return normalized
}

func splitArgs(raw string) []string {
	parts := []string{}
	var builder strings.Builder
	inQuotes := false
	var quote rune
	for _, char := range raw {
		if (char == '\'' || char == '"') && (quote == 0 || quote == char) {
			inQuotes = !inQuotes
			if inQuotes {
				quote = char
			} else {
				quote = 0
			}
		}
		if char == ',' && !inQuotes {
			parts = append(parts, builder.String())
			builder.Reset()
			continue
		}
		builder.WriteRune(char)
	}
	if builder.Len() > 0 {
		parts = append(parts, builder.String())
	}
	return parts
}

func truncateUTF8(value string, maxBytes int, alreadyTruncated bool) (string, bool) {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value, alreadyTruncated
	}
	truncated := value[:maxBytes]
	for !utf8.ValidString(truncated) && len(truncated) > 0 {
		truncated = truncated[:len(truncated)-1]
	}
	return truncated + "\n[truncated]", true
}
