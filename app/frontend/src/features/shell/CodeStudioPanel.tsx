import {Button} from '../../components/ui';
import type {FilePreview, GitStatus, WorkspaceFreshnessStatus, WorkspaceSnapshot} from '../../types';

type CodeStudioPanelProps = {
    activeFile: string;
    dirtyTabPaths: string[];
    filePreview: FilePreview | null;
    gitStatus: GitStatus | null;
    openTabs: FilePreview[];
    onOpenCommandPalette: () => void;
    onRefreshGitStatus: () => void;
    workspace: WorkspaceSnapshot | null;
    workspaceFreshness: WorkspaceFreshnessStatus | null;
};

export function CodeStudioPanel({
    activeFile,
    dirtyTabPaths,
    filePreview,
    gitStatus,
    openTabs,
    onOpenCommandPalette,
    onRefreshGitStatus,
    workspace,
    workspaceFreshness,
}: CodeStudioPanelProps) {
    const nodes = workspace?.nodes ?? [];
    const codeFiles = nodes.filter((node) => node.kind === 'file' && node.fileType === 'code');
    const dataFiles = nodes.filter((node) => node.kind === 'file' && node.fileType === 'data');
    const changedFiles = workspaceFreshness?.changed ?? [];
    const activeLanguage = resolveActiveLanguage(filePreview, activeFile);
    const stagedFiles = gitStatus?.stagedFiles ?? [];
    const unstagedFiles = gitStatus?.unstagedFiles ?? gitStatus?.changedFiles ?? [];

    return (
        <div className="code-studio-panel">
            <section className="code-studio-column code-studio-overview">
                <div className="bottom-section-heading">
                    <strong>Code Studio</strong>
                    <small>{workspace ? workspace.name : 'No workspace open'}</small>
                </div>
                <div className="code-studio-toolbar" aria-label="Code Studio toolbar">
                    <Button onClick={onRefreshGitStatus} disabled={!workspace} variant="subtle">Refresh git</Button>
                    <Button onClick={onOpenCommandPalette} variant="subtle">Commands</Button>
                    <Button disabled={!gitStatus?.diff} variant="subtle">Diff</Button>
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
                        <strong>{gitStatus.dirty ? `${gitStatus.changedFiles.length} changed` : 'Clean'}</strong>
                        <span>{stagedFiles.length} staged / {unstagedFiles.length} unstaged</span>
                        <span>{gitStatus.aheadBehind || gitStatus.message}</span>
                    </div>
                )}
                <div className="code-studio-list" aria-label="Changed workspace files">
                    {gitStatus?.available && gitStatus.changedFiles.length > 0 ? (
                        <>
                            <GitChangeGroup label="Staged" changes={stagedFiles} />
                            <GitChangeGroup label="Unstaged" changes={unstagedFiles} />
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
                    <strong>Git Diff</strong>
                    <small>{gitStatus?.stagedDiffTruncated || gitStatus?.unstagedDiffTruncated ? 'Read-only diff truncated for responsiveness' : 'Read-only staged and unstaged diffs'}</small>
                </div>
                {gitStatus?.stagedDiff || gitStatus?.unstagedDiff || gitStatus?.diff ? (
                    <div className="git-diff-stack">
                        {gitStatus.stagedDiff && (
                            <>
                                <strong>Staged Diff</strong>
                                <pre className="git-diff-view">{gitStatus.stagedDiff}</pre>
                            </>
                        )}
                        {(gitStatus.unstagedDiff || gitStatus.diff) && (
                            <>
                                <strong>Unstaged Diff</strong>
                                <pre className="git-diff-view">{gitStatus.unstagedDiff || gitStatus.diff}</pre>
                            </>
                        )}
                    </div>
                ) : (
                    <div className="code-studio-queue-grid" aria-label="Code studio queues">
                        <QueueCard label="Search" value={workspace ? 'ready' : 'idle'} />
                        <QueueCard label="Problems" value="pending" />
                        <QueueCard label="Tasks" value="pending" />
                        <QueueCard label="AI Review" value={filePreview?.fileType === 'code' ? 'ready' : 'context'} />
                        <QueueCard label="Data files nearby" value={String(dataFiles.length)} />
                        <QueueCard label="Diff viewer" value={gitStatus?.available ? 'clean' : 'pending'} />
                    </div>
                )}
            </section>
        </div>
    );
}

function GitChangeGroup({label, changes}: {label: string; changes: Array<{path: string; oldPath: string; index: string; worktree: string; summary: string}>}) {
    return (
        <div className="git-change-group">
            <small>{label} ({changes.length})</small>
            {changes.length > 0 ? changes.slice(0, 8).map((change) => (
                <div className="code-studio-row" key={`${label}-${change.index}-${change.worktree}-${change.path}`}>
                    <span>{change.summary}</span>
                    <strong title={change.path}>{change.path}</strong>
                    <small>{gitCode(change.index, change.worktree)}{change.oldPath ? ` from ${change.oldPath}` : ''}</small>
                </div>
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
