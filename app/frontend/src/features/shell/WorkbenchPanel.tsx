import {useEffect, useRef, useState} from 'react';
import {brandAssets} from '../../brand/assets';
import {Button, EmptyState, InlineAlert, LoadingState} from '../../components/ui';
import type {FilePreview, FileWriteProposal, TablePreview, WorkspaceSnapshot} from '../../types';
import {ChatMessageContent} from './ChatMessageContent';
import {SortableDataTable} from './DataStudioPanel';
import {HighlightedCode} from './HighlightedCode';
import {MonacoCodePreview} from './MonacoCodePreview';
import {MonacoFileEditor} from './MonacoFileEditor';

type WorkbenchPanelProps = {
    activeFile: string;
    dirtyTabPaths: string[];
    fileDraft: string;
    filePreview: FilePreview | null;
    isApplyingWrite: boolean;
    isEditingFile: boolean;
    isSendingPrompt: boolean;
    isCreatingReport: boolean;
    isDeletingFile: boolean;
    isMovingFile: boolean;
    isSummarizingContext: boolean;
    isLoadingPreview: boolean;
    isPreviewingWrite: boolean;
    onApplyFileWrite: () => void;
    onCancelFileEdit: () => void;
    onExplainContext: () => void;
    onCreateReport: () => void;
    onSummarizeContext: () => void;
    onFileDraftChange: (content: string) => void;
    onDeleteFile: () => void;
    onMoveFile: () => void;
    onPinContext: () => void;
    onPinProjectContext: () => void;
    onPreviewFileWrite: () => void;
    onCloseTab: (relPath: string) => void;
    onSelectTab: (relPath: string) => void;
    onRefreshPreview: () => void;
    onStartFileEdit: () => void;
    openTabs: FilePreview[];
    selectedMeta: string;
    writeProposal: FileWriteProposal | null;
    workspace: WorkspaceSnapshot | null;
};

