import {useEffect, useMemo, useRef, useState} from 'react';
import {faChevronDown, faChevronUp} from '@fortawesome/free-solid-svg-icons';
import {FontAwesomeIcon} from '@fortawesome/react-fontawesome';
import {Button, IconButton} from '../../components/ui';
import type {GitFileAction, GitFileActionPreview, GitFileChange, GitFileDiff, GitHunkActionPreview, GitHunkActionRequest, GitStatus} from '../../types';

type DiffMode = 'unified' | 'split' | 'changes';

type HunkTarget = {
    key: string;
    label: string;
    content: string;
};

type DiffRow =
    | {type: 'meta'; content: string; hunkKey?: string}
    | {type: 'hunk'; content: string; hunkKey: string}
    | {type: 'context'; oldLine: number; newLine: number; oldText: string; newText: string; hunkKey?: string}
    | {type: 'delete'; oldLine: number; oldText: string; hunkKey?: string}
    | {type: 'add'; newLine: number; newText: string; hunkKey?: string};

type ChangedRow = {
    oldLine?: number;
    oldText: string;
    newLine?: number;
    newText: string;
    hunkKey?: string;
};

type GitChangeTreeNode = {
    name: string;
    path: string;
    children: Map<string, GitChangeTreeNode>;
    change?: GitFileChange;
};

type GitDiffPanelProps = {
    gitFileActionPreview: GitFileActionPreview | null;
    gitHunkActionPreview: GitHunkActionPreview | null;
    gitStatus: GitStatus | null;
    selectedGitChangePath: string;
    selectedGitFileDiff: GitFileDiff | null;
    isGeneratingGitInsight: boolean;
    isApplyingGitFileAction: boolean;
    isApplyingGitHunkAction: boolean;
    isLoadingGitFileDiff: boolean;
    isPreviewingGitFileAction: boolean;
    isPreviewingGitHunkAction: boolean;
    onDraftCommitMessage: () => void;
    onPreviewGitFileAction: (action: GitFileAction) => void;
    onApplyGitFileAction: (action: GitFileAction) => void;
    onPreviewGitHunkAction: (request: GitHunkActionRequest) => void;
    onApplyGitHunkAction: (request: GitHunkActionRequest) => void;
    onRefreshGitStatus: () => void;
    onSelectGitChange: (path: string) => void;
    onSummarizeDiff: () => void;
};

