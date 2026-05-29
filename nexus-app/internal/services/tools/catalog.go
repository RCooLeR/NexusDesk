package tools

import (
	"fmt"
	"sort"
	"strings"

	"nexusdesk/internal/services/agent"
)

const (
	ToolStatusImplemented = "implemented"
	ToolStatusPlanned     = "planned"
)

type ToolCatalogEntry struct {
	Descriptor agent.ToolDescriptor
	Category   string
	Status     string
	Stage      string
	Notes      string
}

func DefaultToolCatalog() []ToolCatalogEntry {
	implemented := implementedToolCatalog(NewDefaultDispatcher(Dependencies{}).ToolDescriptors())
	planned := plannedToolCatalog()
	entries := append(implemented, planned...)
	sort.Slice(entries, func(left int, right int) bool {
		if entries[left].Category == entries[right].Category {
			if entries[left].Status == entries[right].Status {
				return entries[left].Descriptor.Name < entries[right].Descriptor.Name
			}
			return entries[left].Status < entries[right].Status
		}
		return entries[left].Category < entries[right].Category
	})
	return entries
}

func implementedToolCatalog(descriptors []agent.ToolDescriptor) []ToolCatalogEntry {
	entries := make([]ToolCatalogEntry, 0, len(descriptors))
	for _, descriptor := range descriptors {
		entries = append(entries, ToolCatalogEntry{
			Descriptor: descriptor,
			Category:   implementedToolCategory(descriptor.Name),
			Status:     ToolStatusImplemented,
			Stage:      "native",
			Notes:      implementedToolNotes(descriptor.Name),
		})
	}
	return entries
}

