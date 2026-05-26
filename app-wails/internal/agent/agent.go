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
	maxObservationBytes       = 24000
	maxHistoryItems           = 10
	agentPromptSafetyTokens   = 512
	agentEmergencyStepGuard   = 256
	defaultAgentContextTokens = 32768

	stopReasonContextLimit          = "context_limit"
	stopReasonContextLimitFinalized = "context_limit_finalized"
	stopReasonSafetyGuard           = "safety_guard"
	stopReasonSafetyGuardFinalized  = "safety_guard_finalized"
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
	RequestID          string `json:"requestId"`
	Prompt             string `json:"prompt"`
	ApproveHighImpact  bool   `json:"approveHighImpact"`
	AllowShellCommands bool   `json:"allowShellCommands"`
}

type RunResult struct {
	Message    string     `json:"message"`
	Plan       []PlanStep `json:"plan"`
	ToolCalls  []ToolCall `json:"toolCalls"`
	Iterations int        `json:"iterations"`
	Truncated  bool       `json:"truncated"`
	StopReason string     `json:"stopReason,omitempty"`
}

type ToolExecutor func(ctx context.Context, call ToolCall, request RunRequest) (ToolCall, error)

type RunEvent struct {
	RequestID   string            `json:"requestId"`
	Type        string            `json:"type"`
	Iteration   int               `json:"iteration"`
	Message     string            `json:"message"`
	Model       string            `json:"model"`
	ToolName    string            `json:"toolName"`
	ToolArgs    map[string]string `json:"toolArgs"`
	Observation string            `json:"observation"`
	Error       string            `json:"error"`
	Risk        string            `json:"risk"`
	Plan        []PlanStep        `json:"plan,omitempty"`
	Timestamp   string            `json:"timestamp"`
}

type RunObserver func(RunEvent)

type Agent struct {
	llmClient *llm.Client
	llmStore  *storage.LLMSettingsStore
}

func New(client *llm.Client, store *storage.LLMSettingsStore) *Agent {
	return &Agent{llmClient: client, llmStore: store}
}