export function GitDiffPanel({
    gitFileActionPreview,
    gitHunkActionPreview,
    gitStatus,
    selectedGitChangePath,
    selectedGitFileDiff,
    isGeneratingGitInsight,
    isApplyingGitFileAction,
    isApplyingGitHunkAction,
    isLoadingGitFileDiff,
    isPreviewingGitFileAction,
    isPreviewingGitHunkAction,
    onDraftCommitMessage,
    onPreviewGitFileAction,
    onApplyGitFileAction,
    onPreviewGitHunkAction,
    onApplyGitHunkAction,
    onRefreshGitStatus,
    onSelectGitChange,
    onSummarizeDiff,
}: GitDiffPanelProps) {
    const [diffMode, setDiffMode] = useState<DiffMode>('unified');
    const [activeHunkIndex, setActiveHunkIndex] = useState(0);
    const [selectedHunkKeys, setSelectedHunkKeys] = useState<string[]>([]);
    const diffScrollRef = useRef<HTMLDivElement | null>(null);
    const changedFiles = gitStatus?.changedFiles ?? [];
    const stagedFiles = gitStatus?.stagedFiles ?? [];
    const unstagedFiles = gitStatus?.unstagedFiles ?? changedFiles;
    const selectedGitChange = changedFiles.find((change) => change.path === selectedGitChangePath) ?? null;
    const selectedIsStaged = stagedFiles.some((change) => change.path === selectedGitChangePath);
    const selectedIsUnstaged = unstagedFiles.some((change) => change.path === selectedGitChangePath);
    const selectedDiff = selectedGitFileDiff?.path === selectedGitChangePath ? selectedGitFileDiff : null;
    const stagedDiff = selectedDiff?.stagedDiff ?? gitStatus?.stagedDiff ?? '';
    const unstagedDiff = selectedDiff?.unstagedDiff ?? gitStatus?.unstagedDiff ?? gitStatus?.diff ?? '';
    const stagedDiffTruncated = selectedDiff?.stagedDiffTruncated ?? gitStatus?.stagedDiffTruncated;
    const unstagedDiffTruncated = selectedDiff?.unstagedDiffTruncated ?? gitStatus?.unstagedDiffTruncated;
    const statusMessage = gitStatus?.available
        ? `${gitStatus.branch}${gitStatus.head ? ` @ ${gitStatus.head}` : ''}`
        : gitStatus?.message ?? 'Open a workspace, then press Refresh git when you need repository status.';
    const hasDiff = Boolean(stagedDiff || unstagedDiff);
    const hunkTargets = useMemo(() => [
        ...collectHunks('staged', stagedDiff),
        ...collectHunks('unstaged', unstagedDiff),
    ], [stagedDiff, unstagedDiff]);
    const activeHunkKey = hunkTargets[activeHunkIndex]?.key ?? '';
    const activeHunkIsSelected = selectedHunkKeys.includes(activeHunkKey);
    const activeHunkIndexRequest = hunkRequestFromKey(activeHunkKey, selectedGitChangePath, 'index');
    const activeHunkDestructiveRequest = hunkRequestFromKey(activeHunkKey, selectedGitChangePath, 'destructive');
    const hunkActionDisabled = !activeHunkIsSelected || isPreviewingGitHunkAction || isApplyingGitHunkAction;

    useEffect(() => {
        setActiveHunkIndex(0);
        setSelectedHunkKeys([]);
    }, [selectedGitChangePath, stagedDiff, unstagedDiff]);

    useEffect(() => {
        if (!activeHunkKey || !diffScrollRef.current) {
            return;
        }
        const target = diffScrollRef.current.querySelector(`[data-hunk-key="${cssEscape(activeHunkKey)}"]`);
        target?.scrollIntoView({block: 'nearest'});
    }, [activeHunkKey, diffMode]);

    function moveHunk(delta: number) {
        if (hunkTargets.length === 0) {
            return;
        }
        setActiveHunkIndex((current) => (current + delta + hunkTargets.length) % hunkTargets.length);
    }

    function toggleActiveHunkSelection() {
        if (!activeHunkKey) {
            return;
        }
        setSelectedHunkKeys((current) => current.includes(activeHunkKey)
            ? current.filter((key) => key !== activeHunkKey)
            : [...current, activeHunkKey]);
    }

    function previewActiveHunkAction(kind: 'index' | 'destructive') {
        const request = kind === 'index' ? activeHunkIndexRequest : activeHunkDestructiveRequest;
        if (request) {
            onPreviewGitHunkAction(request);
        }
    }

    function applyActiveHunkAction(kind: 'index' | 'destructive') {
        const request = kind === 'index' ? activeHunkIndexRequest : activeHunkDestructiveRequest;
        if (request) {
            onApplyGitHunkAction(request);
        }
    }

    return (
        <div className="git-diff-panel">
            <section className="git-diff-sidebar">
                <div className="bottom-section-heading">
                    <strong>Git</strong>
                    <small>{statusMessage}</small>
                </div>
                <div className="code-studio-toolbar" aria-label="Git diff toolbar">
                    <Button onClick={onRefreshGitStatus} variant="subtle">Refresh git</Button>
                    <Button disabled={!selectedIsUnstaged || isPreviewingGitFileAction} onClick={() => onPreviewGitFileAction('stage')} variant="subtle">Preview stage</Button>
                    <Button disabled={!selectedIsUnstaged || isApplyingGitFileAction} onClick={() => onApplyGitFileAction('stage')} variant="subtle">Stage file</Button>
                    <Button disabled={!selectedIsStaged || isPreviewingGitFileAction} onClick={() => onPreviewGitFileAction('unstage')} variant="subtle">Preview unstage</Button>
                    <Button disabled={!selectedIsStaged || isApplyingGitFileAction} onClick={() => onApplyGitFileAction('unstage')} variant="subtle">Unstage file</Button>
                    <Button disabled={!hasDiff || isGeneratingGitInsight} onClick={onSummarizeDiff} variant="subtle">Summarize diff</Button>
                    <Button disabled={!hasDiff || isGeneratingGitInsight} onClick={onDraftCommitMessage} variant="subtle">Draft commit</Button>
                </div>
                {gitStatus?.available && (
                    <div className={gitStatus.dirty ? 'git-summary dirty' : 'git-summary'}>
                        <strong>{gitStatus.dirty ? `${changedFiles.length} changed` : 'Clean'}</strong>
                        <span>{stagedFiles.length} staged / {unstagedFiles.length} unstaged</span>
                        <span>{gitStatus.aheadBehind || gitStatus.message}</span>
                    </div>
                )}
                {gitFileActionPreview && (
                    <div className="git-action-preview">
                        <strong>{gitActionLabel(gitFileActionPreview.action)}</strong>
                        {gitFileActionPreview.command.length > 0 && <code>{formatGitCommand(gitFileActionPreview.command)}</code>}
                        <small>{gitFileActionPreview.message}</small>
                    </div>
                )}
                {gitHunkActionPreview && (
                    <div className="git-action-preview destructive">
                        <strong>{gitHunkActionLabel(gitHunkActionPreview.action)}</strong>
                        {gitHunkActionPreview.command.length > 0 && <code>{formatGitCommand(gitHunkActionPreview.command)}</code>}
                        <small>{gitHunkActionPreview.message}</small>
                    </div>
                )}
                <div className="code-studio-list" aria-label="Working tree changed files">
                    {gitStatus?.available && changedFiles.length > 0 ? (
                        <>
                            <GitChangeGroup changes={stagedFiles} label="Staged" onSelect={onSelectGitChange} selectedPath={selectedGitChangePath} />
                            <GitChangeGroup changes={unstagedFiles} label="Unstaged" onSelect={onSelectGitChange} selectedPath={selectedGitChangePath} />
                        </>
                    ) : (
                        <div className="code-studio-empty">{gitStatus?.available ? 'No git changes detected.' : statusMessage}</div>
                    )}
                </div>
            </section>
            <section className="git-diff-main">
                <div className="bottom-section-heading">
                    <strong>Working Tree Diff</strong>
                    <small>{diffHeading(isLoadingGitFileDiff, selectedGitChange?.path, stagedDiffTruncated || unstagedDiffTruncated)}</small>
                </div>
                {stagedDiff || unstagedDiff ? (
                    <div className="git-diff-stack" ref={diffScrollRef}>
                        <div className="git-diff-controls" aria-label="Diff view controls">
                            <div className="segmented-control" role="group" aria-label="Diff view mode">
                                <button aria-pressed={diffMode === 'unified'} className={diffMode === 'unified' ? 'active' : ''} onClick={() => setDiffMode('unified')} type="button">Unified</button>
                                <button aria-pressed={diffMode === 'split'} className={diffMode === 'split' ? 'active' : ''} onClick={() => setDiffMode('split')} type="button">Split</button>
                                <button aria-pressed={diffMode === 'changes'} className={diffMode === 'changes' ? 'active' : ''} onClick={() => setDiffMode('changes')} type="button">Diff Only</button>
                            </div>
                            <div className="hunk-nav" aria-label="Hunk navigation">
                                <IconButton className="hunk-nav-button" disabled={hunkTargets.length === 0} label="Previous hunk" onClick={() => moveHunk(-1)}>
                                    <FontAwesomeIcon icon={faChevronUp} />
                                </IconButton>
                                <span>{hunkTargets.length > 0 ? `${activeHunkIndex + 1} / ${hunkTargets.length}` : '0 / 0'}</span>
                                <IconButton className="hunk-nav-button" disabled={hunkTargets.length === 0} label="Next hunk" onClick={() => moveHunk(1)}>
                                    <FontAwesomeIcon icon={faChevronDown} />
                                </IconButton>
                                <Button disabled={!activeHunkKey} onClick={toggleActiveHunkSelection} variant="subtle">{activeHunkIsSelected ? 'Unselect hunk' : 'Select hunk'}</Button>
                                <Button disabled={hunkActionDisabled || !activeHunkIndexRequest} onClick={() => previewActiveHunkAction('index')} variant="subtle">Preview {hunkActionLabel(activeHunkIndexRequest).toLowerCase()}</Button>
                                <Button disabled={hunkActionDisabled || !activeHunkIndexRequest} onClick={() => applyActiveHunkAction('index')} variant="subtle">{hunkActionLabel(activeHunkIndexRequest)}</Button>
                                <Button disabled={hunkActionDisabled || !activeHunkDestructiveRequest} onClick={() => previewActiveHunkAction('destructive')} variant="subtle">Preview {hunkActionLabel(activeHunkDestructiveRequest).toLowerCase()}</Button>
                                <Button disabled={hunkActionDisabled || !activeHunkDestructiveRequest} onClick={() => applyActiveHunkAction('destructive')} variant="subtle">{hunkActionLabel(activeHunkDestructiveRequest)}</Button>
                            </div>
                        </div>
                        {hunkTargets.length > 0 && (
                            <div className="hunk-selection-summary">
                                <strong>{selectedHunkKeys.length} / {hunkTargets.length} hunks selected</strong>
                                <small>{selectedHunkKeys.length > 0 ? selectedHunkLabel(selectedHunkKeys, hunkTargets) : 'Selection is read-only and will feed approved hunk actions later.'}</small>
                                {selectedHunkKeys.length > 0 && <button onClick={() => setSelectedHunkKeys([])} type="button">Clear</button>}
                            </div>
                        )}
                        {selectedDiff?.message && <small className="git-diff-message">{selectedDiff.message}</small>}
                        {stagedDiff && (
                            <DiffBlock activeHunkKey={activeHunkKey} diff={stagedDiff} kind="staged" mode={diffMode} selectedHunkKeys={selectedHunkKeys} title="Staged Diff" />
                        )}
                        {unstagedDiff && (
                            <DiffBlock activeHunkKey={activeHunkKey} diff={unstagedDiff} kind="unstaged" mode={diffMode} selectedHunkKeys={selectedHunkKeys} title="Unstaged Diff" />
                        )}
                    </div>
                ) : (
                    <div className="code-studio-empty">Select a changed file or refresh git status to load a diff.</div>
                )}
            </section>
        </div>
    );
}