export function WorkbenchPanel({
    activeFile,
    dirtyTabPaths,
    fileDraft,
    filePreview,
    isApplyingWrite,
    isEditingFile,
    isSendingPrompt,
    isCreatingReport,
    isDeletingFile,
    isMovingFile,
    isSummarizingContext,
    isLoadingPreview,
    isPreviewingWrite,
    onApplyFileWrite,
    onCancelFileEdit,
    onExplainContext,
    onCreateReport,
    onSummarizeContext,
    onFileDraftChange,
    onDeleteFile,
    onMoveFile,
    onPinContext,
    onPinProjectContext,
    onPreviewFileWrite,
    onCloseTab,
    onSelectTab,
    onRefreshPreview,
    onStartFileEdit,
    openTabs,
    selectedMeta,
    writeProposal,
    workspace,
}: WorkbenchPanelProps) {
    const [markdownViewMode, setMarkdownViewMode] = useState<'source' | 'rendered'>('source');
    const [findQuery, setFindQuery] = useState('');
    const findInputRef = useRef<HTMLInputElement>(null);
    const canExplainContext = Boolean(
        workspace && (
            (filePreview?.kind === 'file' && filePreview.content) ||
            (filePreview?.kind === 'pdf' && filePreview.text) ||
            filePreview?.kind === 'directory'
        )
    );
    const canEditContext = Boolean(workspace && filePreview?.kind === 'file' && !filePreview.table);
    const canDeleteContext = Boolean(workspace && filePreview?.kind === 'file' && !isEditingFile);
    const canMoveContext = Boolean(workspace && filePreview?.kind === 'file' && !isEditingFile);
    const canRenderMarkdown = Boolean(filePreview?.kind === 'file' && filePreview.content && isMarkdownFile(filePreview.name));
    const findSource = isEditingFile ? fileDraft : filePreview?.content ?? filePreview?.text ?? '';
    const findMatches = countFindMatches(findSource, findQuery);
    const isDraftDirty = Boolean(filePreview && dirtyTabPaths.includes(filePreview.relPath));

    useEffect(() => {
        function handleFindShortcut(event: KeyboardEvent) {
            if (!(event.ctrlKey || event.metaKey) || event.key.toLowerCase() !== 'f' || !filePreview?.content) {
                return;
            }

            event.preventDefault();
            findInputRef.current?.focus();
            findInputRef.current?.select();
        }

        window.addEventListener('keydown', handleFindShortcut);
        return () => window.removeEventListener('keydown', handleFindShortcut);
    }, [filePreview?.content]);

    return (
        <main className="workbench">
            <header className="topbar">
                <div>
                    <p className="eyebrow">Active Context</p>
                    <h2>{activeFile}</h2>
                </div>
                <div className="topbar-actions">
                    <Button disabled={!workspace || isLoadingPreview} onClick={onRefreshPreview}>
                        {isLoadingPreview ? 'Loading...' : 'Preview'}
                    </Button>
                    <Button disabled={!canExplainContext || isSendingPrompt} onClick={onExplainContext}>
                        {isSendingPrompt ? 'Sending...' : 'Explain'}
                    </Button>
                    <Button disabled={!canExplainContext || isSendingPrompt || isSummarizingContext} onClick={onSummarizeContext}>
                        {isSummarizingContext ? 'Summarizing...' : 'Summarize'}
                    </Button>
                    <Button disabled={!canExplainContext} onClick={onPinContext}>
                        Pin
                    </Button>
                    <Button disabled={!workspace} onClick={onPinProjectContext}>
                        Project
                    </Button>
                    <Button disabled={!canEditContext || isLoadingPreview} onClick={onStartFileEdit}>
                        Edit
                    </Button>
                    <Button disabled={!canDeleteContext || isDeletingFile} onClick={onDeleteFile} variant="subtle">
                        {isDeletingFile ? 'Deleting...' : 'Delete'}
                    </Button>
                    <Button disabled={!canMoveContext || isMovingFile} onClick={onMoveFile} variant="subtle">
                        {isMovingFile ? 'Moving...' : 'Rename'}
                    </Button>
                    <Button disabled={!workspace || isCreatingReport} onClick={onCreateReport}>
                        {isCreatingReport ? 'Creating...' : 'Report'}
                    </Button>
                </div>
            </header>

            <section className="canvas-grid">
                <article className="editor-pane">
                    <div className="editor-tabs">
                        <div className="tab-strip" role="tablist" aria-label="Open files">
                            {openTabs.length === 0 ? (
                                <span className="empty-tabs">No open files</span>
                            ) : openTabs.map((tab) => (
                                <div
                                    aria-selected={activeFile === tab.relPath}
                                    className={activeFile === tab.relPath ? 'editor-tab active' : 'editor-tab'}
                                    key={tab.relPath}
                                    onClick={() => onSelectTab(tab.relPath)}
                                    role="tab"
                                    title={tab.relPath}
                                >
                                    <span>{tab.name}</span>
                                    {dirtyTabPaths.includes(tab.relPath) && <i aria-label="Unsaved changes" />}
                                    <small>{tab.kind === 'pdf' ? 'pdf' : tab.fileType}</small>
                                    <button
                                        aria-label={`Close ${tab.name}`}
                                        onClick={(event) => {
                                            event.stopPropagation();
                                            onCloseTab(tab.relPath);
                                        }}
                                    >
                                        x
                                    </button>
                                </div>
                            ))}
                        </div>
                        <div className="editor-tab-actions">
                            {filePreview?.content && (
                                <div className="editor-find" aria-label="Find in file">
                                    <input
                                        aria-label="Find in file"
                                        onChange={(event) => setFindQuery(event.target.value)}
                                        placeholder="Find"
                                        ref={findInputRef}
                                        value={findQuery}
                                    />
                                    <small>{findQuery.trim() ? `${findMatches} matches` : 'Find in file'}</small>
                                </div>
                            )}
                            {canRenderMarkdown && (
                                <div className="markdown-view-toggle" aria-label="Markdown view mode">
                                    <button
                                        className={markdownViewMode !== 'rendered' ? 'active' : ''}
                                        onClick={() => setMarkdownViewMode('source')}
                                    >
                                        Source
                                    </button>
                                    <button
                                        className={markdownViewMode === 'rendered' ? 'active' : ''}
                                        onClick={() => setMarkdownViewMode('rendered')}
                                    >
                                        Preview
                                    </button>
                                </div>
                            )}
                            <small>{selectedMeta}</small>
                        </div>
                    </div>
                    {workspace ? (
                        <div className="file-preview" aria-label="Workspace file preview">
                            {isLoadingPreview ? (
                                <LoadingState
                                    detail="Reading the selected file inside the approved workspace root."
                                    icon={brandAssets.icons.documents}
                                    title="Loading preview"
                                />
                            ) : filePreview?.kind === 'image' && filePreview.content ? (
                                <>
                                    {filePreview.message && <InlineAlert>{filePreview.message}</InlineAlert>}
                                    <div className="image-preview">
                                        <img src={filePreview.content} alt={filePreview.name} />
                                    </div>
                                </>
                            ) : filePreview?.kind === 'directory' ? (
                                <DirectoryPreviewSafe
                                    directory={filePreview.relPath}
                                    filePreviewMessage={filePreview.message}
                                    workspace={workspace}
                                />
                            ) : filePreview?.kind === 'pdf' && filePreview.content ? (
                                <>
                                    {filePreview.message && <InlineAlert>{filePreview.message}</InlineAlert>}
                                    <div className="document-preview">
                                        <iframe src={filePreview.content} title={filePreview.name} />
                                    </div>
                                    {filePreview.text && (
                                        <div className="document-text-preview">
                                            <strong>Extracted text</strong>
                                            {filePreview.pages && filePreview.pages.length > 0 ? (
                                                filePreview.pages.map((page) => (
                                                    <p key={page.page}><strong>Page {page.page}</strong> {page.text}</p>
                                                ))
                                            ) : (
                                                <p>{filePreview.text}</p>
                                            )}
                                        </div>
                                    )}
                                </>
                            ) : filePreview?.table ? (
                                <>
                                    {filePreview.message && <InlineAlert>{filePreview.message}</InlineAlert>}
                                    <CsvTablePreview table={filePreview.table} />
                                </>
                            ) : isEditingFile ? (
                                <FileWriteEditor
                                    draft={fileDraft}
                                    isApplying={isApplyingWrite}
                                    isPreviewing={isPreviewingWrite}
                                    onApply={onApplyFileWrite}
                                    onCancel={onCancelFileEdit}
                                    onChange={onFileDraftChange}
                                    onPreview={onPreviewFileWrite}
                                    proposal={writeProposal}
                                    originalContent={filePreview?.content ?? ''}
                                    fileName={filePreview?.name ?? activeFile}
                                    isDirty={isDraftDirty}
                                />
                            ) : filePreview?.content && markdownViewMode === 'rendered' && isMarkdownFile(filePreview.name) ? (
                                <>
                                    {filePreview.message && <InlineAlert>{filePreview.message}</InlineAlert>}
                                    <div className="markdown-document-preview">
                                        <ChatMessageContent content={filePreview.content} />
                                    </div>
                                </>
                            ) : filePreview?.content ? (
                                <div className={filePreview.message ? 'source-editor-preview' : 'source-editor-preview no-message'}>
                                    {filePreview.message && <InlineAlert>{filePreview.message}</InlineAlert>}
                                    <MonacoCodePreview content={filePreview.content} fileName={filePreview.name} searchQuery={findQuery} />
                                </div>
                            ) : (
                                <EmptyState
                                    detail={filePreview?.message ?? 'Select a file from the workspace tree to preview it here.'}
                                    icon={brandAssets.icons.documents}
                                    title={filePreview?.kind === 'unsupported' ? 'Preview unavailable' : 'No file selected'}
                                    tone={filePreview?.kind === 'unsupported' ? 'warning' : 'neutral'}
                                />
                            )}
                        </div>
                    ) : (
                        <div className="code-preview" aria-label="Nexus workflow preview">
                            <p><span>01</span>Open a workspace root.</p>
                            <p><span>02</span>Index files, datasets, docs, and metadata.</p>
                            <p><span>03</span>Ask the agent with selected source context.</p>
                            <p><span>04</span>Approve writes, Docker actions, and database mutations.</p>
                            <p><span>05</span>Save reports, charts, diffs, and generated configs as artifacts.</p>
                        </div>
                    )}
                </article>
            </section>
        </main>
    );
}