func plannedToolCatalog() []ToolCatalogEntry {
	return []ToolCatalogEntry{
		plannedTool("workspace", "semantic_search_workspace", "Search indexed workspace chunks and symbols with ranked semantic/project-memory results.", "low", "query, filters(optional), maxResults(optional)", "post-indexing", "Requires durable indexing, embedding/provider policy, cache invalidation, and citation diagnostics."),
		plannedTool("workspace", "read_symbol_index", "Read symbols, definitions, and exported APIs for files or the project.", "low", "relPath(optional), query(optional)", "editor parity", "Backed by language-aware indexers or LSP adapters."),
		plannedTool("workspace", "goto_definition", "Resolve a symbol or location to definition candidates.", "low", "relPath, line, column, symbol(optional)", "editor parity", "Must stay framework-free and expose bounded results to Fyne."),
		plannedTool("workspace", "find_references", "Find bounded references for a symbol or selected range.", "low", "relPath, line, column, symbol(optional)", "editor parity", "Pairs with references UI and source citations."),
		plannedTool("workspace", "read_dependency_graph", "Return module/package/file dependency relationships.", "low", "relPath(optional), depth(optional)", "architecture", "Needed for IDE-grade assistant explanations and refactors."),
		plannedTool("workspace", "update_project_memory", "Store reviewed project facts, conventions, and decisions.", "medium", "key, content, sourceRelPaths(optional)", "memory", "Requires provenance, user review, edit/delete, and stale-source warnings."),

		plannedTool("editor", "format_file", "Format a file with an approved formatter or language service.", "high", "relPath, formatter(optional)", "editor parity", "Must preview diff, require approval for writes, and use rollback snapshots."),
		plannedTool("editor", "lint_file", "Run bounded lint/diagnostic checks for a file or project slice.", "medium", "relPath(optional), linter(optional)", "editor parity", "Can use terminal/job plumbing but needs diagnostic normalization."),
		plannedTool("editor", "apply_code_action", "Apply a language-server or deterministic code action.", "high", "relPath, actionId, range(optional)", "editor parity", "Requires preview, approval, rollback, and language-action provenance."),
		plannedTool("editor", "rename_symbol", "Rename a symbol across all safe references.", "high", "relPath, line, column, newName", "refactor", "Requires conflict-aware multi-file patching and preview."),
		plannedTool("editor", "generate_tests", "Generate or update test files from selected code and project conventions.", "high", "relPath, targetRelPath(optional), instructions(optional)", "coding", "Should use safe write/patch tools and run focused tests after approval."),
		plannedTool("editor", "review_code_selection", "Review selected code or diff with structured findings.", "low", "relPath(optional), diffScope(optional)", "coding", "Can start as an assistant workflow, then become a recorded tool output."),

		plannedTool("git", "stage_file", "Stage one workspace Git path.", "high", "relPath", "git parity", "Requires Git approval policy, path validation, and audit."),
		plannedTool("git", "unstage_file", "Unstage one workspace Git path.", "high", "relPath", "git parity", "Requires Git approval policy, path validation, and audit."),
		plannedTool("git", "stage_hunk", "Stage one validated diff hunk.", "high", "relPath, hunkId", "git parity", "Use existing hunk parser and exact index patching."),
		plannedTool("git", "commit_changes", "Create a commit from approved staged changes.", "high", "message, body(optional)", "git parity", "Requires clean staged summary, author visibility, and no amend/force behavior."),
		plannedTool("git", "create_branch", "Create or switch to a Git branch.", "high", "branchName, startPoint(optional)", "git parity", "Needs branch naming policy and user confirmation."),
		plannedTool("git", "revert_changes", "Revert selected working-tree changes through an explicit preview.", "high", "relPath(optional), scope", "git parity", "Destructive; needs preview, confirmation, and audit."),
		plannedTool("git", "resolve_conflict", "Apply an approved conflict-resolution patch.", "high", "relPath, strategyOrPatch", "git parity", "Requires merge-conflict parser and rollback."),
		plannedTool("github", "draft_pull_request", "Draft a PR title/body from branch diff and issue context.", "medium", "base(optional), head(optional)", "collaboration", "Creation/push stays separate and approval-gated."),
		plannedTool("github", "create_pull_request", "Create a PR through a configured connector.", "high", "base, head, title, body", "collaboration", "Requires connector auth, permission check, and audit."),
		plannedTool("github", "read_pr_comments", "Read review comments and unresolved threads.", "medium", "repo, prNumber", "collaboration", "Requires GitHub connector and source mapping."),

		plannedTool("terminal", "start_terminal_session", "Start an approved interactive terminal/job session.", "high", "command, argsJson(optional), cwd(optional)", "terminal", "Requires durable process supervision, cancel, logs, and UI session controls."),
		plannedTool("terminal", "send_terminal_input", "Send input to an approved running terminal session.", "high", "sessionId, input", "terminal", "Requires session ownership, redaction, and prompt-state safeguards."),
		plannedTool("terminal", "read_terminal_output", "Read capped output from a running terminal session.", "medium", "sessionId, since(optional)", "terminal", "Should stream into Jobs/Terminal UI and agent observations."),
		plannedTool("terminal", "stop_terminal_session", "Cancel or terminate a running terminal session.", "high", "sessionId", "terminal", "Needs graceful/force stop policy and audit."),
		plannedTool("jobs", "list_jobs", "List durable jobs and recent statuses.", "low", "status(optional), limit(optional)", "jobs", "Expose existing job repository safely to the agent."),
		plannedTool("jobs", "read_job_logs", "Read capped logs for one durable job.", "low", "jobId, tailBytes(optional)", "jobs", "Needs redaction and source/job lineage."),
		plannedTool("jobs", "cancel_job", "Cancel a running durable job.", "high", "jobId", "jobs", "Requires ownership, approval, and cancellation audit."),

		plannedTool("browser", "browser_navigate", "Open or navigate an isolated browser page.", "medium", "url, allowLocal(optional)", "browser", "Requires sandboxed browser runtime, URL policy, and network disclosure."),
		plannedTool("browser", "browser_click", "Click an element in an active browser page.", "medium", "sessionId, selectorOrText", "browser", "Needs page session state and observation snapshots."),
		plannedTool("browser", "browser_type", "Type into an active browser page.", "high", "sessionId, selectorOrText, text", "browser", "Can submit data; needs approval and secret redaction."),
		plannedTool("browser", "browser_screenshot", "Capture a screenshot from an active browser page.", "medium", "sessionId, fullPage(optional)", "browser", "Feeds vision/screenshot model route and artifact storage."),
		plannedTool("browser", "browser_extract_page", "Extract structured text, links, forms, and accessibility tree.", "medium", "sessionId", "browser", "Extends web_fetch with rendered-page context."),
		plannedTool("browser", "browser_network_log", "Read capped request/response metadata for a browser session.", "medium", "sessionId", "browser", "Useful for app debugging; must redact headers/tokens."),

		plannedTool("data", "create_dataset_chart", "Create a deterministic chart artifact from dataset query results.", "high", "relPath, chartJson", "data artifacts", "Requires artifact lineage and regeneration metadata."),
		plannedTool("database", "list_db_profiles", "List configured external database profiles visible to the workspace.", "low", "scope(optional)", "connectors", "Must never expose secrets."),
		plannedTool("database", "inspect_db_profile", "Inspect an external database profile schema read-only.", "medium", "profileId", "connectors", "Route through durable jobs with cancellation and lineage."),
		plannedTool("database", "query_db_profile", "Run a guarded read-only query against an external profile.", "medium", "profileId, sql, limit(optional)", "connectors", "Requires approval, timeout, query history, and export lineage."),
		plannedTool("database", "import_database_dump", "Import a dump into a local sandbox for analysis.", "high", "relPath, engine", "connectors", "Slow, high-risk; must use durable jobs, sandboxing, and disk caps."),

		plannedTool("documents", "compare_documents", "Compare two document/artifact sources.", "low", "leftRelPath, rightRelPath", "documents", "Use existing artifact comparison where possible."),
		plannedTool("documents", "generate_docx", "Generate a DOCX from an approved document artifact.", "high", "sourceRelPath, template(optional)", "documents", "Requires packaged export validation and regeneration metadata."),
		plannedTool("presentations", "generate_pptx", "Generate a PPTX deck from an approved presentation artifact.", "high", "sourceRelPath, template(optional)", "presentations", "Requires packaged export validation and regeneration metadata."),
		plannedTool("images", "describe_image", "Describe or extract details from a local image/screenshot.", "medium", "relPath, prompt(optional)", "vision", "Routes to vision-capable model defaults and stores source metadata."),
		plannedTool("images", "compare_screenshots", "Compare screenshots for visual regression or UI polish.", "medium", "baselineRelPath, candidateRelPath", "vision", "Needs deterministic diff plus optional vision summary."),
		plannedTool("images", "generate_image_asset", "Generate an approved bitmap asset.", "high", "prompt, targetRelPath", "media", "Requires model/provider policy, provenance, and write approval."),

		plannedTool("connectors", "list_connectors", "List configured connector providers and readiness.", "low", "kind(optional)", "connectors", "Covers GitHub, Jira, analytics, CRM, cloud storage, and local services."),
		plannedTool("connectors", "run_connector_action", "Run a permissioned connector action.", "high", "connectorId, action, argsJson", "connectors", "Requires provider-specific scopes, dry-run, approval, and audit."),
		plannedTool("mcp", "list_mcp_servers", "List configured MCP servers and health.", "low", "", "extensibility", "Wait until native core tools are stable."),
		plannedTool("mcp", "list_mcp_tools", "List tools exposed by an MCP server.", "low", "serverId", "extensibility", "Needs allow-list and schema preview."),
		plannedTool("mcp", "call_mcp_tool", "Call an approved MCP tool.", "high", "serverId, toolName, argsJson", "extensibility", "Must enforce per-tool risk, approvals, timeouts, and audit."),
		plannedTool("plugins", "discover_plugins", "Discover installed and available NexusDesk plugins.", "low", "query(optional)", "extensibility", "Requires signed/verified plugin policy."),
		plannedTool("plugins", "install_plugin", "Install an approved plugin.", "high", "pluginId, version(optional)", "extensibility", "Requires trust, signature, sandbox, and rollback."),

		plannedTool("automation", "schedule_agent_run", "Create a scheduled or recurring agent run.", "high", "name, prompt, schedule", "automation", "Requires user-visible schedule, workspace binding, approvals, and notifications."),
		plannedTool("automation", "list_automations", "List scheduled agent runs and monitors.", "low", "status(optional)", "automation", "Needed before automation mutation tools."),
		plannedTool("automation", "pause_automation", "Pause or resume an automation.", "high", "automationId, status", "automation", "Requires clear ownership and audit."),
		plannedTool("automation", "run_automation_now", "Trigger an automation manually.", "high", "automationId", "automation", "Must reuse normal approval/job rules."),

		plannedTool("operations", "docker_compose_config", "Run approved `docker compose config` for validation.", "high", "composeRelPath", "operations", "Can route through discovered task/job contract."),
		plannedTool("operations", "docker_compose_logs", "Read capped Docker Compose logs.", "high", "service(optional), tail(optional)", "operations", "Requires Docker policy, redaction, and user approval."),
		plannedTool("operations", "docker_compose_lifecycle", "Start/stop/restart approved Compose services.", "high", "action, service(optional)", "operations", "Destructive/system-impacting; requires explicit UX and rollback/mitigation notes."),

		plannedTool("security", "request_approval", "Create an explicit approval request record for a proposed high-risk action.", "medium", "action, risk, summary", "security", "Needed for multi-step agent plans and deferred approvals."),
		plannedTool("security", "list_approvals", "List relevant approval records and current trust policy.", "low", "status(optional)", "security", "Expose approval posture without secrets."),
		plannedTool("security", "redact_text", "Redact secrets from text before storage or connector transmission.", "low", "content", "security", "Centralize secret redaction diagnostics."),
	}
}