function DiffBlock({activeHunkKey, diff, kind, mode, selectedHunkKeys, title}: {activeHunkKey: string; diff: string; kind: string; mode: DiffMode; selectedHunkKeys: string[]; title: string}) {
    const rows = useMemo(() => parseUnifiedDiff(kind, diff), [kind, diff]);
    return (
        <div className="git-diff-block">
            <strong>{title}</strong>
            {mode === 'split' ? (
                <div className="git-diff-split" role="table" aria-label={title}>
                    <div className="git-diff-split-header" role="row">
                        <span>Old</span>
                        <span>New</span>
                    </div>
                    {rows.map((row, index) => renderSplitRow(row, index, activeHunkKey, selectedHunkKeys))}
                </div>
            ) : mode === 'changes' ? (
                <ChangedLinesDiff rows={rows} selectedHunkKeys={selectedHunkKeys} title={title} />
            ) : (
                <pre className="git-diff-view">{rows.map((row, index) => renderUnifiedRow(row, index, activeHunkKey, selectedHunkKeys))}</pre>
            )}
        </div>
    );
}

function ChangedLinesDiff({rows, selectedHunkKeys, title}: {rows: DiffRow[]; selectedHunkKeys: string[]; title: string}) {
    const changedRows = useMemo(() => collectChangedRows(rows), [rows]);
    return (
        <div className="git-diff-changes" role="table" aria-label={`${title} changed lines only`}>
            <div className="git-diff-split-header" role="row">
                <span>Old</span>
                <span>New</span>
            </div>
            {changedRows.length > 0 ? changedRows.map((row, index) => (
                <div className={selectedHunkKeys.includes(row.hunkKey ?? '') ? 'git-diff-split-row selected-hunk' : 'git-diff-split-row'} key={`${index}-${row.oldLine ?? ''}-${row.newLine ?? ''}-${row.oldText}-${row.newText}`} role="row">
                    {row.oldLine ? <DiffCell line={row.oldLine} text={row.oldText} type="delete" /> : <DiffCell text="" type="blank" />}
                    {row.newLine ? <DiffCell line={row.newLine} text={row.newText} type="add" /> : <DiffCell text="" type="blank" />}
                </div>
            )) : (
                <div className="git-diff-empty">No changed lines in this diff.</div>
            )}
        </div>
    );
}