function isMarkdownFile(fileName: string) {
    return /\.mdx?$/i.test(fileName);
}

function DirectoryPreviewSafe({
    directory,
    filePreviewMessage,
    workspace,
}: {
    directory: string;
    filePreviewMessage: string;
    workspace: WorkspaceSnapshot | null;
}) {
    const nodes = workspace?.nodes ?? [];
    const normalizedDirectory = normalizeRelPath(directory);
    const isRoot = normalizedDirectory === '';

    const directChildren = nodes.filter((node) => {
        if (!node || typeof node.relPath !== 'string') {
            return false;
        }

        const normalizedPath = normalizeRelPath(node.relPath);
        if (normalizedPath === normalizedDirectory) {
            return false;
        }
        if (isRoot) {
            return normalizedPath.length > 0 && !normalizedPath.includes('/');
        }
        if (!normalizedPath.startsWith(`${normalizedDirectory}/`)) {
            return false;
        }
        const remainder = normalizedPath.slice(normalizedDirectory.length + 1);
        return remainder.length > 0 && !remainder.includes('/');
    });

    const sortedChildren = [...directChildren]
        .sort((left, right) => {
            const leftKind = left.kind || '';
            const rightKind = right.kind || '';
            if (leftKind !== rightKind) {
                return leftKind === 'directory' ? -1 : 1;
            }
            return (left.name || '').localeCompare(right.name || '');
        })
        .slice(0, 120);

    return (
        <div className="directory-preview">
            <div className="directory-preview-heading">
                <strong>{directory || 'Workspace root'}</strong>
                <small>Directory preview</small>
            </div>
            {filePreviewMessage && <InlineAlert>{filePreviewMessage}</InlineAlert>}
            {sortedChildren.length > 0 ? (
                <div className="directory-entries">
                    {sortedChildren.map((child) => (
                        <div className="directory-entry" key={child.relPath}>
                            <span>{child.kind === 'directory' ? '[DIR]' : 'file'}</span>
                            <span>{child.name}</span>
                            <small>{child.meta ?? ''}</small>
                        </div>
                    ))}
                </div>
            ) : (
                <div className="directory-empty">No indexed items found in this directory.</div>
            )}
        </div>
    );
}