func (a *Agent) Run(ctx context.Context, request RunRequest, execute ToolExecutor, observe RunObserver) (RunResult, error) {
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
	settings, err = a.llmStore.ResolveForUse(settings)
	if err != nil {
		return RunResult{}, err
	}

	state := runState{
		userPrompt:          request.Prompt,
		writeAccessApproved: request.ApproveHighImpact,
		shellAccessAllowed:  request.AllowShellCommands && request.ApproveHighImpact,
		plan:                []PlanStep{{Step: "Understand the user request", Status: "in_progress"}},
		history:             []string{},
	}
	emitRunEvent(request, observe, RunEvent{Type: "start", Message: "Agent started.", Plan: state.plan})

	for iteration := 0; ; iteration++ {
		prompt := state.prompt()
		if !fitsModelContext(prompt, settings) {
			return a.wrapUpStoppedRun(ctx, settings, &state, request, observe, iteration, stopReasonContextLimit, stopReasonContextLimitFinalized, "The selected model context is full. The agent is wrapping up with completed results.")
		}
		if iteration >= agentEmergencyStepGuard {
			return a.wrapUpStoppedRun(ctx, settings, &state, request, observe, iteration, stopReasonSafetyGuard, stopReasonSafetyGuardFinalized, "The local agent safety guard stopped a repeated tool loop. The agent is wrapping up with completed results.")
		}

		emitRunEvent(request, observe, RunEvent{Type: "model_request", Iteration: iteration + 1, Message: fmt.Sprintf("Step %d: asking model for the next step.", iteration+1), Plan: state.plan})
		result, err := a.llmClient.Chat(ctx, settings, llm.ChatRequest{Prompt: prompt})
		if err != nil {
			emitRunEvent(request, observe, RunEvent{Type: "error", Iteration: iteration + 1, Error: err.Error(), Message: "Model request failed."})
			return RunResult{}, err
		}

		message := strings.TrimSpace(result.Message)
		state.appendHistory("Assistant", message)
		emitRunEvent(request, observe, RunEvent{Type: "model_response", Iteration: iteration + 1, Message: limitEventText(message, 2000), Model: result.Model})

		if steps, ok := parsePlanUpdate(message); ok {
			state.plan = steps
			emitRunEvent(request, observe, RunEvent{Type: "plan_update", Iteration: iteration + 1, Message: "Plan updated.", Plan: state.plan})
		}

		if final := parseFinalAnswer(message); final != "" {
			if state.needsRequestedMutationTool() {
				state.rejectPrematureWriteAnswer()
				emitRunEvent(request, observe, RunEvent{Type: "model_response", Iteration: iteration + 1, Message: "Model tried to finish before completing the requested workspace change; requiring a write tool.", Model: result.Model})
				continue
			}
			state.finishPlan()
			final = state.guardUnverifiedSideEffectClaims(final)
			emitRunEvent(request, observe, RunEvent{Type: "final", Iteration: iteration + 1, Message: limitEventText(final, 2000), Plan: state.plan})
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
			if state.needsRequestedMutationTool() {
				state.rejectPrematureWriteAnswer()
				emitRunEvent(request, observe, RunEvent{Type: "model_response", Iteration: iteration + 1, Message: "Model responded without the requested workspace change; requiring a write tool.", Model: result.Model})
				continue
			}
			state.finishPlan()
			message = state.guardUnverifiedSideEffectClaims(message)
			emitRunEvent(request, observe, RunEvent{Type: "final", Iteration: iteration + 1, Message: limitEventText(message, 2000), Plan: state.plan})
			return RunResult{
				Message:    message,
				Plan:       state.plan,
				ToolCalls:  state.toolCalls,
				Iterations: iteration + 1,
				Truncated:  state.truncated,
			}, nil
		}

		call.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
		emitRunEvent(request, observe, RunEvent{Type: "tool_start", Iteration: iteration + 1, Message: "Tool requested.", ToolName: call.Name, ToolArgs: call.Arguments})
		completed, runErr := execute(ctx, call, request)
		completed.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if runErr != nil && completed.Error == "" {
			completed.Error = runErr.Error()
		}
		completed.Observation, state.truncated = truncateUTF8(completed.Observation, maxObservationBytes, state.truncated)
		state.toolCalls = append(state.toolCalls, completed)
		eventType := "tool_done"
		if completed.Error != "" {
			eventType = "tool_error"
		}
		emitRunEvent(request, observe, RunEvent{
			Type:        eventType,
			Iteration:   iteration + 1,
			Message:     "Tool completed.",
			ToolName:    completed.Name,
			ToolArgs:    completed.Arguments,
			Observation: limitEventText(completed.Observation, 2000),
			Error:       completed.Error,
			Risk:        completed.Risk,
		})

		observation := completed.Observation
		if completed.Error != "" {
			observation = "ERROR: " + completed.Error
		}
		state.appendHistory("Observation", observation)
		state.pruneHistory()
	}
}

func (a *Agent) wrapUpStoppedRun(ctx context.Context, settings storage.LLMSettings, state *runState, request RunRequest, observe RunObserver, iterations int, stopReason string, finalizedStopReason string, eventMessage string) (RunResult, error) {
	state.finishPlan()
	emitRunEvent(request, observe, RunEvent{Type: "finalizing", Message: eventMessage, Plan: state.plan})
	message, finalized := a.finalizeStoppedRun(ctx, settings, state, request, observe)
	if finalized {
		stopReason = finalizedStopReason
	}
	return RunResult{
		Message:    message,
		Plan:       state.plan,
		ToolCalls:  state.toolCalls,
		Iterations: iterations,
		Truncated:  state.truncated,
		StopReason: stopReason,
	}, nil
}

