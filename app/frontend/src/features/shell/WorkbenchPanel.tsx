import {useEffect, useRef, useState} from 'react';
import {brandAssets, capabilityIconByTitle} from '../../brand/assets';
import {Button, EmptyState, InlineAlert, LoadingState, StatusBadge} from '../../components/ui';
import type {Capability, ColumnProfile, DatasetProfile, DatasetQueryResult, FilePreview, FileWriteProposal, TablePreview, WorkspaceArtifact, WorkspaceSnapshot} from '../../types';
import {ChatMessageContent} from './ChatMessageContent';
import {HighlightedCode} from './HighlightedCode';
import {MonacoCodePreview} from './MonacoCodePreview';
import {MonacoFileEditor} from './MonacoFileEditor';

type WorkbenchPanelProps = {
    activeFile: string;
    activeDatasetProfile: DatasetProfile | null;
    artifacts: WorkspaceArtifact[];
    capabilities: Capability[];
    datasetProfiles: DatasetProfile[];
    datasetChartCategory: string;
    datasetChartType: string;
    datasetChartValue: string;
    datasetQuery: string;
    datasetQueryResult: DatasetQueryResult | null;
    dirtyTabPaths: string[];
    fileDraft: string;
    filePreview: FilePreview | null;
    isApplyingWrite: boolean;
    isEditingFile: boolean;
    isSendingPrompt: boolean;
    isCreatingReport: boolean;
    isCreatingDatasetChart: boolean;
    isDeletingFile: boolean;
    isMovingFile: boolean;
    isProfilingDataset: boolean;
    isQueryingDataset: boolean;
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
    onDatasetChartCategoryChange: (content: string) => void;
    onDatasetChartTypeChange: (content: string) => void;
    onDatasetChartValueChange: (content: string) => void;
    onDeleteFile: () => void;
    onMoveFile: () => void;
    onPinContext: () => void;
    onPinProjectContext: () => void;
    onPreviewFileWrite: () => void;
    onProfileDataset: () => void;
    onQueryDataset: () => void;
    onCreateDatasetChart: () => void;
    onCloseTab: (relPath: string) => void;
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
    capabilities,
    datasetProfiles,
    datasetChartCategory,
    datasetChartType,
    datasetChartValue,
    datasetQuery,
    datasetQueryResult,
    dirtyTabPaths,
    fileDraft,
    filePreview,
    isApplyingWrite,
    isEditingFile,
    isSendingPrompt,
    isCreatingReport,
    isCreatingDatasetChart,
    isDeletingFile,
    isMovingFile,
    isProfilingDataset,
    isQueryingDataset,
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
    onDatasetChartCategoryChange,
    onDatasetChartTypeChange,
    onDatasetChartValueChange,
    onDeleteFile,
    onMoveFile,
    onPinContext,
    onPinProjectContext,
    onPreviewFileWrite,
    onProfileDataset,
    onQueryDataset,
    onCreateDatasetChart,
    onCloseTab,
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
                            {activeDatasetProfile && <DatasetProfileSummary profile={activeDatasetProfile} />}
                            {(activeDatasetProfile || filePreview?.table) && (
                                <>
                                    <DatasetQueryPanel
                                        query={datasetQuery}
                                        result={datasetQueryResult}
                                        isQuerying={isQueryingDataset}
                                        onChange={onDatasetQueryChange}
                                        onQuery={onQueryDataset}
                                    />
                                    <DatasetChartPanel
                                        categoryColumn={datasetChartCategory}
                                        chartType={datasetChartType}
                                        columns={filePreview?.table?.columns ?? activeDatasetProfile?.profiles.map((profile) => profile.name) ?? []}
                                        isCreating={isCreatingDatasetChart}
                                        onCategoryChange={onDatasetChartCategoryChange}
                                        onChartTypeChange={onDatasetChartTypeChange}
                                        onCreate={onCreateDatasetChart}
                                        onValueChange={onDatasetChartValueChange}
                                        profiles={filePreview?.table?.profiles ?? activeDatasetProfile?.profiles ?? []}
                                        valueColumn={datasetChartValue}
                                    />
                                </>
                            )}
                            {artifacts.length === 0 ? (
                                <EmptyState
                                    detail="Create a report to add the first workspace artifact."
                                    iconSrc={brandAssets.icons.documents}
                                    title="No artifacts yet"
                                />
                            ) : artifacts.map((artifact) => (
                                <button className="artifact-item" key={artifact.relPath} onClick={() => onSelectArtifact(artifact)}>
                                    <img src={artifact.kind === 'chart-svg' ? brandAssets.icons.data : brandAssets.icons.documents} alt="" />
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

function DatasetQueryPanel({
    query,
    result,
    isQuerying,
    onChange,
    onQuery,
}: {
    query: string;
    result: DatasetQueryResult | null;
    isQuerying: boolean;
    onChange: (value: string) => void;
    onQuery: () => void;
}) {
    return (
        <div className="dataset-query-panel">
            <strong>Dataset Query</strong>
            <div className="dataset-query-row">
                <input
                    aria-label="Dataset query"
                    onChange={(event) => onChange(event.target.value)}
                    onKeyDown={(event) => {
                        if (event.key === 'Enter') {
                            onQuery();
                        }
                    }}
                    placeholder="Search rows or use column=value"
                    value={query}
                />
                <Button disabled={isQuerying} onClick={onQuery} variant="subtle">
                    {isQuerying ? 'Querying...' : 'Run'}
                </Button>
            </div>
            {result && (
                <div className="dataset-query-result">
                    <small>{result.message}</small>
                    <CsvTablePreview table={{
                        columns: result.columns,
                        rows: result.rows,
                        profiles: [],
                        totalRows: result.matchedRows,
                        truncated: result.rows.length < result.matchedRows,
                    }} />
                </div>
            )}
        </div>
    );
}

function DatasetChartPanel({
    categoryColumn,
    chartType,
    columns,
    isCreating,
    onCategoryChange,
    onChartTypeChange,
    onCreate,
    onValueChange,
    profiles,
    valueColumn,
}: {
    categoryColumn: string;
    chartType: string;
    columns: string[];
    isCreating: boolean;
    onCategoryChange: (value: string) => void;
    onChartTypeChange: (value: string) => void;
    onCreate: () => void;
    onValueChange: (value: string) => void;
    profiles: ColumnProfile[];
    valueColumn: string;
}) {
    const numericColumns = profiles
        .filter((profile) => profile.type === 'integer' || profile.type === 'number')
        .map((profile) => profile.name)
        .filter((name) => columns.includes(name));

    return (
        <div className="dataset-chart-panel">
            <strong>Dataset Chart</strong>
            <div className="dataset-chart-grid">
                <label>
                    <span>Type</span>
                    <select aria-label="Chart type" onChange={(event) => onChartTypeChange(event.target.value)} value={chartType}>
                        <option value="bar">Bar</option>
                    </select>
                </label>
                <label>
                    <span>Category</span>
                    <select aria-label="Chart category column" onChange={(event) => onCategoryChange(event.target.value)} value={categoryColumn}>
                        {columns.map((column) => (
                            <option key={column} value={column}>{column}</option>
                        ))}
                    </select>
                </label>
                <label>
                    <span>Value</span>
                    <select aria-label="Chart value column" onChange={(event) => onValueChange(event.target.value)} value={valueColumn}>
                        <option value="">Count rows</option>
                        {numericColumns.map((column) => (
                            <option key={column} value={column}>Sum {column}</option>
                        ))}
                    </select>
                </label>
                <Button disabled={isCreating || columns.length === 0 || !categoryColumn} onClick={onCreate} variant="subtle">
                    {isCreating ? 'Creating...' : 'Create chart'}
                </Button>
            </div>
        </div>
    );
}

function DatasetProfileSummary({profile}: {profile: DatasetProfile}) {
    return (
        <div className="dataset-profile-summary">
            <strong>{profile.name}</strong>
            <small>{profile.kind}</small>
            {profile.kind === 'csv' ? (
                <p>{profile.rows} rows, {profile.columns} columns</p>
            ) : (
                <p>{profile.sheets.length} sheets: {profile.sheets.join(', ')}</p>
            )}
        </div>
    );
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
    return (
        <div className="csv-preview" aria-label="CSV table preview">
            {table.profiles.length > 0 && (
                <div className="csv-profile-strip" aria-label="CSV column profile">
                    {table.profiles.map((profile, index) => (
                        <div className="csv-profile" key={`${profile.name}-${index}`}>
                            <strong>{profile.name || `Column ${index + 1}`}</strong>
                            <span>{profile.type}</span>
                            <small>
                                {profile.distinct} distinct
                                {profile.missing > 0 ? `, ${profile.missing} missing` : ''}
                                {profile.min && profile.max ? `, ${profile.min}-${profile.max}` : ''}
                            </small>
                        </div>
                    ))}
                </div>
            )}
            <table>
                <thead>
                    <tr>
                        {table.columns.map((column, index) => (
                            <th key={`${column}-${index}`}>{column || `Column ${index + 1}`}</th>
                        ))}
                    </tr>
                </thead>
                <tbody>
                    {table.rows.map((row, rowIndex) => (
                        <tr key={rowIndex}>
                            {table.columns.map((_, columnIndex) => (
                                <td key={columnIndex}>{row[columnIndex] ?? ''}</td>
                            ))}
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}
