import {useEffect, useRef, useState} from 'react';
import {brandAssets, capabilityIconByTitle} from '../../brand/assets';
import {Button, EmptyState, InlineAlert, LoadingState, StatusBadge} from '../../components/ui';
import type {ApprovalRecord, ArtifactComparison, ArtifactLineage, ArtifactMetadata, Capability, ColumnProfile, DatasetChartResult, DatasetProfile, DatasetQueryResult, DatasetSQLQueryResult, FilePreview, FileWriteProposal, MetadataBrowser, SavedDatasetQuery, SQLiteMetadataStatus, TablePreview, WorkspaceArtifact, WorkspaceFreshnessStatus, WorkspaceSnapshot} from '../../types';
import {ApprovalLogPanel} from './ApprovalLogPanel';
import {ArtifactMetadataPanel} from './ArtifactMetadataPanel';
import {ChatMessageContent} from './ChatMessageContent';
import {DataStudioPanel, SortableDataTable} from './DataStudioPanel';
import {HighlightedCode} from './HighlightedCode';
import {MonacoCodePreview} from './MonacoCodePreview';
import {MonacoFileEditor} from './MonacoFileEditor';
import {OperationsInspector} from './OperationsInspector';

type WorkbenchPanelProps = {
    activeFile: string;
    activeDatasetProfile: DatasetProfile | null;
    artifacts: WorkspaceArtifact[];
    artifactMetadata: ArtifactMetadata | null;
    approvalRecords: ApprovalRecord[];
    capabilities: Capability[];
    datasetProfiles: DatasetProfile[];
    datasetChartCategory: string;
    datasetChartPreview: DatasetChartResult | null;
    datasetChartType: string;
    datasetChartValue: string;
    datasetQuery: string;
    datasetQueryLabel: string;
    datasetQueryResult: DatasetQueryResult | null;
    datasetSQLQuery: string;
    datasetSQLQueryResult: DatasetSQLQueryResult | null;
    metadataBrowser: MetadataBrowser | null;
    savedDatasetQueries: SavedDatasetQuery[];
    artifactComparison: ArtifactComparison | null;
    artifactLineage: ArtifactLineage | null;
    sqliteStatus: SQLiteMetadataStatus | null;
    workspaceFreshness: WorkspaceFreshnessStatus | null;
    dirtyTabPaths: string[];
    fileDraft: string;
    filePreview: FilePreview | null;
    isApplyingWrite: boolean;
    isEditingFile: boolean;
    isSendingPrompt: boolean;
    isCreatingReport: boolean;
    isCreatingDatasetChart: boolean;
    isExportingDatasetQuery: boolean;
    isArchivingArtifact: boolean;
    isDeletingArtifact: boolean;
    isDeletingFile: boolean;
    isMovingFile: boolean;
    isProfilingDataset: boolean;
    isPreviewingDatasetChart: boolean;
    isQueryingDataset: boolean;
    isQueryingDatasetSQL: boolean;
    isExportingDatasetSQL: boolean;
    isSavingDatasetQuery: boolean;
    isPreparingMetadataStore: boolean;
    isCreatingDatasetSummary: boolean;
    isSummarizingContext: boolean;
    isLoadingPreview: boolean;
    isPreviewingWrite: boolean;
    onApplyFileWrite: () => void;
    onCancelFileEdit: () => void;
    onExplainContext: () => void;
    onCreateReport: () => void;
    onSummarizeContext: () => void;
    onFileDraftChange: (content: string) => void;
    onDatasetQueryChange: (content: string) => void;
    onDatasetSQLQueryChange: (content: string) => void;
    onDatasetQueryLabelChange: (content: string) => void;
    onSaveDatasetQuery: () => void;
    onDatasetChartCategoryChange: (content: string) => void;
    onDatasetChartTypeChange: (content: string) => void;
    onDatasetChartValueChange: (content: string) => void;
    onDeleteFile: () => void;
    onMoveFile: () => void;
    onPinContext: () => void;
    onPinProjectContext: () => void;
    onPreviewFileWrite: () => void;
    onProfileDataset: () => void;
    onPreviewDatasetChart: () => void;
    onQueryDataset: () => void;
    onQueryDatasetSQL: () => void;
    onExportDatasetSQL: () => void;
    onCreateDatasetChart: () => void;
    onCreateDatasetSummary: () => void;
    onExportDatasetQuery: () => void;
    onArchiveArtifact: () => void;
    onCompareArtifact: () => void;
    onPrepareMetadataStore: () => void;
    onCloseTab: (relPath: string) => void;
    onDeleteArtifact: () => void;
    onOpenArtifactSource: () => void;
    onInspectMetadata: () => void;
    onRefreshLineage: () => void;
    onSelectTab: (relPath: string) => void;
    onSelectArtifact: (artifact: WorkspaceArtifact) => void;
    onRefreshPreview: () => void;
    onStartFileEdit: () => void;
    openTabs: FilePreview[];
    selectedMeta: string;
    writeProposal: FileWriteProposal | null;
    workspace: WorkspaceSnapshot | null;
};