func (a *Agent) finalizeStoppedRun(ctx context.Context, settings storage.LLMSettings, state *runState, request RunRequest, observe RunObserver) (string, bool) {
	result, err := a.llmClient.Chat(ctx, settings, llm.ChatRequest{Prompt: state.finalizationPrompt()})
	if err != nil {
		emitRunEvent(request, observe, RunEvent{Type: "stopped", Message: "Agent paused while wrapping up.", Error: err.Error(), Plan: state.plan})
		return stoppedRunMessage(state), false
	}

	message := strings.TrimSpace(result.Message)
	if final := parseFinalAnswer(message); final != "" {
		final = state.guardUnverifiedSideEffectClaims(final)
		emitRunEvent(request, observe, RunEvent{Type: "stopped_finalized", Message: limitEventText(final, 2000), Model: result.Model, Plan: state.plan})
		return final, true
	}
	if message != "" && !strings.Contains(strings.ToLower(message), "action:") {
		message = state.guardUnverifiedSideEffectClaims(message)
		emitRunEvent(request, observe, RunEvent{Type: "stopped_finalized", Message: limitEventText(message, 2000), Model: result.Model, Plan: state.plan})
		return message, true
	}
	emitRunEvent(request, observe, RunEvent{Type: "stopped", Message: "Agent paused while wrapping up.", Model: result.Model, Plan: state.plan})
	return stoppedRunMessage(state), false
}

