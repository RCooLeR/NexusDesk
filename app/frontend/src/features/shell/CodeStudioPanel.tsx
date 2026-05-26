import {Button} from '../../components/ui';
import type {FilePreview, GitFileChange, GitFileDiff, GitStatus, WorkspaceFreshnessStatus, WorkspaceProblemSummary, WorkspaceSearchResult, WorkspaceSnapshot, WorkspaceTask, WorkspaceTaskRunResult, WorkspaceTaskSummary} from '../../types';

type CodeStudioPanelProps = {
    activeFile: string;
    dirtyTabPaths: string[];
    filePreview: FilePreview | null;
    gitStatus: GitStatus | null;
    selectedGitChangePath: string;
    selectedGitFileDiff: GitFileDiff | null;
    isLoadingGitFileDiff: boolean;
    isLoadingWorkspaceProblems: boolean;
    isLoadingWorkspaceTasks: boolean;
    isReviewingCode: boolean;
    isRunningWorkspaceTask: boolean;
    isSearchingWorkspace: boolean;
    openTabs: FilePreview[];
    onClearWorkspaceSearch: () => void;
    onOpenCommandPalette: () => void;
    onReplacePreviewChange: (value: string) => void;
    onRefreshGitStatus: () => void;
    onRefreshWorkspaceProblems: () => void;
    onRefreshWorkspaceTasks: () => void;
    onReviewCurrentFile: () => void;
    onReviewGitDiff: () => void;
    onRunWorkspaceTask: (task: WorkspaceTask) => void;
    onSearchWorkspace: () => void;
    onSelectGitChange: (path: string) => void;
    onSelectSearchResult: (result: WorkspaceSearchResult) => void;
    onWorkspaceSearchQueryChange: (value: string) => void;
    onWorkspaceSearchRegexChange: (value: boolean) => void;
    workspace: WorkspaceSnapshot | null;
    workspaceFreshness: WorkspaceFreshnessStatus | null;
    workspaceProblems: WorkspaceProblemSummary | null;
    workspaceSearchQuery: string;
    workspaceSearchRegex: boolean;
    workspaceSearchResults: WorkspaceSearchResult[];
    workspaceReplacePreview: string;
    workspaceTaskRun: WorkspaceTaskRunResult | null;
    workspaceTasks: WorkspaceTaskSummary | null;
};