export function WorkbenchPanel({
    activeFile,
    activeDatasetProfile,
    artifacts,
    artifactMetadata,
    approvalRecords,
    capabilities,
    datasetProfiles,
    datasetChartCategory,
    datasetChartPreview,
    datasetChartType,
    datasetChartValue,
    datasetQuery,
    datasetQueryLabel,
    datasetQueryResult,
    datasetSQLQuery,
    datasetSQLQueryResult,
    metadataBrowser,
    savedDatasetQueries,
    artifactComparison,
    artifactLineage,
    sqliteStatus,
    workspaceFreshness,
    dirtyTabPaths,
    fileDraft,
    filePreview,
    isApplyingWrite,
    isEditingFile,
    isSendingPrompt,
    isCreatingReport,
    isCreatingDatasetChart,
    isExportingDatasetQuery,
    isArchivingArtifact,
    isDeletingArtifact,
    isDeletingFile,
    isMovingFile,
    isProfilingDataset,
    isPreviewingDatasetChart,
    isQueryingDataset,
    isQueryingDatasetSQL,
    isExportingDatasetSQL,
    isSavingDatasetQuery,
    isPreparingMetadataStore,
    isCreatingDatasetSummary,
    isSummarizingContext,
    isLoadingPreview,
    isPreviewingWrite,
    onApplyFileWrite,
    onCancelFileEdit,
    onExplainContext,
    onCreateReport,
    onSummarizeContext,
    onFileDraftChange,
    onDatasetQueryChange,
    onDatasetSQLQueryChange,
    onDatasetQueryLabelChange,
    onSaveDatasetQuery,
    onDatasetChartCategoryChange,
    onDatasetChartTypeChange,
    onDatasetChartValueChange,
    onDeleteFile,
    onMoveFile,
    onPinContext,
    onPinProjectContext,
    onPreviewFileWrite,
    onProfileDataset,
    onPreviewDatasetChart,
    onQueryDataset,
    onQueryDatasetSQL,
    onExportDatasetSQL,
    onCreateDatasetChart,
    onCreateDatasetSummary,
    onExportDatasetQuery,
    onArchiveArtifact,
    onCompareArtifact,
    onPrepareMetadataStore,
    onCloseTab,
    onDeleteArtifact,
    onOpenArtifactSource,
    onInspectMetadata,
    onRefreshLineage,
    onSelectTab,
    onSelectArtifact,
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
    const canProfileDataset = Boolean(workspace && filePreview?.fileType === 'data');
    const canRenderMarkdown = Boolean(filePreview?.kind === 'file' && filePreview.content && isMarkdownFile(filePreview.name));
    const studioMode = resolveStudioMode(filePreview, activeDatasetProfile, activeFile);
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
                    <div className="studio-mode-strip" aria-label="Active studio surface">
                        <span>{studioMode.label}</span>
                        <small>{studioMode.detail}</small>
                    </div>
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
                    <Button disabled={!canProfileDataset || isProfilingDataset} onClick={onProfileDataset}>
                        {isProfilingDataset ? 'Profiling...' : 'Profile'}
                    </Button>
                    <Button disabled={!workspace || isPreparingMetadataStore} onClick={onPrepareMetadataStore} variant="subtle">
                        {isPreparingMetadataStore ? 'Preparing...' : 'Metadata'}
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
                                    iconSrc={brandAssets.icons.documents}
                                    title="Loading preview"
                                />
                            ) : filePreview?.kind === 'image' && filePreview.content ? (
                                <>
                                    {filePreview.message && <InlineAlert>{filePreview.message}</InlineAlert>}
                                    <div className="image-preview">
                                        <img src={filePreview.content} alt={filePreview.name} />
                                    </div>
                                </>
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
                                    iconSrc={brandAssets.icons.documents}
                                    title={filePreview?.kind === 'unsupported' ? 'Preview unavailable' : 'No file selected'}
                                    tone={filePreview?.kind === 'unsupported' ? 'warning' : 'neutral'}
                                />
                            )}
                        </div>
                    ) : (
                        <div className="code-preview" aria-label="NexusDesk workflow preview">
                            <p><span>01</span>Open a workspace root.</p>
                            <p><span>02</span>Index files, datasets, docs, and metadata.</p>
                            <p><span>03</span>Ask the agent with selected source context.</p>
                            <p><span>04</span>Approve writes, Docker actions, and database mutations.</p>
                            <p><span>05</span>Save reports, charts, diffs, and generated configs as artifacts.</p>
                        </div>
                    )}
                </article>

                <article className="status-pane">
                    <div className="pane-title">
                        <span>{workspace ? 'Artifacts & Data' : 'MVP Capabilities'}</span>
                        <small>{workspace ? `${artifacts.length} artifacts / ${datasetProfiles.length} profiles` : 'Phase 1 focus'}</small>
                    </div>
                    {workspace ? (
                        <div className="artifact-list">
                            {(activeDatasetProfile || filePreview?.table) && (
                                <DataStudioPanel
                                    activeDatasetProfile={activeDatasetProfile}
                                    chartCategory={datasetChartCategory}
                                    chartPreview={datasetChartPreview}
                                    chartType={datasetChartType}
                                    chartValue={datasetChartValue}
                                    columns={filePreview?.table?.columns ?? activeDatasetProfile?.profiles.map((profile) => profile.name) ?? []}
                                    isCreatingChart={isCreatingDatasetChart}
                                    isCreatingSummary={isCreatingDatasetSummary}
                                    isExporting={isExportingDatasetQuery}
                                    isPreviewingChart={isPreviewingDatasetChart}
                                    isQuerying={isQueryingDataset}
                                    isSavingQuery={isSavingDatasetQuery}
                                    onChartCategoryChange={onDatasetChartCategoryChange}
                                    onChartTypeChange={onDatasetChartTypeChange}
                                    onChartValueChange={onDatasetChartValueChange}
                                    onCreateChart={onCreateDatasetChart}
                                    onCreateSummary={onCreateDatasetSummary}
                                    onExportQuery={onExportDatasetQuery}
                                    onPreviewChart={onPreviewDatasetChart}
                                    onQuery={onQueryDataset}
                                    onQueryChange={onDatasetQueryChange}
                                    onQueryLabelChange={onDatasetQueryLabelChange}
                                    onSaveQuery={onSaveDatasetQuery}
                                    profiles={filePreview?.table?.profiles ?? activeDatasetProfile?.profiles ?? []}
                                    query={datasetQuery}
                                    queryLabel={datasetQueryLabel}
                                    queryResult={datasetQueryResult}
                                    sqlQuery={datasetSQLQuery}
                                    sqlResult={datasetSQLQueryResult}
                                    savedQueries={savedDatasetQueries}
                                    table={filePreview?.table ?? null}
                                    isQueryingSQL={isQueryingDatasetSQL}
                                    isExportingSQL={isExportingDatasetSQL}
                                    onSQLChange={onDatasetSQLQueryChange}
                                    onSQLQuery={onQueryDatasetSQL}
                                    onSQLExport={onExportDatasetSQL}
                                />
                            )}
                            <OperationsInspector preview={filePreview} workspace={workspace} />
                            {artifactMetadata && (
                                <ArtifactMetadataPanel
                                    isArchiving={isArchivingArtifact}
                                    isDeleting={isDeletingArtifact}
                                    metadata={artifactMetadata}
                                    onArchive={onArchiveArtifact}
                                    onCompare={onCompareArtifact}
                                    onDelete={onDeleteArtifact}
                                    onOpenSource={onOpenArtifactSource}
                                    preview={filePreview}
                                    relPath={filePreview?.relPath ?? ''}
                                />
                            )}
                            {artifactComparison && <ArtifactComparisonPanel comparison={artifactComparison} />}
                            {sqliteStatus && <MetadataStorePanel status={sqliteStatus} />}
                            {metadataBrowser && <MetadataBrowserPanel browser={metadataBrowser} />}
                            {workspaceFreshness && <WorkspaceFreshnessPanel status={workspaceFreshness} />}
                            <ArtifactLineagePanel lineage={artifactLineage} onRefresh={onRefreshLineage} />
                            <Button onClick={onInspectMetadata} variant="subtle">Inspect metadata</Button>
                            <ApprovalLogPanel records={approvalRecords} />
                            {artifacts.length === 0 ? (
                                <EmptyState
                                    detail="Create a report to add the first workspace artifact."
                                    iconSrc={brandAssets.icons.documents}
                                    title="No artifacts yet"
                                />
                            ) : artifacts.map((artifact) => (
                                <button className="artifact-item" key={artifact.relPath} onClick={() => onSelectArtifact(artifact)}>
                                    <img src={artifact.kind === 'chart-svg' || artifact.kind === 'dataset-query-csv' ? brandAssets.icons.data : brandAssets.icons.documents} alt="" />
                                    <span>
                                        <strong>{artifact.name}</strong>
                                        <small>{artifact.summary || artifact.source || artifact.relPath}</small>
                                        {artifact.model && <small>{artifact.model}</small>}
                                    </span>
                                    <StatusBadge tone="warning">{artifact.kind}</StatusBadge>
                                </button>
                            ))}
                        </div>
                    ) : (
                        <div className="capability-list">
                            {capabilities.map((capability) => (
                                <div className="capability-card" key={capability.title}>
                                    <img src={capabilityIconByTitle[capability.title] ?? brandAssets.icons.ai} alt="" />
                                    <strong>{capability.title}</strong>
                                    <p>{capability.description}</p>
                                    <StatusBadge tone="warning">{capability.status}</StatusBadge>
                                </div>
                            ))}
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

type StudioMode = {
    label: string;
    detail: string;
};

function resolveStudioMode(
    preview: FilePreview | null,
    datasetProfile: DatasetProfile | null,
    activeFile: string
): StudioMode {
    const relPath = (preview?.relPath || activeFile || '').replaceAll('\\', '/').toLowerCase();
    const fileName = (preview?.name || activeFile || '').toLowerCase();

    if (relPath.startsWith('.nexusdesk/artifacts/') || relPath.includes('/.nexusdesk/artifacts/')) {
        return {label: 'Artifact Studio', detail: 'Generated reports, summaries, and provenance'};
    }

    if (datasetProfile || preview?.table || preview?.fileType === 'data') {
        return {label: 'Data Studio', detail: 'Tables, profiles, bounded queries, and analysis context'};
    }

    if (preview?.kind === 'pdf' || isDocumentLikeFile(fileName)) {
        return {label: 'Document Studio', detail: 'Documents, Markdown, extracted text, and summaries'};
    }

    if (isOperationsFile(relPath, fileName)) {
        return {label: 'Operations Studio', detail: 'Docker, services, scripts, and environment files'};
    }

    if (preview?.fileType === 'code' || isCodeLikeFile(fileName)) {
        return {label: 'Code Studio', detail: 'Source files, editor tabs, explanations, and safe edits'};
    }

    return {label: 'Workspace Studio', detail: 'Project navigation, search, context packs, and artifacts'};
}

function isDocumentLikeFile(fileName: string) {
    return /\.(mdx?|docx?|pdf|rtf|txt)$/i.test(fileName);
}

function isCodeLikeFile(fileName: string) {
    return /\.(go|ts|tsx|js|jsx|json|css|scss|html|py|rs|cs|java|kt|sql|xml)$/i.test(fileName);
}

function isOperationsFile(relPath: string, fileName: string) {
    if (fileName === 'dockerfile' || fileName.includes('docker-compose') || relPath.startsWith('services/')) {
        return true;
    }

    return /\.(env|ps1|sh|bat|cmd|toml|ya?ml)$/i.test(fileName);
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

function ArtifactComparisonPanel({comparison}: {comparison: ArtifactComparison}) {
    return (
        <div className="artifact-comparison-panel">
            <strong>Artifact Comparison</strong>
            <small>{comparison.leftTitle} {'->'} {comparison.rightTitle}</small>
            <dl>
                <div><dt>Kind</dt><dd>{comparison.sameKind ? 'same' : 'different'}</dd></div>
                <div><dt>Size delta</dt><dd>{comparison.sizeDelta} bytes</dd></div>
            </dl>
            <div className="artifact-diff-grid">
                <span>
                    <strong>Removed</strong>
                    {comparison.removedLines.length === 0 ? <small>No removed lines</small> : comparison.removedLines.map((line) => <small key={line}>- {line}</small>)}
                </span>
                <span>
                    <strong>Added</strong>
                    {comparison.addedLines.length === 0 ? <small>No added lines</small> : comparison.addedLines.map((line) => <small key={line}>+ {line}</small>)}
                </span>
            </div>
        </div>
    );
}

function MetadataStorePanel({status}: {status: SQLiteMetadataStatus}) {
    return (
        <div className="metadata-store-panel">
            <strong>SQLite Metadata</strong>
            <small>{status.message}</small>
            <p>{status.tables.join(', ')}</p>
            <small>Schema v{status.schemaVersion}: {status.schemaHash.slice(0, 12)}</small>
        </div>
    );
}

function MetadataBrowserPanel({browser}: {browser: MetadataBrowser}) {
    return (
        <div className="metadata-store-panel metadata-browser-panel">
            <strong>Metadata Browser</strong>
            <small>{browser.message}</small>
            {browser.tables.map((table) => (
                <details key={table.name}>
                    <summary>{table.name} / {table.rowCount} rows</summary>
                    <small>{table.columns.map((column) => `${column.name}:${column.type}`).join(', ')}</small>
                    {table.sampleRows.length > 0 && (
                        <div className="metadata-sample">
                            {table.sampleRows.map((row, rowIndex) => (
                                <p key={`${table.name}-${rowIndex}`}>{row.slice(0, 4).join(' | ')}</p>
                            ))}
                        </div>
                    )}
                </details>
            ))}
            {browser.datasetViews.length > 0 && (
                <div className="metadata-dataset-views">
                    <strong>Dataset Views</strong>
                    {browser.datasetViews.map((view) => (
                        <p key={view.relPath}>{view.name}: {view.rows} rows, {view.columns.length} columns <small>{view.engine}</small></p>
                    ))}
                </div>
            )}
        </div>
    );
}

function WorkspaceFreshnessPanel({status}: {status: WorkspaceFreshnessStatus}) {
    if (status.changed.length === 0 && status.staleArtifacts.length === 0) {
        return null;
    }
    return (
        <div className="metadata-store-panel">
            <strong>Workspace Watcher</strong>
            <small>{status.message}</small>
            {status.changed.slice(0, 5).map((change) => (
                <p key={`${change.kind}-${change.relPath}`}>{change.kind}: {change.relPath}</p>
            ))}
            {status.staleArtifacts.length > 0 && (
                <small>Stale artifacts: {status.staleArtifacts.slice(0, 4).join(', ')}</small>
            )}
        </div>
    );
}

function ArtifactLineagePanel({lineage, onRefresh}: {lineage: ArtifactLineage | null; onRefresh: () => void}) {
    const [filter, setFilter] = useState('all');
    const visibleEdges = lineage?.edges.filter((edge) => {
        if (filter === 'all') {
            return true;
        }
        const from = lineage.nodes.find((node) => node.id === edge.from);
        const to = lineage.nodes.find((node) => node.id === edge.to);
        return from?.kind === filter || to?.kind === filter;
    }) ?? [];
    return (
        <div className="metadata-store-panel">
            <div className="panel-toolbar">
                <strong>Artifact Lineage</strong>
                <Button onClick={onRefresh} variant="subtle">Refresh</Button>
            </div>
            <small>{lineage?.message ?? 'Build graph from chats, tools, source files, and artifacts.'}</small>
            {lineage && (
                <div className="lineage-filter-row" aria-label="Lineage filter">
                    {['all', 'source', 'chat', 'tool', 'artifact'].map((kind) => (
                        <button className={filter === kind ? 'selected' : ''} key={kind} onClick={() => setFilter(kind)}>
                            {kind}
                        </button>
                    ))}
                </div>
            )}
            {lineage && (
                <div className="lineage-list">
                    {visibleEdges.slice(0, 8).map((edge, index) => {
                        const from = lineage.nodes.find((node) => node.id === edge.from);
                        const to = lineage.nodes.find((node) => node.id === edge.to);
                        return (
                            <p key={`${edge.from}-${edge.to}-${index}`}>
                                {from?.label ?? edge.from} {'->'} {to?.label ?? edge.to} <small>{edge.label}</small>
                            </p>
                        );
                    })}
                </div>
            )}
        </div>
    );
}
