import {useEffect, useRef, useState} from 'react';
import {faBullseye, faListUl, faMap, faTableColumns, faThumbtack, faWandMagicSparkles, faXmark} from '@fortawesome/free-solid-svg-icons';
import {FontAwesomeIcon} from '@fortawesome/react-fontawesome';
import {brandAssets} from '../../brand/assets';
import {Button, EmptyState, InlineAlert, LoadingState} from '../../components/ui';
import type {FilePreview, FileWriteProposal, TablePreview, WorkspaceSnapshot} from '../../types';
import {ChatMessageContent} from './ChatMessageContent';
import {SortableDataTable} from './DataStudioPanel';
import {EditorOutlinePanel} from './EditorOutlinePanel';
import {buildEditorOutline} from './editorOutline';
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
    isSplitEditorEnabled: boolean;
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
    onSelectBreadcrumb: (relPath: string) => void;
    onSecondaryEditorChange: (relPath: string) => void;
    onCloseTab: (relPath: string) => void;
    onSelectTab: (relPath: string) => void;
    onToggleMinimap: () => void;
    onTogglePinTab: (relPath: string) => void;
    onToggleSplitEditor: () => void;
    onRefreshPreview: () => void;
    onStartFileEdit: () => void;
    openTabs: FilePreview[];
    pinnedTabPaths: string[];
    secondaryFile: string;
    secondaryPreview: FilePreview | null;
    selectedMeta: string;
    showMinimap: boolean;
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
    isSplitEditorEnabled,
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
    onSelectBreadcrumb,
    onSecondaryEditorChange,
    onCloseTab,
    onSelectTab,
    onToggleMinimap,
    onTogglePinTab,
    onToggleSplitEditor,
    onRefreshPreview,
    onStartFileEdit,
    openTabs,
    pinnedTabPaths,
    secondaryFile,
    secondaryPreview,
    selectedMeta,
    showMinimap,
    writeProposal,
    workspace,
}: WorkbenchPanelProps) {
    const [markdownViewMode, setMarkdownViewMode] = useState<'source' | 'rendered'>('source');
    const [findQuery, setFindQuery] = useState('');
    const [isOutlineVisible, setIsOutlineVisible] = useState(false);
    const [outlineTargetLine, setOutlineTargetLine] = useState(0);
    const [outlineTargetNonce, setOutlineTargetNonce] = useState(0);
    const [definitionNonce, setDefinitionNonce] = useState(0);
    const [formatNonce, setFormatNonce] = useState(0);
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
    const breadcrumbs = buildBreadcrumbs(activeFile, workspace?.name ?? 'Workspace');
    const secondaryOptions = openTabs.filter((tab) => tab.relPath !== activeFile);
    const outlineItems = buildEditorOutline(filePreview?.name ?? activeFile, findSource);
    const canUseDefinitionHook = Boolean(filePreview?.content && filePreview.kind === 'file');
    const canFormatDraft = Boolean(isEditingFile && filePreview?.content && filePreview.kind === 'file');

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
                            ) : openTabs.map((tab) => {
                                const isPinned = pinnedTabPaths.includes(tab.relPath);
                                return (
                                    <div
                                        aria-selected={activeFile === tab.relPath}
                                        className={[
                                            'editor-tab',
                                            activeFile === tab.relPath ? 'active' : '',
                                            isPinned ? 'pinned' : '',
                                        ].filter(Boolean).join(' ')}
                                        key={tab.relPath}
                                        onClick={() => onSelectTab(tab.relPath)}
                                        role="tab"
                                        title={tab.relPath}
                                    >
                                        <span>{tab.name}</span>
                                        {dirtyTabPaths.includes(tab.relPath) && <i aria-label="Unsaved changes" />}
                                        <small>{tab.kind === 'pdf' ? 'pdf' : tab.fileType}</small>
                                        <button
                                            aria-label={isPinned ? `Unpin ${tab.name}` : `Pin ${tab.name}`}
                                            className={isPinned ? 'tab-pin active' : 'tab-pin'}
                                            onClick={(event) => {
                                                event.stopPropagation();
                                                onTogglePinTab(tab.relPath);
                                            }}
                                            title={isPinned ? 'Unpin tab' : 'Pin tab'}
                                            type="button"
                                        >
                                            <FontAwesomeIcon icon={faThumbtack} />
                                        </button>
                                        <button
                                            aria-label={`Close ${tab.name}`}
                                            onClick={(event) => {
                                                event.stopPropagation();
                                                onCloseTab(tab.relPath);
                                            }}
                                            title="Close tab"
                                            type="button"
                                        >
                                            <FontAwesomeIcon icon={faXmark} />
                                        </button>
                                    </div>
                                );
                            })}
                        </div>
                        <div className="editor-tab-actions">
                            <button
                                aria-label="Go to definition"
                                className="editor-icon-toggle"
                                disabled={!canUseDefinitionHook}
                                onClick={() => {
                                    if (markdownViewMode === 'rendered') {
                                        setMarkdownViewMode('source');
                                    }
                                    setDefinitionNonce((current) => current + 1);
                                }}
                                title="Go to definition"
                                type="button"
                            >
                                <FontAwesomeIcon icon={faBullseye} />
                            </button>
                            <button
                                aria-label="Format document"
                                className="editor-icon-toggle"
                                disabled={!canFormatDraft}
                                onClick={() => setFormatNonce((current) => current + 1)}
                                title={canFormatDraft ? 'Format editable draft' : 'Start editing to format safely'}
                                type="button"
                            >
                                <FontAwesomeIcon icon={faWandMagicSparkles} />
                            </button>
                            <button
                                aria-pressed={isOutlineVisible}
                                className={isOutlineVisible ? 'editor-icon-toggle active' : 'editor-icon-toggle'}
                                onClick={() => setIsOutlineVisible((current) => !current)}
                                title={isOutlineVisible ? 'Hide outline' : 'Show outline'}
                                type="button"
                            >
                                <FontAwesomeIcon icon={faListUl} />
                            </button>
                            <button
                                aria-pressed={isSplitEditorEnabled}
                                className={isSplitEditorEnabled ? 'editor-icon-toggle active' : 'editor-icon-toggle'}
                                onClick={onToggleSplitEditor}
                                title={isSplitEditorEnabled ? 'Close split editor' : 'Open split editor'}
                                type="button"
                            >
                                <FontAwesomeIcon icon={faTableColumns} />
                            </button>
                            <button
                                aria-pressed={showMinimap}
                                className={showMinimap ? 'editor-icon-toggle active' : 'editor-icon-toggle'}
                                onClick={onToggleMinimap}
                                title={showMinimap ? 'Hide minimap' : 'Show minimap'}
                                type="button"
                            >
                                <FontAwesomeIcon icon={faMap} />
                            </button>
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
                    {breadcrumbs.length > 0 && (
                        <nav className="editor-breadcrumbs" aria-label="Editor breadcrumbs">
                            {breadcrumbs.map((crumb, index) => (
                                <button
                                    disabled={!crumb.relPath}
                                    key={`${crumb.relPath}-${index}`}
                                    onClick={() => onSelectBreadcrumb(crumb.relPath)}
                                    title={crumb.relPath || workspace?.root || crumb.label}
                                    type="button"
                                >
                                    {crumb.label}
                                </button>
                            ))}
                        </nav>
                    )}
                    {workspace ? (
                        <div className={isOutlineVisible ? 'editor-workspace-layout with-outline' : 'editor-workspace-layout'}>
                            <div className={isSplitEditorEnabled ? 'editor-split-layout' : 'editor-single-layout'}>
                                <section className="editor-group primary">
                                    {isSplitEditorEnabled && (
                                        <div className="editor-group-header">
                                            <strong>{filePreview?.name ?? activeFile}</strong>
                                            <small>Primary</small>
                                        </div>
                                    )}
                                    <PrimaryPreviewPane
                                        activeFile={activeFile}
                                        definitionNonce={definitionNonce}
                                        fileDraft={fileDraft}
                                        filePreview={filePreview}
                                        formatNonce={formatNonce}
                                        findQuery={findQuery}
                                        isApplyingWrite={isApplyingWrite}
                                        isDraftDirty={isDraftDirty}
                                        isEditingFile={isEditingFile}
                                        isLoadingPreview={isLoadingPreview}
                                        isPreviewingWrite={isPreviewingWrite}
                                        markdownViewMode={markdownViewMode}
                                        onApplyFileWrite={onApplyFileWrite}
                                        onCancelFileEdit={onCancelFileEdit}
                                        onFileDraftChange={onFileDraftChange}
                                        onPreviewFileWrite={onPreviewFileWrite}
                                        outlineTargetLine={outlineTargetLine}
                                        outlineTargetNonce={outlineTargetNonce}
                                        showMinimap={showMinimap}
                                        workspace={workspace}
                                        writeProposal={writeProposal}
                                    />
                                </section>
                                {isSplitEditorEnabled && (
                                    <section className="editor-group secondary">
                                        <div className="editor-group-header">
                                            <strong>{secondaryPreview?.name ?? 'No secondary tab'}</strong>
                                            <select
                                                aria-label="Secondary editor file"
                                                disabled={secondaryOptions.length === 0}
                                                onChange={(event) => onSecondaryEditorChange(event.target.value)}
                                                value={secondaryPreview?.relPath ?? secondaryFile}
                                            >
                                                {secondaryOptions.length === 0 ? (
                                                    <option value="">Open another tab</option>
                                                ) : secondaryOptions.map((tab) => (
                                                    <option key={tab.relPath} value={tab.relPath}>{tab.relPath}</option>
                                                ))}
                                            </select>
                                        </div>
                                        <SecondaryPreviewPane
                                            preview={secondaryPreview}
                                            showMinimap={showMinimap}
                                            workspace={workspace}
                                        />
                                    </section>
                                )}
                            </div>
                            {isOutlineVisible && (
                                <EditorOutlinePanel
                                    items={outlineItems}
                                    onSelect={(line) => {
                                        setOutlineTargetLine(line);
                                        setOutlineTargetNonce((current) => current + 1);
                                        if (markdownViewMode === 'rendered') {
                                            setMarkdownViewMode('source');
                                        }
                                    }}
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

function buildBreadcrumbs(activeFile: string, workspaceName: string) {
    const normalized = normalizeRelPath(activeFile);
    if (!normalized) {
        return [{label: workspaceName, relPath: ''}];
    }

    const parts = normalized.split('/').filter(Boolean);
    const crumbs = [{label: workspaceName, relPath: ''}];
    parts.forEach((part, index) => {
        crumbs.push({
            label: part,
            relPath: parts.slice(0, index + 1).join('/'),
        });
    });
    return crumbs;
}

function PrimaryPreviewPane({
    activeFile,
    definitionNonce,
    fileDraft,
    filePreview,
    formatNonce,
    findQuery,
    isApplyingWrite,
    isDraftDirty,
    isEditingFile,
    isLoadingPreview,
    isPreviewingWrite,
    markdownViewMode,
    onApplyFileWrite,
    onCancelFileEdit,
    onFileDraftChange,
    onPreviewFileWrite,
    outlineTargetLine,
    outlineTargetNonce,
    showMinimap,
    workspace,
    writeProposal,
}: {
    activeFile: string;
    definitionNonce: number;
    fileDraft: string;
    filePreview: FilePreview | null;
    formatNonce: number;
    findQuery: string;
    isApplyingWrite: boolean;
    isDraftDirty: boolean;
    isEditingFile: boolean;
    isLoadingPreview: boolean;
    isPreviewingWrite: boolean;
    markdownViewMode: 'source' | 'rendered';
    onApplyFileWrite: () => void;
    onCancelFileEdit: () => void;
    onFileDraftChange: (content: string) => void;
    onPreviewFileWrite: () => void;
    outlineTargetLine: number;
    outlineTargetNonce: number;
    showMinimap: boolean;
    workspace: WorkspaceSnapshot;
    writeProposal: FileWriteProposal | null;
}) {
    return (
        <div className="file-preview" aria-label="Workspace file preview">
            {isLoadingPreview ? (
                <LoadingState
                    detail="Reading the selected file inside the approved workspace root."
                    icon={brandAssets.icons.documents}
                    title="Loading preview"
                />
            ) : filePreview?.kind === 'image' && filePreview.content ? (
                <ImagePreview preview={filePreview} />
            ) : filePreview?.kind === 'directory' ? (
                <DirectoryPreviewSafe
                    directory={filePreview.relPath}
                    filePreviewMessage={filePreview.message}
                    workspace={workspace}
                />
            ) : filePreview?.kind === 'pdf' && filePreview.content ? (
                <PdfPreview preview={filePreview} />
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
                    definitionNonce={definitionNonce}
                    formatNonce={formatNonce}
                    revealLine={outlineTargetLine}
                    revealNonce={outlineTargetNonce}
                    showMinimap={showMinimap}
                />
            ) : filePreview?.content && markdownViewMode === 'rendered' && isMarkdownFile(filePreview.name) ? (
                <MarkdownPreview preview={filePreview} />
            ) : filePreview?.content ? (
                <div className={filePreview.message ? 'source-editor-preview' : 'source-editor-preview no-message'}>
                    {filePreview.message && <InlineAlert>{filePreview.message}</InlineAlert>}
                    <MonacoCodePreview content={filePreview.content} definitionNonce={definitionNonce} fileName={filePreview.name} revealLine={outlineTargetLine} revealNonce={outlineTargetNonce} searchQuery={findQuery} showMinimap={showMinimap} />
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
    );
}

function SecondaryPreviewPane({
    preview,
    showMinimap,
    workspace,
}: {
    preview: FilePreview | null;
    showMinimap: boolean;
    workspace: WorkspaceSnapshot;
}) {
    return (
        <div className="file-preview secondary-preview" aria-label="Secondary workspace file preview">
            {preview?.kind === 'image' && preview.content ? (
                <ImagePreview preview={preview} />
            ) : preview?.kind === 'directory' ? (
                <DirectoryPreviewSafe
                    directory={preview.relPath}
                    filePreviewMessage={preview.message}
                    workspace={workspace}
                />
            ) : preview?.kind === 'pdf' && preview.content ? (
                <PdfPreview preview={preview} />
            ) : preview?.table ? (
                <>
                    {preview.message && <InlineAlert>{preview.message}</InlineAlert>}
                    <CsvTablePreview table={preview.table} />
                </>
            ) : preview?.content && isMarkdownFile(preview.name) ? (
                <MarkdownPreview preview={preview} />
            ) : preview?.content ? (
                <div className={preview.message ? 'source-editor-preview' : 'source-editor-preview no-message'}>
                    {preview.message && <InlineAlert>{preview.message}</InlineAlert>}
                    <MonacoCodePreview content={preview.content} definitionNonce={0} fileName={preview.name} revealLine={0} revealNonce={0} searchQuery="" showMinimap={showMinimap} />
                </div>
            ) : (
                <EmptyState
                    detail={preview?.message ?? 'Open another tab to show it in the secondary editor group.'}
                    icon={brandAssets.icons.documents}
                    title={preview?.kind === 'unsupported' ? 'Preview unavailable' : 'No secondary file'}
                    tone={preview?.kind === 'unsupported' ? 'warning' : 'neutral'}
                />
            )}
        </div>
    );
}

function ImagePreview({preview}: {preview: FilePreview}) {
    return (
        <>
            {preview.message && <InlineAlert>{preview.message}</InlineAlert>}
            <div className="image-preview">
                <img src={preview.content} alt={preview.name} />
            </div>
        </>
    );
}

function PdfPreview({preview}: {preview: FilePreview}) {
    return (
        <>
            {preview.message && <InlineAlert>{preview.message}</InlineAlert>}
            <div className="document-preview">
                <iframe src={preview.content} title={preview.name} />
            </div>
            {preview.text && (
                <div className="document-text-preview">
                    <strong>Extracted text</strong>
                    {preview.pages && preview.pages.length > 0 ? (
                        preview.pages.map((page) => (
                            <p key={page.page}><strong>Page {page.page}</strong> {page.text}</p>
                        ))
                    ) : (
                        <p>{preview.text}</p>
                    )}
                </div>
            )}
        </>
    );
}

function MarkdownPreview({preview}: {preview: FilePreview}) {
    return (
        <>
            {preview.message && <InlineAlert>{preview.message}</InlineAlert>}
            <div className="markdown-document-preview">
                <ChatMessageContent content={preview.content} />
            </div>
        </>
    );
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
    definitionNonce,
    formatNonce,
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
    revealLine,
    revealNonce,
    showMinimap,
}: {
    draft: string;
    definitionNonce: number;
    formatNonce: number;
    fileName: string;
    revealLine: number;
    revealNonce: number;
    showMinimap: boolean;
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
                    definitionNonce={definitionNonce}
                    fileName={fileName}
                    formatNonce={formatNonce}
                    onChange={onChange}
                onSave={saveDraftShortcut}
                revealLine={revealLine}
                revealNonce={revealNonce}
                showMinimap={showMinimap}
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