export function CodeStudioPanel({
    activeFile,
    dirtyTabPaths,
    filePreview,
    gitStatus,
    selectedGitChangePath,
    selectedGitFileDiff,
    isLoadingGitFileDiff,
    isLoadingWorkspaceProblems,
    isLoadingWorkspaceTasks,
    isReviewingCode,
    isRunningWorkspaceTask,
    isSearchingWorkspace,
    openTabs,
    onClearWorkspaceSearch,
    onOpenCommandPalette,
    onReplacePreviewChange,
    onRefreshGitStatus,
    onRefreshWorkspaceProblems,
    onRefreshWorkspaceTasks,
    onReviewCurrentFile,
    onReviewGitDiff,
    onRunWorkspaceTask,
    onSearchWorkspace,
    onSelectGitChange,
    onSelectSearchResult,
    onWorkspaceSearchQueryChange,
    onWorkspaceSearchRegexChange,
    workspace,
    workspaceFreshness,
    workspaceProblems,
    workspaceSearchQuery,
    workspaceSearchRegex,
    workspaceSearchResults,
    workspaceReplacePreview,
    workspaceTaskRun,
    workspaceTasks,
}: CodeStudioPanelProps) {
    const nodes = workspace?.nodes ?? [];
    const codeFiles = nodes.filter((node) => node.kind === 'file' && node.fileType === 'code');
    const dataFiles = nodes.filter((node) => node.kind === 'file' && node.fileType === 'data');
    const changedFiles = workspaceFreshness?.changed ?? [];
    const activeLanguage = resolveActiveLanguage(filePreview, activeFile);
    const gitChangedFiles = gitStatus?.changedFiles ?? [];
    const stagedFiles = gitStatus?.stagedFiles ?? [];
    const unstagedFiles = gitStatus?.unstagedFiles ?? gitChangedFiles;
    const selectedGitChange = gitChangedFiles.find((change) => change.path === selectedGitChangePath) ?? null;
    const hasSelectedDiff = selectedGitFileDiff?.path === selectedGitChangePath && Boolean(selectedGitFileDiff.stagedDiff || selectedGitFileDiff.unstagedDiff);
    const canReviewCurrentFile = Boolean(workspace && filePreview?.kind === 'file' && filePreview.content);
    const canReviewGitDiff = Boolean(hasSelectedDiff || gitStatus?.stagedDiff || gitStatus?.unstagedDiff || gitStatus?.diff);

    return (
        <div className="code-studio-panel">
            <section className="code-studio-column code-studio-overview">
                <div className="bottom-section-heading">
                    <strong>Workbench</strong>
                    <small>{workspace ? workspace.name : 'No workspace open'}</small>
                </div>
                <div className="code-studio-toolbar" aria-label="Workbench toolbar">
                    <Button onClick={onRefreshGitStatus} disabled={!workspace} variant="subtle">Refresh git</Button>
                    <Button onClick={onReviewCurrentFile} disabled={!canReviewCurrentFile || isReviewingCode} variant="subtle">
                        {isReviewingCode ? 'Reviewing...' : 'Review file'}
                    </Button>
                    <Button onClick={onReviewGitDiff} disabled={!canReviewGitDiff || isReviewingCode} variant="subtle">
                        {isReviewingCode ? 'Reviewing...' : 'Review diff'}
                    </Button>
                    <Button onClick={onOpenCommandPalette} variant="subtle">Commands</Button>
                    <Button disabled variant="subtle">Terminal</Button>
                </div>
                <div className="code-studio-metrics" aria-label="Code studio status">
                    <Metric label="Indexed files" value={String(nodes.filter((node) => node.kind === 'file').length)} />
                    <Metric label="Code files" value={String(codeFiles.length)} />
                    <Metric label="Open tabs" value={String(openTabs.length)} />
                    <Metric label="Dirty tabs" value={String(dirtyTabPaths.length)} />
                </div>
                <div className="code-studio-active-file">
                    <span>Active file</span>
                    <strong title={activeFile}>{activeFile || 'Workspace root'}</strong>
                    <small>{activeLanguage}</small>
                    {selectedGitChange && (
                        <>
                        <span>Selected change</span>
                        <strong title={selectedGitChange.path}>{selectedGitChange.path}</strong>
                        <small>{selectedGitChange.summary} / {gitCode(selectedGitChange.index, selectedGitChange.worktree)}{isLoadingGitFileDiff ? ' / loading diff' : hasSelectedDiff ? ' / diff loaded' : ''}</small>
                        </>
                    )}
                </div>
            </section>

            <section className="code-studio-column">
                <div className="bottom-section-heading">
                    <strong>Project Session</strong>
                    <small>{workspace ? `${nodes.length} indexed tree entries` : 'Open a workspace to populate this surface'}</small>
                </div>
                <div className="code-studio-list" aria-label="Open editor tabs">
                    {openTabs.length > 0 ? openTabs.slice(0, 8).map((tab) => (
                        <div className="code-studio-row" key={tab.relPath}>
                            <span>{tab.fileType || tab.kind}</span>
                            <strong title={tab.relPath}>{tab.relPath}</strong>
                            <small>{dirtyTabPaths.includes(tab.relPath) ? 'dirty' : tab.encoding || tab.kind}</small>
                        </div>
                    )) : (
                        <div className="code-studio-empty">No editor tabs open.</div>
                    )}
                </div>
            </section>

            <section className="code-studio-column">
                <div className="bottom-section-heading">
                    <strong>Repository</strong>
                    <small>{gitStatus?.available ? `${gitStatus.branch}${gitStatus.head ? ` @ ${gitStatus.head}` : ''}` : gitStatus?.message ?? 'Git status surface'}</small>
                </div>
                {gitStatus?.available && (
                    <div className={gitStatus.dirty ? 'git-summary dirty' : 'git-summary'}>
                        <strong>{gitStatus.dirty ? `${gitChangedFiles.length} changed` : 'Clean'}</strong>
                        <span>{stagedFiles.length} staged / {unstagedFiles.length} unstaged</span>
                        <span>{gitStatus.aheadBehind || gitStatus.message}</span>
                    </div>
                )}
                <div className="code-studio-list" aria-label="Changed workspace files">
                    {gitStatus?.available && gitChangedFiles.length > 0 ? (
                        <>
                            <GitChangeGroup changes={stagedFiles} label="Staged" onSelect={onSelectGitChange} selectedPath={selectedGitChangePath} />
                            <GitChangeGroup changes={unstagedFiles} label="Unstaged" onSelect={onSelectGitChange} selectedPath={selectedGitChangePath} />
                        </>
                    ) : changedFiles.length > 0 ? changedFiles.slice(0, 8).map((change) => (
                        <div className="code-studio-row" key={change.relPath}>
                            <span>{change.kind || 'changed'}</span>
                            <strong title={change.relPath}>{change.relPath}</strong>
                            <small>{change.message}</small>
                        </div>
                    )) : (
                        <div className="code-studio-empty">{gitStatus?.available ? 'No git changes detected.' : 'No changed files reported by freshness polling.'}</div>
                    )}
                </div>
            </section>

            <section className="code-studio-column">
                <div className="bottom-section-heading">
                    <strong>Search</strong>
                    <small>{workspaceSearchResults.length > 0 ? `${workspaceSearchResults.length} matches` : 'Path, text, symbols, regex, and replace preview'}</small>
                </div>
                <div className="code-studio-search-panel">
                    <div className="code-studio-search-controls">
                        <input
                            aria-label="Workbench search query"
                            disabled={!workspace}
                            onChange={(event) => onWorkspaceSearchQueryChange(event.target.value)}
                            onKeyDown={(event) => {
                                if (event.key === 'Enter' && workspaceSearchQuery.trim()) {
                                    onSearchWorkspace();
                                }
                            }}
                            placeholder={workspaceSearchRegex ? 'Regex search files, symbols, artifacts, chat' : 'Search files, symbols, artifacts, chat'}
                            value={workspaceSearchQuery}
                        />
                        <Button disabled={!workspace || isSearchingWorkspace || !workspaceSearchQuery.trim()} onClick={onSearchWorkspace} variant="subtle">
                            {isSearchingWorkspace ? 'Searching...' : 'Search'}
                        </Button>
                        <Button disabled={workspaceSearchResults.length === 0} onClick={onClearWorkspaceSearch} variant="subtle">Clear</Button>
                    </div>
                    <div className="code-studio-search-options">
                        <label>
                            <input checked={workspaceSearchRegex} disabled={!workspace} onChange={(event) => onWorkspaceSearchRegexChange(event.target.checked)} type="checkbox" />
                            Regex
                        </label>
                        <input
                            aria-label="Replace preview text"
                            disabled={!workspace || workspaceSearchResults.length === 0}
                            onChange={(event) => onReplacePreviewChange(event.target.value)}
                            placeholder="Replace preview"
                            value={workspaceReplacePreview}
                        />
                    </div>
                    <div className="code-studio-list" aria-label="Workbench search results">
                        {workspaceSearchResults.length > 0 ? workspaceSearchResults.slice(0, 12).map((result, index) => (
                            <button className="code-studio-search-result" key={`${result.relPath}-${result.matchType}-${index}`} onClick={() => onSelectSearchResult(result)} type="button">
                                <span>{result.matchType || result.fileType || result.kind}</span>
                                <strong title={result.relPath}>{result.relPath || result.name}</strong>
                                <small>{result.line > 0 ? `line ${result.line} / ` : ''}{result.snippet || result.name}</small>
                                {workspaceReplacePreview && <em>{replacePreviewText(result.snippet || result.relPath || result.name, workspaceSearchQuery, workspaceReplacePreview, workspaceSearchRegex)}</em>}
                            </button>
                        )) : (
                            <div className="code-studio-empty">{workspace ? 'Run a search to populate this panel.' : 'Open a workspace to search.'}</div>
                        )}
                    </div>
                    <div className="code-studio-problems-panel" aria-label="Workspace problems">
                        <div className="code-studio-task-header">
                            <div>
                                <strong>Problems</strong>
                                <small>{workspaceProblems?.message ?? 'Lightweight TODO, conflict, and JSON checks'}</small>
                            </div>
                            <Button disabled={!workspace || isLoadingWorkspaceProblems} onClick={onRefreshWorkspaceProblems} variant="subtle">
                                {isLoadingWorkspaceProblems ? 'Scanning...' : 'Refresh problems'}
                            </Button>
                        </div>
                        <div className="code-studio-list">
                            {workspaceProblems && workspaceProblems.problems.length > 0 ? workspaceProblems.problems.slice(0, 8).map((problem, index) => (
                                <button className={`code-studio-problem-row ${problem.severity}`} key={`${problem.relPath}-${problem.line}-${index}`} onClick={() => onSelectSearchResult(problemToSearchResult(problem))} type="button">
                                    <span>{problem.severity}</span>
                                    <strong title={problem.relPath}>{problem.relPath}</strong>
                                    <small>{problem.line > 0 ? `line ${problem.line} / ` : ''}{problem.message}</small>
                                </button>
                            )) : (
                                <div className="code-studio-empty">{workspace ? 'Refresh problems to scan lightweight diagnostics.' : 'Open a workspace to scan problems.'}</div>
                            )}
                        </div>
                    </div>
                    <div className="code-studio-queue-grid compact" aria-label="Code studio queues">
                        <QueueCard label="AI Review" value={filePreview?.fileType === 'code' ? 'ready' : 'context'} />
                        <QueueCard label="Data files nearby" value={String(dataFiles.length)} />
                    </div>
                    <div className="code-studio-task-panel" aria-label="Detected workspace tasks">
                        <div className="code-studio-task-header">
                            <div>
                                <strong>Tasks</strong>
                                <small>{workspaceTasks?.message ?? 'Package scripts, Go tests, and Compose files'}</small>
                            </div>
                            <Button disabled={!workspace || isLoadingWorkspaceTasks} onClick={onRefreshWorkspaceTasks} variant="subtle">
                                {isLoadingWorkspaceTasks ? 'Scanning...' : 'Refresh tasks'}
                            </Button>
                        </div>
                        <div className="code-studio-list">
                            {workspaceTasks && workspaceTasks.tasks.length > 0 ? workspaceTasks.tasks.slice(0, 12).map((task) => (
                                <button className="code-studio-task-row" disabled={isRunningWorkspaceTask} key={task.id} onClick={() => onRunWorkspaceTask(task)} type="button">
                                    <span>{task.kind}</span>
                                    <strong title={task.command}>{task.label}</strong>
                                    <small title={`${task.cwd} / ${task.source}`}>{isRunningWorkspaceTask ? 'running...' : `${task.cwd} / ${task.source}`}</small>
                                </button>
                            )) : (
                                <div className="code-studio-empty">{workspace ? 'Refresh tasks to scan package scripts, Go tests, and Compose files.' : 'Open a workspace to detect tasks.'}</div>
                            )}
                        </div>
                        {workspaceTaskRun && (
                            <div className={`code-studio-task-output ${workspaceTaskRun.status}`}>
                                <strong>{workspaceTaskRun.status} / exit {workspaceTaskRun.exitCode}</strong>
                                <small>{workspaceTaskRun.message}</small>
                                <pre>{taskRunPreview(workspaceTaskRun)}</pre>
                            </div>
                        )}
                    </div>
                </div>
            </section>
        </div>
    );
}

