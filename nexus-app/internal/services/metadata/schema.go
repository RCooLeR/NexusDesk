package metadata

const schemaVersion = 9

const schemaSQL = `PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS workspaces (
    id TEXT PRIMARY KEY,
    root TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    opened_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    workspace_root TEXT NOT NULL,
    kind TEXT NOT NULL,
    label TEXT NOT NULL,
    status TEXT NOT NULL,
    message TEXT,
    error TEXT,
    log_tail_json TEXT,
    started_at TEXT NOT NULL,
    completed_at TEXT
);

CREATE TABLE IF NOT EXISTS task_runs (
    id TEXT PRIMARY KEY,
    workspace_root TEXT NOT NULL,
    job_id TEXT,
    task_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    label TEXT NOT NULL,
    command TEXT NOT NULL,
    cwd TEXT NOT NULL,
    source TEXT,
    status TEXT NOT NULL,
    exit_code INTEGER,
    stdout TEXT,
    stderr TEXT,
    message TEXT,
    artifact_path TEXT,
    started_at TEXT NOT NULL,
    completed_at TEXT,
    duration_ms INTEGER
);

CREATE TABLE IF NOT EXISTS agent_runs (
    id TEXT PRIMARY KEY,
    workspace_root TEXT NOT NULL,
    job_id TEXT,
    prompt TEXT NOT NULL,
    status TEXT NOT NULL,
    message TEXT,
    model TEXT,
    model_route_id TEXT,
    model_route TEXT,
    route_warning TEXT,
    iterations INTEGER,
    stop_reason TEXT,
    plan_json TEXT,
    source_paths_json TEXT,
    started_at TEXT NOT NULL,
    completed_at TEXT,
    duration_ms INTEGER
);

CREATE TABLE IF NOT EXISTS tool_runs (
    id TEXT PRIMARY KEY,
    workspace_root TEXT NOT NULL,
    agent_run_id TEXT NOT NULL,
    job_id TEXT,
    sequence INTEGER,
    tool_name TEXT NOT NULL,
    risk TEXT,
    mutated INTEGER,
    args_json TEXT,
    observation TEXT,
    error TEXT,
    started_at TEXT,
    completed_at TEXT
);

CREATE TABLE IF NOT EXISTS chat_messages (
    id TEXT PRIMARY KEY,
    workspace_root TEXT NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    model TEXT,
    context_rel_path TEXT,
    source_paths_json TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS artifacts (
    id TEXT PRIMARY KEY,
    workspace_root TEXT NOT NULL,
    kind TEXT NOT NULL,
    title TEXT,
    rel_path TEXT NOT NULL,
    metadata_path TEXT,
    size INTEGER,
    job_id TEXT,
    task_id TEXT,
    source TEXT,
    source_paths_json TEXT,
    archived INTEGER,
    created_at TEXT,
    generated_at TEXT,
    updated_at TEXT NOT NULL,
    UNIQUE(workspace_root, rel_path)
);

CREATE TABLE IF NOT EXISTS approval_records (
    id TEXT PRIMARY KEY,
    workspace_root TEXT NOT NULL,
    action TEXT NOT NULL,
    target TEXT,
    risk TEXT NOT NULL,
    decision TEXT NOT NULL,
    message TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sql_runs (
    id TEXT PRIMARY KEY,
    workspace_root TEXT NOT NULL,
    rel_path TEXT NOT NULL,
    sql_text TEXT NOT NULL,
    engine TEXT NOT NULL,
    status TEXT NOT NULL,
    row_count INTEGER,
    matched_rows INTEGER,
    shown_rows INTEGER,
    message TEXT,
    error TEXT,
    artifact_path TEXT,
    started_at TEXT NOT NULL,
    completed_at TEXT,
    duration_ms INTEGER
);

CREATE TABLE IF NOT EXISTS dataset_dependencies (
    id TEXT PRIMARY KEY,
    workspace_root TEXT NOT NULL,
    source_path TEXT NOT NULL,
    dependent_kind TEXT NOT NULL,
    dependent_ref TEXT NOT NULL,
    relation TEXT NOT NULL,
    metadata_json TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(workspace_root, source_path, dependent_kind, dependent_ref, relation)
);
`

func SchemaSQL() string {
	return schemaSQL
}