function normalizeRelPath(path: string) {
    return String(path || '')
        .replace(/\\/g, '/')
        .replace(/\/+$/g, '')
        .replace(/^\/+/, '');
}

function countFindMatches(content: string, query: string) {
    const needle = query.trim().toLowerCase();
    if (!needle || !content) {
        return 0;
    }

    let count = 0;
    let cursor = 0;
    const haystack = content.toLowerCase();
    while (cursor <= haystack.length) {
        const index = haystack.indexOf(needle, cursor);
        if (index === -1) {
            break;
        }
        count += 1;
        cursor = index + Math.max(needle.length, 1);
    }
    return count;
}

function FileWriteEditor({
    draft,
    isApplying,
    isPreviewing,
    onApply,
    onCancel,
    onChange,
    onPreview,
    proposal,
    originalContent,
    fileName,
    isDirty,
}: {
    draft: string;
    fileName: string;
    isApplying: boolean;
    isPreviewing: boolean;
    onApply: () => void;
    onCancel: () => void;
    onChange: (content: string) => void;
    onPreview: () => void;
    proposal: FileWriteProposal | null;
    originalContent: string;
    isDirty: boolean;
}) {
    function saveDraftShortcut() {
        if (proposal && !isApplying) {
            onApply();
            return;
        }
        if (isDirty && !isPreviewing && !isApplying) {
            onPreview();
        }
    }

    return (
        <div className="file-write-editor">
            <div className="write-toolbar">
                <span className={isDirty ? 'dirty-indicator dirty' : 'dirty-indicator'}>
                    {isDirty ? 'Unsaved changes' : 'No changes'}
                </span>
                <Button disabled={!isDirty || isPreviewing || isApplying} onClick={onPreview}>
                    {isPreviewing ? 'Previewing...' : 'Preview diff'}
                </Button>
                <Button disabled={!proposal || isApplying} onClick={onApply}>
                    {isApplying ? 'Applying...' : 'Apply'}
                </Button>
                <Button disabled={!isDirty || isApplying} onClick={() => onChange(originalContent)} variant="subtle">
                    Revert
                </Button>
                <Button disabled={isApplying} onClick={onCancel} variant="subtle">
                    Cancel
                </Button>
            </div>
            <MonacoFileEditor
                fileName={fileName}
                onChange={onChange}
                onSave={saveDraftShortcut}
                value={draft}
            />
            {proposal && (
                <div className="write-diff">
                    <InlineAlert>{proposal.message}</InlineAlert>
                    <HighlightedCode content={proposal.diff} fileName={`${proposal.name}.diff`} />
                </div>
            )}
        </div>
    );
}

function CsvTablePreview({table}: {table: TablePreview}) {
    return <SortableDataTable table={table} title="CSV Preview" />;
}