function GitChangeGroup({
    label,
    changes,
    selectedPath,
    onSelect,
}: {
    label: string;
    changes: GitFileChange[];
    selectedPath: string;
    onSelect: (path: string) => void;
}) {
    return (
        <div className="git-change-group">
            <small>{label} ({changes.length})</small>
            {changes.length > 0 ? changes.slice(0, 8).map((change) => (
                <button
                    aria-pressed={selectedPath === change.path}
                    className={selectedPath === change.path ? 'code-studio-row selected' : 'code-studio-row'}
                    key={`${label}-${change.index}-${change.worktree}-${change.path}`}
                    onClick={() => onSelect(change.path)}
                    type="button"
                >
                    <span>{change.summary}</span>
                    <strong title={change.path}>{change.path}</strong>
                    <small>{gitCode(change.index, change.worktree)}{change.oldPath ? ` from ${change.oldPath}` : ''}</small>
                </button>
            )) : (
                <div className="code-studio-empty">No {label.toLowerCase()} files.</div>
            )}
        </div>
    );
}

function Metric({label, value}: {label: string; value: string}) {
    return (
        <div className="code-studio-metric">
            <strong>{value}</strong>
            <span>{label}</span>
        </div>
    );
}

function QueueCard({label, value}: {label: string; value: string}) {
    return (
        <div className="code-studio-queue-card">
            <span>{label}</span>
            <strong>{value}</strong>
        </div>
    );
}

