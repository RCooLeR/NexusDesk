import {Button} from '../../components/ui';
import type {FilePreview, GitFileChange, GitFileDiff, GitStatus, WorkspaceFreshnessStatus, WorkspaceSearchResult, WorkspaceSnapshot} from '../../types';

type CodeStudioPanelProps = {
    activeFile: string;
    dirtyTabPaths: string[];
    filePreview: FilePreview | null;
    gitStatus: GitStatus | null;
    selectedGitChangePath: string;
    selectedGitFileDiff: GitFileDiff | null;
    isLoadingGitFileDiff: boolean;
    isSearchingWorkspace: boolean;
    openTabs: FilePreview[];
    onClearWorkspaceSearch: () => void;
    onOpenCommandPalette: () => void;
    onRefreshGitStatus: () => void;
    onSearchWorkspace: () => void;
    onSelectGitChange: (path: string) => void;
    onSelectSearchResult: (result: WorkspaceSearchResult) => void;
    onWorkspaceSearchQueryChange: (value: string) => void;
    workspace: WorkspaceSnapshot | null;
    workspaceFreshness: WorkspaceFreshnessStatus | null;
    workspaceSearchQuery: string;
    workspaceSearchResults: WorkspaceSearchResult[];
};

export function CodeStudioPanel({
    activeFile,
    dirtyTabPaths,
    filePreview,
    gitStatus,
    selectedGitChangePath,
    selectedGitFileDiff,
    isLoadingGitFileDiff,
    isSearchingWorkspace,
    openTabs,
    onClearWorkspaceSearch,
    onOpenCommandPalette,
    onRefreshGitStatus,
    onSearchWorkspace,
    onSelectGitChange,
    onSelectSearchResult,
    onWorkspaceSearchQueryChange,
    workspace,
    workspaceFreshness,
    workspaceSearchQuery,
    workspaceSearchResults,
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

    return (
        <div className="code-studio-panel">
            <section className="code-studio-column code-studio-overview">
                <div className="bottom-section-heading">
                    <strong>Workbench</strong>
                    <small>{workspace ? workspace.name : 'No workspace open'}</small>
                </div>
                <div className="code-studio-toolbar" aria-label="Workbench toolbar">
                    <Button onClick={onRefreshGitStatus} disabled={!workspace} variant="subtle">Refresh git</Button>
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
                    <small>{workspaceSearchResults.length > 0 ? `${workspaceSearchResults.length} matches` : 'Path and previewable text search'}</small>
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
                            placeholder="Search files, artifacts, chat"
                            value={workspaceSearchQuery}
                        />
                        <Button disabled={!workspace || isSearchingWorkspace || !workspaceSearchQuery.trim()} onClick={onSearchWorkspace} variant="subtle">
                            {isSearchingWorkspace ? 'Searching...' : 'Search'}
                        </Button>
                        <Button disabled={workspaceSearchResults.length === 0} onClick={onClearWorkspaceSearch} variant="subtle">Clear</Button>
                    </div>
                    <div className="code-studio-list" aria-label="Workbench search results">
                        {workspaceSearchResults.length > 0 ? workspaceSearchResults.slice(0, 12).map((result, index) => (
                            <button className="code-studio-search-result" key={`${result.relPath}-${result.matchType}-${index}`} onClick={() => onSelectSearchResult(result)} type="button">
                                <span>{result.matchType || result.fileType || result.kind}</span>
                                <strong title={result.relPath}>{result.relPath || result.name}</strong>
                                <small>{result.line > 0 ? `line ${result.line} / ` : ''}{result.snippet || result.name}</small>
                            </button>
                        )) : (
                            <div className="code-studio-empty">{workspace ? 'Run a search to populate this panel.' : 'Open a workspace to search.'}</div>
                        )}
                    </div>
                    <div className="code-studio-queue-grid compact" aria-label="Code studio queues">
                        <QueueCard label="Problems" value="pending" />
                        <QueueCard label="Tasks" value="pending" />
                        <QueueCard label="AI Review" value={filePreview?.fileType === 'code' ? 'ready' : 'context'} />
                        <QueueCard label="Data files nearby" value={String(dataFiles.length)} />
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