func plannedTool(category string, name string, description string, risk string, inputs string, stage string, notes string) ToolCatalogEntry {
	return ToolCatalogEntry{
		Descriptor: agent.ToolDescriptor{Name: name, Description: description, Risk: risk, Inputs: inputs},
		Category:   category,
		Status:     ToolStatusPlanned,
		Stage:      stage,
		Notes:      notes,
	}
}

func implementedToolCategory(name string) string {
	switch name {
	case "read_context", "read_file", "search_workspace", "read_problems":
		return "workspace"
	case "profile_dataset", "query_dataset", "query_dataset_sql":
		return "data"
	case "inspect_sqlite", "query_sqlite":
		return "database"
	case "extract_document":
		return "documents"
	case "inspect_operations_files", "generate_runbook":
		return "operations"
	case "read_git_status", "read_git_diff", "read_git_history", "read_git_blame":
		return "git"
	case "list_tasks", "run_task", "run_terminal_command":
		return "terminal"
	case "write_file", "append_file", "copy_file", "move_file", "delete_file", "apply_patch", "list_rollbacks", "rollback_file_mutation":
		return "files"
	case "read_artifact_lineage", "regenerate_artifact":
		return "artifacts"
	case "web_fetch":
		return "browser"
	case "list_external_agent_tools", "plan_external_agent_run":
		return "external-agents"
	case "list_tool_catalog":
		return "registry"
	default:
		return "other"
	}
}

