import {useEffect, useMemo, useRef, useState} from 'react';
import {Button} from '../../components/ui';
import type {GitFileChange, GitFileDiff, GitStatus} from '../../types';

type DiffMode = 'unified' | 'split';

type HunkTarget = {
    key: string;
    label: string;
};

type DiffRow =
    | {type: 'meta'; content: string; hunkKey?: string}
    | {type: 'hunk'; content: string; hunkKey: string}
    | {type: 'context'; oldLine: number; newLine: number; oldText: string; newText: string}
    | {type: 'delete'; oldLine: number; oldText: string}
    | {type: 'add'; newLine: number; newText: string};

type GitDiffPanelProps = {
    gitStatus: GitStatus | null;
    selectedGitChangePath: string;
    selectedGitFileDiff: GitFileDiff | null;
    isLoadingGitFileDiff: boolean;
    onRefreshGitStatus: () => void;
    onSelectGitChange: (path: string) => void;
};

export function GitDiffPanel({
    gitStatus,
    selectedGitChangePath,
    selectedGitFileDiff,
    isLoadingGitFileDiff,
    onRefreshGitStatus,
    onSelectGitChange,
}: GitDiffPanelProps) {
    const [diffMode, setDiffMode] = useState<DiffMode>('unified');
    const [activeHunkIndex, setActiveHunkIndex] = useState(0);
    const diffScrollRef = useRef<HTMLDivElement | null>(null);
    const stagedFiles = gitStatus?.stagedFiles ?? [];
    const unstagedFiles = gitStatus?.unstagedFiles ?? gitStatus?.changedFiles ?? [];
    const selectedGitChange = gitStatus?.changedFiles.find((change) => change.path === selectedGitChangePath) ?? null;
    const selectedDiff = selectedGitFileDiff?.path === selectedGitChangePath ? selectedGitFileDiff : null;
    const stagedDiff = selectedDiff?.stagedDiff ?? gitStatus?.stagedDiff ?? '';
    const unstagedDiff = selectedDiff?.unstagedDiff ?? gitStatus?.unstagedDiff ?? gitStatus?.diff ?? '';
    const stagedDiffTruncated = selectedDiff?.stagedDiffTruncated ?? gitStatus?.stagedDiffTruncated;
    const unstagedDiffTruncated = selectedDiff?.unstagedDiffTruncated ?? gitStatus?.unstagedDiffTruncated;
    const statusMessage = gitStatus?.available
        ? `${gitStatus.branch}${gitStatus.head ? ` @ ${gitStatus.head}` : ''}`
        : gitStatus?.message ?? 'Open a git-backed workspace to inspect changes.';
    const hunkTargets = useMemo(() => [
        ...collectHunks('staged', stagedDiff),
        ...collectHunks('unstaged', unstagedDiff),
    ], [stagedDiff, unstagedDiff]);
    const activeHunkKey = hunkTargets[activeHunkIndex]?.key ?? '';

    useEffect(() => {
        setActiveHunkIndex(0);
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

    return (
        <div className="git-diff-panel">
            <section className="git-diff-sidebar">
                <div className="bottom-section-heading">
                    <strong>Git</strong>
                    <small>{statusMessage}</small>
                </div>
                <div className="code-studio-toolbar" aria-label="Git diff toolbar">
                    <Button onClick={onRefreshGitStatus} variant="subtle">Refresh git</Button>
                </div>
                {gitStatus?.available && (
                    <div className={gitStatus.dirty ? 'git-summary dirty' : 'git-summary'}>
                        <strong>{gitStatus.dirty ? `${gitStatus.changedFiles.length} changed` : 'Clean'}</strong>
                        <span>{stagedFiles.length} staged / {unstagedFiles.length} unstaged</span>
                        <span>{gitStatus.aheadBehind || gitStatus.message}</span>
                    </div>
                )}
                <div className="code-studio-list" aria-label="Working tree changed files">
                    {gitStatus?.available && gitStatus.changedFiles.length > 0 ? (
                        <>
                            <GitChangeGroup changes={stagedFiles} label="Staged" onSelect={onSelectGitChange} selectedPath={selectedGitChangePath} />
                            <GitChangeGroup changes={unstagedFiles} label="Unstaged" onSelect={onSelectGitChange} selectedPath={selectedGitChangePath} />
                        </>
                    ) : (
                        <div className="code-studio-empty">{gitStatus?.available ? 'No git changes detected.' : 'Git status is unavailable for this workspace.'}</div>
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
                            </div>
                            <div className="hunk-nav" aria-label="Hunk navigation">
                                <Button disabled={hunkTargets.length === 0} onClick={() => moveHunk(-1)} variant="subtle">Prev hunk</Button>
                                <span>{hunkTargets.length > 0 ? `${activeHunkIndex + 1} / ${hunkTargets.length}` : '0 / 0'}</span>
                                <Button disabled={hunkTargets.length === 0} onClick={() => moveHunk(1)} variant="subtle">Next hunk</Button>
                            </div>
                        </div>
                        {selectedDiff?.message && <small className="git-diff-message">{selectedDiff.message}</small>}
                        {stagedDiff && (
                            <DiffBlock activeHunkKey={activeHunkKey} diff={stagedDiff} kind="staged" mode={diffMode} title="Staged Diff" />
                        )}
                        {unstagedDiff && (
                            <DiffBlock activeHunkKey={activeHunkKey} diff={unstagedDiff} kind="unstaged" mode={diffMode} title="Unstaged Diff" />
                        )}
                    </div>
                ) : (
                    <div className="code-studio-empty">Select a changed file or refresh git status to load a diff.</div>
                )}
            </section>
        </div>
    );
}

function DiffBlock({activeHunkKey, diff, kind, mode, title}: {activeHunkKey: string; diff: string; kind: string; mode: DiffMode; title: string}) {
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
                    {rows.map((row, index) => renderSplitRow(row, index, activeHunkKey))}
                </div>
            ) : (
                <pre className="git-diff-view">{rows.map((row, index) => renderUnifiedRow(row, index, activeHunkKey))}</pre>
            )}
        </div>
    );
}