function collectChangedRows(rows: DiffRow[]) {
    const changedRows: ChangedRow[] = [];
    let pendingDeletes: Array<{line: number; text: string; hunkKey?: string}> = [];
    let pendingAdds: Array<{line: number; text: string; hunkKey?: string}> = [];

    function flush() {
        const count = Math.max(pendingDeletes.length, pendingAdds.length);
        for (let index = 0; index < count; index += 1) {
            const deletion = pendingDeletes[index];
            const addition = pendingAdds[index];
            changedRows.push({
                oldLine: deletion?.line,
                oldText: deletion?.text ?? '',
                newLine: addition?.line,
                newText: addition?.text ?? '',
                hunkKey: deletion?.hunkKey ?? addition?.hunkKey,
            });
        }
        pendingDeletes = [];
        pendingAdds = [];
    }

    for (const row of rows) {
        if (row.type === 'delete') {
            pendingDeletes.push({line: row.oldLine, text: row.oldText, hunkKey: row.hunkKey});
            continue;
        }
        if (row.type === 'add') {
            pendingAdds.push({line: row.newLine, text: row.newText, hunkKey: row.hunkKey});
            continue;
        }
        flush();
    }
    flush();
    return changedRows;
}

function renderUnifiedRow(row: DiffRow, index: number, activeHunkKey: string, selectedHunkKeys: string[]) {
    const hunkKey = row.hunkKey;
    const className = [
        'git-diff-line',
        `git-diff-line-${row.type}`,
        hunkKey && hunkKey === activeHunkKey ? 'active-hunk' : '',
        hunkKey && selectedHunkKeys.includes(hunkKey) ? 'selected-hunk' : '',
    ].filter(Boolean).join(' ');
    const text = rowText(row);
    return (
        <span className={className} data-hunk-key={hunkKey} key={`${index}-${text}`}>
            {text}
            {'\n'}
        </span>
    );
}

