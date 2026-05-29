# NexusDesk Native Agent Tool Registry

This document is the source-of-truth map for NexusDesk's own first-party LLM toolbelt. The goal is Codex/Claude/OpenCode-class agent capability inside NexusDesk without making external CLIs the main product surface.

Rules:

- Implemented tools are registered in `nexus-app/internal/services/tools` and may be called by the native agent.
- Planned tools are roadmap contracts only. They must not appear as executable tool descriptors until backend validation, approval policy, audit logging, output caps, tests, and UI affordances exist.
- Services remain framework-free. Fyne imports stay in UI/app/theme packages.
- High-risk actions require explicit approval, audit, rollback or mitigation where practical, redaction, and bounded execution.
- Workspace open must never run Git, Docker, terminal commands, connector pulls, model calls, browser automation, OCR, dump imports, or deep indexing automatically.

## Implemented Tool Surface

The implemented registry is exposed to the agent through `ToolDescriptors()` and can be inspected by the low-risk `list_tool_catalog` tool.

| Category | Implemented tools | Notes |
| --- | --- | --- |
| Registry | `list_tool_catalog` | Lists implemented and planned first-party tools by category/status. |
| Workspace | `read_context`, `read_file`, `search_workspace`, `read_problems` | Bounded workspace reads, search, context packs, and lightweight diagnostics. |
| Files | `write_file`, `append_file`, `copy_file`, `move_file`, `delete_file`, `apply_patch`, `list_rollbacks`, `rollback_file_mutation` | Approval-gated safe mutation tools with path validation and rollback snapshots. |
| Git | `read_git_status`, `read_git_diff`, `read_git_history`, `read_git_blame`, `stage_file`, `unstage_file`, `stage_hunk`, `unstage_hunk`, `commit_changes`, `create_branch`, `resolve_conflict` | Read-only repository context plus approval-gated index-only staging/unstaging, commits from already-staged changes, branch creation with optional checkout, and rollback-backed conflict-marker resolution. Destructive Git actions remain planned. |
| Terminal/tasks | `list_tasks`, `run_task`, `run_terminal_command` | Discovered safe tasks and one-shot approved terminal commands by executable name plus explicit JSON args. Shell interpreters and command paths are blocked. |
| Jobs | `list_jobs`, `read_job_logs`, `cancel_job` | Redacted durable job status/log access plus approval-gated cancellation for running jobs. |
| Browser/web | `web_fetch` | Approval-gated HTTP(S) text fetch only. Rendered browser automation remains planned. |
| Artifacts | `read_artifact_lineage`, `regenerate_artifact` | Artifact lineage context and approval-gated regeneration for supported artifact kinds. |
| Data | `profile_dataset`, `query_dataset`, `query_dataset_sql`, `create_dataset_chart` | Local dataset profiling, bounded row queries, medium-risk approval-gated SELECT-only dataset SQL, and high-risk approval-gated chart artifact generation. |
| Database | `inspect_sqlite`, `query_sqlite` | Medium-risk approval-gated workspace SQLite schema inspection and bounded read-only SELECT/WITH queries. |
| Documents | `extract_document` | Bounded text and metadata extraction for supported workspace documents. |
| Operations | `inspect_operations_files`, `generate_runbook` | Read-only operations file scan/inspection plus approval-gated runbook artifact generation from redacted evidence; no Docker/shell execution. |
| External agent readiness | `list_external_agent_tools`, `plan_external_agent_run` | Detection/planning only for optional Codex, Claude Code, and OpenCode integrations. NexusDesk's own tools remain primary. |

## Planned Complete Toolbelt

The planned registry should be implemented in priority order, with tests and documentation per logical milestone.

### Workspace And IDE Intelligence

- `semantic_search_workspace`: ranked semantic/project-memory search.
- `read_symbol_index`: symbol, definition, and export index.
- `goto_definition`: definition candidates for a symbol/location.
- `find_references`: bounded reference lookup.
- `read_dependency_graph`: module/package/file dependency relationships.
- `update_project_memory`: reviewed project facts and conventions with provenance.

### Editor And Refactoring

- `format_file`: approved formatter with diff preview and rollback.
- `lint_file`: bounded linter/diagnostic execution.
- `apply_code_action`: approved language-service or deterministic code action.
- `rename_symbol`: conflict-aware multi-file symbol rename.
- `generate_tests`: create or update tests from selected code and conventions.
- `review_code_selection`: structured findings for selected code/diff.

### Git And Collaboration

- `revert_changes`: destructive revert/discard through explicit preview.
- `draft_pull_request`, `create_pull_request`, `read_pr_comments`: GitHub/PR workflows through configured connectors.

### Terminal And Durable Jobs

- `start_terminal_session`, `send_terminal_input`, `read_terminal_output`, `stop_terminal_session`: interactive terminal sessions with durable process supervision.

### Browser Automation

- `browser_navigate`, `browser_click`, `browser_type`, `browser_screenshot`, `browser_extract_page`, `browser_network_log`: isolated rendered-browser sessions with URL policy, screenshots, page extraction, and network diagnostics.

### Data, Databases, And Connectors

- `list_db_profiles`, `inspect_db_profile`, `query_db_profile`, `import_database_dump`: external database profile and sandbox import tools.
- `list_connectors`, `run_connector_action`: permissioned non-database connectors such as GitHub, Jira, analytics, CRM, and cloud storage.

### Documents, Presentations, Images

- `compare_documents`, `generate_docx`, `generate_pptx`: document and presentation intelligence/generation.
- `describe_image`, `compare_screenshots`, `generate_image_asset`: vision/screenshot and approved media-generation workflows.

### MCP, Plugins, And Extensibility

- `list_mcp_servers`, `list_mcp_tools`, `call_mcp_tool`: MCP discovery and permissioned tool calls.
- `discover_plugins`, `install_plugin`: signed/verified plugin lifecycle.

### Automation

- `schedule_agent_run`, `list_automations`, `pause_automation`, `run_automation_now`: scheduled and recurring agent work with visible ownership, approvals, notifications, and audit.

### Operations And Security

- `docker_compose_config`, `docker_compose_logs`, `docker_compose_lifecycle`: redacted operations tooling and approved Docker workflows.
- `request_approval`, `list_approvals`, `redact_text`: approval and redaction primitives for multi-step tool plans.

## Implementation Gate For Each Tool

Before a planned tool becomes executable, it needs:

1. Framework-free service implementation.
2. Rooted path validation or connector scope validation.
3. Risk classification and approval behavior.
4. Timeouts, caps, cancellation, and redaction.
5. Audit records and lineage for outputs.
6. Rollback or mitigation for mutations where practical.
7. Focused tests for success, denial, invalid inputs, traversal/scope violations, and timeout/cancel behavior.
8. UI affordance when the action needs user review, preview, or cancellation.
9. Tracker and production-plan updates.

## Priority Order

1. Add mutating Git tools with preview/approval/audit.
2. Add browser automation with screenshots and rendered-page extraction.
3. Add interactive terminal sessions on top of durable jobs.
4. Add GitHub/PR connector tools.
5. Add semantic/symbol indexing and LSP-backed editor actions.
6. Add MCP/plugin discovery after native core tools are stable.
7. Add automation scheduling after jobs, approvals, and notification UX are mature.