function renderUnifiedRow(row: DiffRow, index: number, activeHunkKey: string) {
    const hunkKey = row.type === 'hunk' || row.type === 'meta' ? row.hunkKey : undefined;
    const className = [
        'git-diff-line',
        `git-diff-line-${row.type}`,
        hunkKey && hunkKey === activeHunkKey ? 'active-hunk' : '',
    ].filter(Boolean).join(' ');
    const text = rowText(row);
    return (
        <span className={className} data-hunk-key={hunkKey} key={`${index}-${text}`}>
            {text}
            {'\n'}
        </span>
    );
}

function renderSplitRow(row: DiffRow, index: number, activeHunkKey: string) {
    if (row.type === 'meta' || row.type === 'hunk') {
        const className = [
            'git-diff-split-row',
            `git-diff-split-${row.type}`,
            row.hunkKey === activeHunkKey ? 'active-hunk' : '',
        ].filter(Boolean).join(' ');
        return (
            <div className={className} data-hunk-key={row.hunkKey} key={`${index}-${row.content}`} role="row">
                <span>{row.content}</span>
            </div>
        );
    }
    if (row.type === 'context') {
        return (
            <div className="git-diff-split-row" key={`${index}-${row.oldLine}-${row.newLine}`} role="row">
                <DiffCell line={row.oldLine} text={row.oldText} type="context" />
                <DiffCell line={row.newLine} text={row.newText} type="context" />
            </div>
        );
    }
    if (row.type === 'delete') {
        return (
            <div className="git-diff-split-row" key={`${index}-${row.oldLine}`} role="row">
                <DiffCell line={row.oldLine} text={row.oldText} type="delete" />
                <DiffCell text="" type="blank" />
            </div>
        );
    }
    return (
        <div className="git-diff-split-row" key={`${index}-${row.newLine}`} role="row">
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
    return (
        <div className="git-change-group">
            <small>{label} ({changes.length})</small>
            {changes.length > 0 ? changes.slice(0, 12).map((change) => (
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

function collectHunks(kind: string, diff: string): HunkTarget[] {
    return diff
        .replace(/\r\n/g, '\n')
        .split('\n')
        .reduce<HunkTarget[]>((hunks, line) => {
            if (line.startsWith('@@')) {
                const index = hunks.length + 1;
                hunks.push({key: `${kind}-${index}`, label: `${kind} ${index}`});
            }
            return hunks;
        }, []);
}

function parseUnifiedDiff(kind: string, diff: string): DiffRow[] {
    const rows: DiffRow[] = [];
    let oldLine = 0;
    let newLine = 0;
    let hunkIndex = 0;

    for (const line of diff.replace(/\r\n/g, '\n').split('\n')) {
        if (line === '') {
            rows.push({type: 'context', oldLine: oldLine > 0 ? oldLine++ : 0, newLine: newLine > 0 ? newLine++ : 0, oldText: '', newText: ''});
            continue;
        }
        if (line.startsWith('@@')) {
            hunkIndex += 1;
            const range = line.match(/^@@\s+-(\d+)(?:,\d+)?\s+\+(\d+)(?:,\d+)?\s+@@/);
            oldLine = Number(range?.[1] ?? 0);
            newLine = Number(range?.[2] ?? 0);
            rows.push({type: 'hunk', content: line, hunkKey: `${kind}-${hunkIndex}`});
            continue;
        }
        if (line.startsWith('diff --git') || line.startsWith('index ') || line.startsWith('---') || line.startsWith('+++')) {
            rows.push({type: 'meta', content: line});
            continue;
        }
        if (line.startsWith('-')) {
            rows.push({type: 'delete', oldLine: oldLine, oldText: line.slice(1)});
            oldLine += 1;
            continue;
        }
        if (line.startsWith('+')) {
            rows.push({type: 'add', newLine: newLine, newText: line.slice(1)});
            newLine += 1;
            continue;
        }
        const text = line.startsWith(' ') ? line.slice(1) : line;
        rows.push({type: 'context', oldLine: oldLine, newLine: newLine, oldText: text, newText: text});
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