function renderSplitRow(row: DiffRow, index: number, activeHunkKey: string, selectedHunkKeys: string[]) {
    if (row.type === 'meta' || row.type === 'hunk') {
        const className = [
            'git-diff-split-row',
            `git-diff-split-${row.type}`,
            row.hunkKey === activeHunkKey ? 'active-hunk' : '',
            row.hunkKey && selectedHunkKeys.includes(row.hunkKey) ? 'selected-hunk' : '',
        ].filter(Boolean).join(' ');
        return (
            <div className={className} data-hunk-key={row.hunkKey} key={`${index}-${row.content}`} role="row">
                <span>{row.content}</span>
            </div>
        );
    }
    if (row.type === 'context') {
        return (
            <div className={selectedHunkKeys.includes(row.hunkKey ?? '') ? 'git-diff-split-row selected-hunk' : 'git-diff-split-row'} key={`${index}-${row.oldLine}-${row.newLine}`} role="row">
                <DiffCell line={row.oldLine} text={row.oldText} type="context" />
                <DiffCell line={row.newLine} text={row.newText} type="context" />
            </div>
        );
    }
    if (row.type === 'delete') {
        return (
            <div className={selectedHunkKeys.includes(row.hunkKey ?? '') ? 'git-diff-split-row selected-hunk' : 'git-diff-split-row'} key={`${index}-${row.oldLine}`} role="row">
                <DiffCell line={row.oldLine} text={row.oldText} type="delete" />
                <DiffCell text="" type="blank" />
            </div>
        );
    }
    return (
        <div className={selectedHunkKeys.includes(row.hunkKey ?? '') ? 'git-diff-split-row selected-hunk' : 'git-diff-split-row'} key={`${index}-${row.newLine}`} role="row">
            <DiffCell text="" type="blank" />
            <DiffCell line={row.newLine} text={row.newText} type="add" />
        </div>
    );
}