func implementedToolNotes(name string) string {
	switch name {
	case "run_terminal_command":
		return "One-shot approved argv execution only; richer terminal sessions remain planned."
	case "web_fetch":
		return "HTTP(S) text fetch only; rendered browser automation remains planned."
	case "list_external_agent_tools", "plan_external_agent_run":
		return "Detection/planning only; NexusDesk's own tools are the primary agent surface."
	case "regenerate_artifact":
		return "Supports artifact kinds with saved source/dependency metadata."
	default:
		return "Registered in the native deterministic dispatcher."
	}
}

func formatToolCatalog(entries []ToolCatalogEntry) string {
	if len(entries) == 0 {
		return "No tools match the requested catalog filter."
	}
	var builder strings.Builder
	implemented := 0
	planned := 0
	for _, entry := range entries {
		switch entry.Status {
		case ToolStatusImplemented:
			implemented++
		case ToolStatusPlanned:
			planned++
		}
	}
	builder.WriteString(fmt.Sprintf("NexusDesk native tool catalog: %d implemented, %d planned.\n", implemented, planned))
	builder.WriteString("Implemented tools are executable now. Planned tools are roadmap contracts and must not be called until implemented.\n")
	currentCategory := ""
	for _, entry := range entries {
		if entry.Category != currentCategory {
			currentCategory = entry.Category
			builder.WriteString("\n## ")
			builder.WriteString(currentCategory)
			builder.WriteString("\n")
		}
		builder.WriteString("- [")
		builder.WriteString(entry.Status)
		builder.WriteString("] ")
		builder.WriteString(entry.Descriptor.Name)
		builder.WriteString(" (risk=")
		builder.WriteString(entry.Descriptor.Risk)
		if entry.Stage != "" {
			builder.WriteString(", stage=")
			builder.WriteString(entry.Stage)
		}
		builder.WriteString("): ")
		builder.WriteString(entry.Descriptor.Description)
		if entry.Descriptor.Inputs != "" {
			builder.WriteString(" Inputs=")
			builder.WriteString(entry.Descriptor.Inputs)
		}
		if entry.Notes != "" {
			builder.WriteString(" Notes=")
			builder.WriteString(entry.Notes)
		}
		builder.WriteString("\n")
	}
	return builder.String()
}