type runState struct {
	userPrompt          string
	writeAccessApproved bool
	shellAccessAllowed  bool
	plan                []PlanStep
	history             []string
	toolCalls           []ToolCall
	truncated           bool
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
	builder.WriteString("- read_context: Build a bounded context pack from one or more workspace files/directories/root. Risk=low Inputs=relPaths\n")
	builder.WriteString("- read_git_diff: Read git status and staged/unstaged diffs, optionally for one file. Risk=low Inputs=relPath(optional)\n")
	builder.WriteString("- read_changed_files: Read changed-file status plus bounded previews for current Git working tree files. Risk=low Inputs=maxFiles(optional), includeContent(optional)\n")
	builder.WriteString("- read_git_history: Read bounded Git commit history for the repository or a single file. Risk=low Inputs=relPath(optional), limit(optional)\n")
	builder.WriteString("- read_git_blame: Read bounded Git blame attribution for one file or line range. Risk=low Inputs=relPath, startLine(optional), endLine(optional)\n")
	builder.WriteString("- read_problems: Run lightweight workspace diagnostics for TODO/FIXME/HACK/BUG markers, merge conflicts, and invalid JSON. Risk=low Inputs=maxResults(optional)\n")
	builder.WriteString("- list_tasks: List discovered safe workspace tasks such as npm scripts, Go tests, and Docker Compose config checks. Risk=low Inputs=none\n")
	builder.WriteString("- list_artifacts: List generated workspace artifacts with metadata. Risk=low Inputs=none\n")
	builder.WriteString("- read_artifact: Read a bounded artifact preview and metadata. Risk=low Inputs=relPath\n")
	builder.WriteString("- read_artifact_lineage: Read the artifact/source/chat/tool relationship graph as bounded context. Risk=low Inputs=none\n")
	builder.WriteString("- web_fetch: Fetch one approved HTTP(S) text-like URL with redirect, size, content-type, local-network, and optional domain allow-list guards. Risk=medium Inputs=url, allowedDomains(optional), allowLocal(optional), maxBytes(optional)\n")
	builder.WriteString("- list_datasets: List persisted dataset profiles. Risk=low Inputs=none\n")
	builder.WriteString("- profile_dataset: Profile a CSV/TSV/JSON/NDJSON/XLSX/Parquet/log file and return schema/profile metadata. Risk=low Inputs=relPath\n")
	builder.WriteString("- query_dataset: Query a CSV/TSV/JSON/NDJSON dataset with text/column/order/limit filters. Risk=low Inputs=relPath, query(optional)\n")
	builder.WriteString("- query_dataset_sql: Run a read-only SELECT query against a dataset using DuckDB when available. Risk=low Inputs=relPath, sql\n")
	builder.WriteString("- inspect_sqlite: Inspect a workspace SQLite database schema, indexes, relationships, and samples in read-only mode. Risk=low Inputs=relPath\n")
	builder.WriteString("- query_sqlite: Run a bounded read-only SELECT/WITH query against a workspace SQLite database. Risk=low Inputs=relPath, sql, resultLimit(optional), timeoutSeconds(optional)\n")
	builder.WriteString("- inspect_operations: Read a bounded Dockerfile, Compose, environment, script, config, or log file for operations analysis; environment-like secrets are redacted. Risk=low Inputs=relPath\n")
	builder.WriteString("- read_document_set: Read bounded text context from Markdown, TXT, PDF, DOCX, HTML, and XML files or folders. Risk=low Inputs=relPaths, maxFiles(optional)\n")
	builder.WriteString("- write_file: Create or replace a text/code/config/document file inside the workspace. Risk=high Inputs=relPath, content, encoding(optional)\n")
	builder.WriteString("- write_binary_file: Create or replace a binary file from standard base64 bytes inside the workspace. Risk=high Inputs=relPath, base64Content, contentType(optional)\n")
	builder.WriteString("- apply_patch: Apply a unified diff across one or more text/code files. Risk=high Inputs=patch\n")
	builder.WriteString("- append_file: Append text to a workspace file without replacing existing content; creates the file if it does not exist. Risk=high Inputs=relPath, content, encoding(optional)\n")
	builder.WriteString("- copy_file: Copy one workspace file to a new workspace path. Risk=high Inputs=sourceRelPath, targetRelPath\n")
	builder.WriteString("- move_file: Move or rename one workspace file to a new workspace path. Risk=high Inputs=sourceRelPath, targetRelPath\n")
	builder.WriteString("- delete_file: Delete one workspace file. Risk=high Inputs=relPath\n")
	builder.WriteString("- list_rollbacks: List rollback snapshots from approved workspace file mutations. Risk=low Inputs=none\n")
	builder.WriteString("- rollback_file_mutation: Restore or remove files from a rollback snapshot. Risk=high Inputs=id\n")
	builder.WriteString("- run_task: Run a discovered workspace task by taskId through the safe task runner. Risk=high Inputs=taskId\n")
	builder.WriteString("- execute_shell_command: Run a shell command inside the workspace. Risk=high Inputs=command\n")
	builder.WriteString("- analyze_csv_excel: Profile/query a dataset. Risk=low Inputs=relPath, query\n")
	builder.WriteString("- generate_artifact: Create a deterministic Markdown artifact. Risk=low Inputs=sourcePath\n")
	builder.WriteString("- update_plan: Replace visible plan steps. Risk=low Inputs=steps\n\n")
	builder.WriteString("Respond with one of these forms only:\n")
	builder.WriteString("Thought: ...\nAction: tool_name({\"key\":\"value\"})\n")
	builder.WriteString("or\nFinal Answer: ...\n\n")
	builder.WriteString("Write contract:\n")
	builder.WriteString("- To understand a feature, folder, or whole project before editing, call read_context with relPaths such as [\"src\"] or [\".\"]. It returns a capped context pack selected by the workspace context builder.\n")
	builder.WriteString("- To understand current uncommitted work, call read_git_diff before making related edits or summaries.\n")
	builder.WriteString("- To inspect current changed files as file content rather than diffs, call read_changed_files with a bounded maxFiles value.\n")
	builder.WriteString("- To understand when or why committed code changed, call read_git_history for commit context and read_git_blame for line attribution.\n")
	builder.WriteString("- To inspect current lightweight diagnostics, call read_problems. To run tests/builds, call list_tasks first, then run_task with a discovered taskId after approval.\n")
	builder.WriteString("- To inspect generated reports, task runs, query exports, or saved assistant outputs, call list_artifacts and then read_artifact.\n")
	builder.WriteString("- To understand provenance across generated outputs, source files, chats, and tool runs, call read_artifact_lineage.\n")
	builder.WriteString("- To inspect a web page or HTTP text source, call web_fetch only after approval. Use allowedDomains when the user scoped the source, and cite the fetched URL/status in your answer.\n")
	builder.WriteString("- To analyze table-like files, call profile_dataset first for schema/profile metadata, then query_dataset or query_dataset_sql for bounded rows. Use read-only SELECT queries only.\n")
	builder.WriteString("- To analyze SQLite databases, call inspect_sqlite first for schema and relationships, then query_sqlite with a read-only SELECT/WITH query and a bounded resultLimit.\n")
	builder.WriteString("- To analyze Dockerfiles, Compose files, env files, scripts, configs, or logs, call inspect_operations. It is read-only and redacts environment-like secret values.\n")
	builder.WriteString("- To analyze a folder of documents, call read_document_set with the folder path or explicit document paths before summarizing, comparing, or extracting findings.\n")
	builder.WriteString("- To create or replace text/code/config/Markdown/JSON/source files, call write_file with the desired workspace-relative relPath and full text content.\n")
	builder.WriteString("- For multi-file code/text edits or precise edits to existing files, prefer apply_patch with a standard unified diff. Include enough context so each hunk matches exactly.\n")
	builder.WriteString("- To create or replace binary files such as images, archives, fonts, executables, databases, or Office/PDF binaries, call write_binary_file with standard base64Content. Do not use write_file for binary bytes.\n")
	builder.WriteString("- To add to a file, call append_file with only the text to append.\n")
	builder.WriteString("- To copy, rename, move, or delete a file, call copy_file, move_file, or delete_file; never simulate those operations by only describing them.\n")
	builder.WriteString("- To undo an approved file mutation, call list_rollbacks first, then rollback_file_mutation with the rollback id after approval.\n")
	builder.WriteString("- If the user asks to create, save, write, document, record, edit, fix, implement, or list findings in a file or project, you must use a write tool before saying the change exists.\n")
	builder.WriteString("- If a write returns approval required, tell the user the file was not changed and summarize the proposed diff or binary summary.\n")
	builder.WriteString("- Never say a file was created, saved, written, copied, moved, renamed, deleted, or modified unless a mutation tool observation confirms success.\n\n")
	builder.WriteString("Current permissions:\n")
	builder.WriteString(fmt.Sprintf("- File writes approved for this run: %t\n", s.writeAccessApproved))
	builder.WriteString(fmt.Sprintf("- Shell commands approved for this run: %t\n\n", s.shellAccessAllowed))
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

func (s *runState) finalizationPrompt() string {
	var builder strings.Builder
	builder.WriteString("The Nexus Agent must wrap up now. Do not request more tools and do not output another Action.\n")
	builder.WriteString("Produce a concise user-facing answer now using only the completed observations and working memory. ")
	builder.WriteString("If the work is incomplete, say exactly where it stopped and what the next safe step is.\n")
	builder.WriteString("Use this exact prefix:\nFinal Answer: ...\n\n")
	builder.WriteString("User request:\n")
	builder.WriteString(s.userPrompt)
	builder.WriteString("\n\nCurrent plan:\n")
	for _, step := range s.plan {
		builder.WriteString(fmt.Sprintf("- [%s] %s\n", step.Status, step.Step))
	}
	builder.WriteString("\nCompleted tool calls:\n")
	if len(s.toolCalls) == 0 {
		builder.WriteString("- none\n")
	} else {
		for index, call := range s.toolCalls {
			status := strings.TrimSpace(call.Observation)
			if call.Error != "" {
				status = "ERROR: " + call.Error
			}
			status, _ = truncateUTF8(status, 900, false)
			builder.WriteString(fmt.Sprintf("- %d. %s: %s\n", index+1, call.Name, status))
		}
	}
	builder.WriteString("\nRecent working memory:\n")
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

func (s *runState) guardUnverifiedSideEffectClaims(message string) string {
	message = strings.TrimSpace(message)
	if message == "" || !claimsWorkspaceSideEffect(message) || s.hasSuccessfulSideEffectTool() {
		return message
	}
	return message + "\n\nVerification warning: this agent run has no successful file-write, append, shell, or artifact tool record, so any claim that a file was created, saved, written, or modified is unverified."
}

func (s *runState) needsRequestedMutationTool() bool {
	return s.writeAccessApproved && requestsPersistentWorkspaceChange(s.userPrompt) && !s.hasSuccessfulFileWriteTool()
}

func (s *runState) rejectPrematureWriteAnswer() {
	s.appendHistory("Observation", "The user requested a persistent workspace change and write access is approved, but no successful mutation tool has run yet. Do not produce a final answer until you call the appropriate write, patch, copy, move, delete, or artifact tool with the complete requested inputs.")
}

func (s *runState) hasSuccessfulSideEffectTool() bool {
	for _, call := range s.toolCalls {
		if call.Error != "" {
			continue
		}
		switch call.Name {
		case "write_file", "write_binary_file", "apply_patch", "append_file", "copy_file", "move_file", "delete_file", "run_task", "execute_shell_command", "generate_artifact", "workspace.write", "workspace.writeBinary", "workspace.patch", "workspace.copy", "workspace.move", "workspace.delete", "workspace.task.run", "artifact.create", "artifact.archive":
			return true
		}
	}
	return false
}

func (s *runState) hasSuccessfulFileWriteTool() bool {
	for _, call := range s.toolCalls {
		if call.Error != "" {
			continue
		}
		switch call.Name {
		case "write_file", "write_binary_file", "apply_patch", "append_file", "copy_file", "move_file", "delete_file", "workspace.write", "workspace.writeBinary", "workspace.patch", "workspace.copy", "workspace.move", "workspace.delete":
			return true
		}
	}
	return false
}

func stoppedRunMessage(state *runState) string {
	toolCount := len(state.toolCalls)
	toolLabel := "tool calls"
	if toolCount == 1 {
		toolLabel = "tool call"
	}
	return fmt.Sprintf(
		"I had to pause before I could finish cleanly. I completed %d %s. The plan and tool calls below show the last confirmed state; run the agent again to continue from there.",
		toolCount,
		toolLabel,
	)
}

func fitsModelContext(prompt string, settings storage.LLMSettings) bool {
	maxPromptTokens := settings.MaxContextTokens - settings.ResponseReserveTokens - agentPromptSafetyTokens
	if maxPromptTokens <= 0 {
		maxPromptTokens = defaultAgentContextTokens - agentPromptSafetyTokens
	}
	return estimateTokens(prompt) <= maxPromptTokens
}

func estimateTokens(value string) int {
	if value == "" {
		return 0
	}
	return (utf8.RuneCountInString(value) + 3) / 4
}

func emitRunEvent(request RunRequest, observe RunObserver, event RunEvent) {
	if observe == nil {
		return
	}
	event.RequestID = request.RequestID
	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	observe(event)
}

func limitEventText(value string, maxBytes int) string {
	trimmed, _ := truncateUTF8(strings.TrimSpace(value), maxBytes, false)
	return trimmed
}

func SystemPrompt() string {
	return strings.Join([]string{
		"You are Nexus Agent, the local-first assistant inside Nexus Augentic Studio for code, data, documents, operations, and artifacts.",
		"Work as a permissioned co-pilot: inspect before acting, keep all actions inside the active workspace, and never claim access to files you did not inspect.",
		"Use a ReAct loop with concise Thought, a single Action, the resulting Observation, and then a Final Answer when the task is done.",
		"Use update_plan for multi-step work and keep exactly one step in_progress.",
		"High-impact tools such as writes, task runs, shell commands, deletes, moves, and Docker actions require explicit approval. If approval is missing, explain the proposed action instead of pretending it ran.",
		"To inspect a folder or project, request read_context with workspace-relative paths; to inspect current uncommitted work, request read_git_diff and read_changed_files; to inspect committed history or attribution, request read_git_history or read_git_blame; to inspect diagnostics, request read_problems; to analyze datasets, request profile_dataset then query_dataset or query_dataset_sql; to analyze SQLite databases, request inspect_sqlite then query_sqlite; to analyze Dockerfiles, Compose files, env files, scripts, configs, or logs, request inspect_operations; to analyze document folders, request read_document_set; to inspect generated outputs, request list_artifacts and read_artifact; to inspect provenance, request read_artifact_lineage; to inspect approved HTTP(S) text sources, request web_fetch; to run tests/builds, request list_tasks first and then run_task with a discovered taskId after approval; to create or update text/code/config files, request write_file with workspace-relative relPath and full content; for multi-file or precise code edits, prefer apply_patch with a standard unified diff; to add text to a file, request append_file with only the appended content.",
		"To create or update binary files such as images, archives, fonts, executables, databases, or Office/PDF binaries, request write_binary_file with workspace-relative relPath and standard base64Content.",
		"To copy, rename, move, or delete files, request copy_file, move_file, or delete_file with workspace-relative paths.",
		"To undo an approved workspace file mutation, request list_rollbacks first and then rollback_file_mutation with the rollback id after approval.",
		"If the user asks to create, save, write, edit, fix, implement, document, record, or list findings in a file or project, use an appropriate write tool before saying it is done.",
		"Never claim a file or artifact was created, saved, written, or modified unless a tool observation confirmed the mutation.",
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
				args[key] = stringifyActionArgument(value)
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

func stringifyActionArgument(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	case map[string]any, []any:
		encoded, err := json.Marshal(typed)
		if err == nil {
			return string(encoded)
		}
	}
	return fmt.Sprint(value)
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

func claimsWorkspaceSideEffect(message string) bool {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return false
	}
	negativePhrases := []string{
		"did not create",
		"didn't create",
		"could not create",
		"cannot create",
		"can't create",
		"not created",
		"no file was created",
		"did not write",
		"didn't write",
		"could not write",
		"cannot write",
		"can't write",
		"not written",
		"did not save",
		"could not save",
		"cannot save",
		"can't save",
		"not saved",
		"not documented",
		"did not document",
		"didn't document",
		"could not document",
		"cannot document",
		"can't document",
		"not recorded",
		"did not record",
		"didn't record",
		"could not record",
		"cannot record",
		"can't record",
	}
	for _, phrase := range negativePhrases {
		if strings.Contains(normalized, phrase) {
			return false
		}
	}
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\b(i|i have|i've|we|we have|we've)\s+(created|wrote|written|saved|updated|modified|appended|generated|documented|recorded|added)\b`),
		regexp.MustCompile(`\b(file|document|report|artifact|markdown|md|tracker|docs?|findings?)\b[^.\n]{0,100}\b(created|written|saved|updated|modified|appended|generated|documented|recorded|added)\b`),
		regexp.MustCompile(`\b(created|wrote|written|saved|updated|modified|appended|generated|documented|recorded|added)\b[^.\n]{0,100}\b(file|document|report|artifact|markdown|\.md|tracker|docs?|findings?)\b`),
		regexp.MustCompile(`\b(documented|recorded|saved|added)\b[^.\n]{0,100}\b(in|to)\s+(the\s+)?(new\s+)?(file|document|report|artifact|markdown|\.md)\b`),
		regexp.MustCompile(`\b[a-z0-9_.-]+\.md\b[^.\n]{0,100}\b(created|written|saved|updated|modified|appended|generated|documented|recorded|added)\b`),
		regexp.MustCompile(`\b(created|wrote|written|saved|updated|modified|appended|generated|documented|recorded|added)\b[^.\n]{0,100}\b[a-z0-9_.-]+\.md\b`),
	}
	for _, pattern := range patterns {
		if pattern.MatchString(normalized) {
			return true
		}
	}
	return false
}

func requestsPersistentWorkspaceChange(message string) bool {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if normalized == "" {
		return false
	}
	negativePhrases := []string{
		"do not write",
		"don't write",
		"without writing",
		"no file",
		"not to a file",
	}
	for _, phrase := range negativePhrases {
		if strings.Contains(normalized, phrase) {
			return false
		}
	}
	mutationVerbs := `(create|write|save|append|update|modify|edit|change|fix|implement|refactor|rename|move|remove|delete|patch|apply|generate|document|record|list|cleanup|clean\s+up)`
	workspaceTargets := `(file|document|report|artifact|markdown|\.md|tracker|findings?|code|source|bug|bugs|issue|issues|app|project|workspace|ui|backend|frontend|component|module|docs?|tests?)`
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\b` + mutationVerbs + `\b[^.\n?]{0,120}\b` + workspaceTargets + `\b`),
		regexp.MustCompile(`\b` + workspaceTargets + `\b[^.\n?]{0,120}\b` + mutationVerbs + `\b`),
		regexp.MustCompile(`\b(to|into|in)\s+(a\s+|the\s+)?(new\s+)?(file|document|report|artifact|markdown)\b`),
		regexp.MustCompile(`\b[a-z0-9_.-]+\.md\b`),
	}
	for _, pattern := range patterns {
		if pattern.MatchString(normalized) {
			return true
		}
	}
	return false
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