function DiffCell({line, text, type}: {line?: number; text: string; type: 'add' | 'blank' | 'context' | 'delete'}) {
    return (
        <span className={`git-diff-cell ${type}`} role="cell">
            <small>{line ?? ''}</small>
            <code>{text}</code>
        </span>
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
    const tree = useMemo(() => buildGitChangeTree(changes), [changes]);

    return (
        <div className="git-change-group">
            <small>{label} ({changes.length})</small>
            {changes.length > 0 ? (
                <div className="git-change-tree" role="tree" aria-label={`${label} changed files`}>
                    {tree.map((node) => (
                        <GitChangeTreeNodeView
                            key={node.path}
                            node={node}
                            onSelect={onSelect}
                            selectedPath={selectedPath}
                        />
                    ))}
                </div>
            ) : (
                <div className="code-studio-empty">No {label.toLowerCase()} files.</div>
            )}
        </div>
    );
}

function GitChangeTreeNodeView({
    node,
    onSelect,
    selectedPath,
}: {
    node: GitChangeTreeNode;
    onSelect: (path: string) => void;
    selectedPath: string;
}) {
    if (node.change) {
        const change = node.change;
        return (
            <button
                aria-pressed={selectedPath === change.path}
                className={selectedPath === change.path ? 'git-change-file selected' : 'git-change-file'}
                onClick={() => onSelect(change.path)}
                role="treeitem"
                title={change.path}
                type="button"
            >
                <span>{change.summary}</span>
                <strong>{node.name}</strong>
                <small>{gitCode(change.index, change.worktree)}{change.oldPath ? ` from ${change.oldPath}` : ''}</small>
            </button>
        );
    }

    return (
        <div className="git-change-dir" role="group">
            <div className="git-change-dir-label" role="treeitem" aria-expanded="true">
                <span>{node.name}</span>
            </div>
            <div className="git-change-dir-children">
                {sortedGitTreeChildren(node).map((child) => (
                    <GitChangeTreeNodeView
                        key={child.path}
                        node={child}
                        onSelect={onSelect}
                        selectedPath={selectedPath}
                    />
                ))}
            </div>
        </div>
    );
}

function buildGitChangeTree(changes: GitFileChange[]) {
    const root: GitChangeTreeNode = {name: '', path: '', children: new Map()};
    for (const change of changes) {
        const parts = change.path.split('/').filter(Boolean);
        let current = root;
        parts.forEach((part, index) => {
            const path = parts.slice(0, index + 1).join('/');
            const next: GitChangeTreeNode = current.children.get(part) ?? {name: part, path, children: new Map<string, GitChangeTreeNode>()};
            if (index === parts.length - 1) {
                next.change = change;
            }
            current.children.set(part, next);
            current = next;
        });
    }
    return sortedGitTreeChildren(root);
}

function sortedGitTreeChildren(node: GitChangeTreeNode) {
    return Array.from(node.children.values()).sort((left, right) => {
        if (Boolean(left.change) !== Boolean(right.change)) {
            return left.change ? 1 : -1;
        }
        return left.name.localeCompare(right.name);
    });
}

function diffHeading(isLoading: boolean, selectedPath: string | undefined, isTruncated: boolean | undefined) {
    if (isLoading) {
        return 'Loading selected file diff...';
    }
    if (selectedPath) {
        return `Reviewing ${selectedPath}`;
    }
    if (isTruncated) {
        return 'Read-only diff truncated for responsiveness';
    }
    return 'Read-only staged and unstaged diffs';
}

function selectedHunkLabel(selectedHunkKeys: string[], hunkTargets: HunkTarget[]) {
    const labels = hunkTargets
        .filter((target) => selectedHunkKeys.includes(target.key))
        .map((target) => target.label);
    if (labels.length <= 3) {
        return `Selected: ${labels.join(', ')}`;
    }
    return `Selected: ${labels.slice(0, 3).join(', ')} and ${labels.length - 3} more`;
}

function collectHunks(kind: string, diff: string): HunkTarget[] {
    return diff
        .replace(/\r\n/g, '\n')
        .split('\n')
        .reduce<HunkTarget[]>((hunks, line) => {
            if (line.startsWith('@@')) {
                const index = hunks.length + 1;
                hunks.push({key: `${kind}-${index}`, label: `${kind} ${index}`, content: line});
            }
            return hunks;
        }, []);
}

function parseUnifiedDiff(kind: string, diff: string): DiffRow[] {
    const rows: DiffRow[] = [];
    let oldLine = 0;
    let newLine = 0;
    let hunkIndex = 0;
    let currentHunkKey = '';

    for (const line of diff.replace(/\r\n/g, '\n').split('\n')) {
        if (line === '') {
            rows.push({type: 'context', oldLine: oldLine > 0 ? oldLine++ : 0, newLine: newLine > 0 ? newLine++ : 0, oldText: '', newText: '', hunkKey: currentHunkKey});
            continue;
        }
        if (line.startsWith('@@')) {
            hunkIndex += 1;
            currentHunkKey = `${kind}-${hunkIndex}`;
            const range = line.match(/^@@\s+-(\d+)(?:,\d+)?\s+\+(\d+)(?:,\d+)?\s+@@/);
            oldLine = Number(range?.[1] ?? 0);
            newLine = Number(range?.[2] ?? 0);
            rows.push({type: 'hunk', content: line, hunkKey: currentHunkKey});
            continue;
        }
        if (line.startsWith('diff --git') || line.startsWith('index ') || line.startsWith('---') || line.startsWith('+++')) {
            rows.push({type: 'meta', content: line});
            continue;
        }
        if (line.startsWith('-')) {
            rows.push({type: 'delete', oldLine: oldLine, oldText: line.slice(1), hunkKey: currentHunkKey});
            oldLine += 1;
            continue;
        }
        if (line.startsWith('+')) {
            rows.push({type: 'add', newLine: newLine, newText: line.slice(1), hunkKey: currentHunkKey});
            newLine += 1;
            continue;
        }
        const text = line.startsWith(' ') ? line.slice(1) : line;
        rows.push({type: 'context', oldLine: oldLine, newLine: newLine, oldText: text, newText: text, hunkKey: currentHunkKey});
        oldLine += 1;
        newLine += 1;
    }

    return rows;
}

function rowText(row: DiffRow) {
    switch (row.type) {
    case 'add':
        return `+${row.newText}`;
    case 'delete':
        return `-${row.oldText}`;
    case 'context':
        return ` ${row.oldText}`;
    case 'hunk':
    case 'meta':
    default:
        return row.content;
    }
}

function cssEscape(value: string) {
    if ('CSS' in window && typeof window.CSS.escape === 'function') {
        return window.CSS.escape(value);
    }
    return value.replace(/["\\]/g, '\\$&');
}

function gitCode(index: string, worktree: string) {
    const code = `${index || ' '}${worktree || ' '}`.trim();
    return code || 'tracked';
}

function gitActionLabel(action: string) {
    if (action === 'unstage') {
        return 'Unstage preview';
    }
    return 'Stage preview';
}

function gitHunkActionLabel(action: string) {
    if (action === 'stage') {
        return 'Stage hunk preview';
    }
    if (action === 'unstage') {
        return 'Unstage hunk preview';
    }
    if (action === 'discard') {
        return 'Discard hunk preview';
    }
    return 'Revert hunk preview';
}

function hunkActionLabel(request: GitHunkActionRequest | null) {
    switch (request?.action) {
    case 'stage':
        return 'Stage hunk';
    case 'unstage':
        return 'Unstage hunk';
    case 'discard':
        return 'Discard hunk';
    case 'revert':
        return 'Revert hunk';
    default:
        return 'Hunk action';
    }
}

function hunkRequestFromKey(key: string, path: string, kind: 'index' | 'destructive'): GitHunkActionRequest | null {
    if (!key || !path) {
        return null;
    }
    const match = key.match(/^(staged|unstaged)-(\d+)$/);
    if (!match) {
        return null;
    }
    const diffKind = match[1];
    const action = kind === 'index'
        ? (diffKind === 'unstaged' ? 'stage' : 'unstage')
        : (diffKind === 'unstaged' ? 'discard' : 'revert');
    return {
        path,
        diffKind,
        hunkIndex: Number(match[2]),
        action,
    };
}

function formatGitCommand(command: string[]) {
    return command.map((part) => /[\s"]/u.test(part) ? `"${part.replace(/"/g, '\\"')}"` : part).join(' ');
}