function problemToSearchResult(problem: WorkspaceProblemSummary['problems'][number]): WorkspaceSearchResult {
    return {
        relPath: problem.relPath,
        name: problem.name,
        kind: 'file',
        fileType: 'code',
        matchType: problem.source,
        line: problem.line,
        snippet: problem.message,
    };
}

function taskRunPreview(result: WorkspaceTaskRunResult) {
    const text = [result.stdout, result.stderr].filter(Boolean).join('\n--- stderr ---\n').trim();
    if (!text) {
        return 'No output captured.';
    }
    return text.length > 900 ? `${text.slice(0, 900)}\n...` : text;
}

function replacePreviewText(value: string, query: string, replacement: string, regex: boolean) {
    if (!query.trim()) {
        return value;
    }
    try {
        if (regex) {
            return value.replace(new RegExp(query, 'gi'), replacement);
        }
        return value.replace(new RegExp(escapeRegExp(query), 'gi'), replacement);
    } catch {
        return 'Invalid replace preview pattern';
    }
}

function escapeRegExp(value: string) {
    return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function gitCode(index: string, worktree: string) {
    const code = `${index || ' '}${worktree || ' '}`.trim();
    return code || 'tracked';
}

function resolveActiveLanguage(preview: FilePreview | null, activeFile: string) {
    if (preview?.encoding) {
        return `${preview.fileType || preview.kind} / ${preview.encoding}`;
    }
    if (preview?.fileType || preview?.kind) {
        return preview.fileType || preview.kind;
    }
    return activeFile ? 'not loaded' : 'workspace';
}
