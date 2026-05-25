import {useEffect, useState} from 'react';
import {FontAwesomeIcon} from '@fortawesome/react-fontawesome';
import {brandAssets, capabilityIconByTitle} from '../../brand/assets';
import {Button, EmptyState, StatusBadge} from '../../components/ui';
import type {Capability, DatasetChartResult, DatasetDependency, DatasetProfile, DatasetQueryResult, DatasetSQLQueryResult, FilePreview, MetadataBrowser, MetadataSearchResult, SavedDatasetQuery, SQLRun, SQLiteMetadataStatus, SQLiteQueryResult, WorkspaceFreshnessStatus, WorkspaceSnapshot} from '../../types';
import {DataStudioPanel, SortableDataTable} from './DataStudioPanel';
import {OperationsInspector} from './OperationsInspector';

type DataOperationsPanelProps = {
    activeDatasetProfile: DatasetProfile | null;
    capabilities: Capability[];
    datasetProfiles: DatasetProfile[];
    datasetDependencies: DatasetDependency[];
    datasetSQLRuns: SQLRun[];
    datasetChartCategory: string;
    datasetChartPreview: DatasetChartResult | null;
    datasetChartType: string;
    datasetChartValue: string;
    datasetQuery: string;
    datasetQueryLabel: string;
    datasetQueryResult: DatasetQueryResult | null;
    datasetSQLQuery: string;
    datasetSQLQueryLabel: string;
    datasetSQLQueryResult: DatasetSQLQueryResult | null;
    filePreview: FilePreview | null;
    isCreatingDatasetChart: boolean;
    isCreatingDatasetSummary: boolean;
    isExportingDatasetQuery: boolean;
    isExportingDatasetSQL: boolean;
    isPreparingMetadataStore: boolean;
    isProfilingDataset: boolean;
    isPreviewingDatasetChart: boolean;
    isQueryingDataset: boolean;
    isQueryingDatasetSQL: boolean;
    isQueryingSQLiteConnector: boolean;
    isRefreshingStaleContext: boolean;
    isSavingDatasetQuery: boolean;
    isSavingDatasetSQLQuery: boolean;
    isSearchingMetadata: boolean;
    metadataBrowser: MetadataBrowser | null;
    metadataSearchQuery: string;
    metadataSearchResults: MetadataSearchResult[];
    onCreateDatasetChart: () => void;
    onCreateDatasetSummary: () => void;
    onDatasetChartCategoryChange: (content: string) => void;
    onDatasetChartTypeChange: (content: string) => void;
    onDatasetChartValueChange: (content: string) => void;
    onDatasetQueryChange: (content: string) => void;
    onDatasetQueryLabelChange: (content: string) => void;
    onDatasetSQLQueryChange: (content: string) => void;
    onDatasetSQLQueryLabelChange: (content: string) => void;
    onExportDatasetQuery: () => void;
    onExportDatasetSQL: () => void;
    onInspectMetadata: () => void;
    onMetadataSearchQueryChange: (content: string) => void;
    onPrepareMetadataStore: () => void;
    onProfileDataset: () => void;
    onPreviewDatasetChart: () => void;
    onQueryDataset: () => void;
    onQueryDatasetSQL: () => void;
    onQuerySQLiteConnector: () => void;
    onRebuildDatasetDependency: (dependencyId: string) => void;
    onRefreshStaleContext: () => void;
    onSaveDatasetQuery: () => void;
    onSaveDatasetSQLQuery: () => void;
    onSearchMetadata: () => void;
    onSQLiteConnectorQueryChange: (content: string) => void;
    rebuildingDatasetDependencyId: string;
    savedDatasetQueries: SavedDatasetQuery[];
    savedDatasetSQLQueries: SavedDatasetQuery[];
    sqliteConnectorQuery: string;
    sqliteConnectorResult: SQLiteQueryResult | null;
    sqliteStatus: SQLiteMetadataStatus | null;
    workspace: WorkspaceSnapshot | null;
    workspaceFreshness: WorkspaceFreshnessStatus | null;
};

export function DataOperationsPanel({
    activeDatasetProfile,
    capabilities,
    datasetProfiles,
    datasetDependencies,
    datasetSQLRuns,
    datasetChartCategory,
    datasetChartPreview,
    datasetChartType,
    datasetChartValue,
    datasetQuery,
    datasetQueryLabel,
    datasetQueryResult,
    datasetSQLQuery,
    datasetSQLQueryLabel,
    datasetSQLQueryResult,
    filePreview,
    isCreatingDatasetChart,
    isCreatingDatasetSummary,
    isExportingDatasetQuery,
    isExportingDatasetSQL,
    isPreparingMetadataStore,
    isProfilingDataset,
    isPreviewingDatasetChart,
    isQueryingDataset,
    isQueryingDatasetSQL,
    isQueryingSQLiteConnector,
    isRefreshingStaleContext,
    isSavingDatasetQuery,
    isSavingDatasetSQLQuery,
    isSearchingMetadata,
    metadataBrowser,
    metadataSearchQuery,
    metadataSearchResults,
    onCreateDatasetChart,
    onCreateDatasetSummary,
    onDatasetChartCategoryChange,
    onDatasetChartTypeChange,
    onDatasetChartValueChange,
    onDatasetQueryChange,
    onDatasetQueryLabelChange,
    onDatasetSQLQueryChange,
    onDatasetSQLQueryLabelChange,
    onExportDatasetQuery,
    onExportDatasetSQL,
    onInspectMetadata,
    onMetadataSearchQueryChange,
    onPrepareMetadataStore,
    onProfileDataset,
    onPreviewDatasetChart,
    onQueryDataset,
    onQueryDatasetSQL,
    onQuerySQLiteConnector,
    onRebuildDatasetDependency,
    onRefreshStaleContext,
    onSaveDatasetQuery,
    onSaveDatasetSQLQuery,
    onSearchMetadata,
    onSQLiteConnectorQueryChange,
    rebuildingDatasetDependencyId,
    savedDatasetQueries,
    savedDatasetSQLQueries,
    sqliteConnectorQuery,
    sqliteConnectorResult,
    sqliteStatus,
    workspace,
    workspaceFreshness,
}: DataOperationsPanelProps) {
    if (!workspace) {
        return (
            <div className="data-operations-panel">
                <div className="capability-list">
                    {capabilities.map((capability) => (
                        <div className="capability-card" key={capability.title}>
                            <FontAwesomeIcon icon={capabilityIconByTitle[capability.title] ?? brandAssets.icons.ai} />
                            <strong>{capability.title}</strong>
                            <p>{capability.description}</p>
                            <StatusBadge tone="warning">{capability.status}</StatusBadge>
                        </div>
                    ))}
                </div>
            </div>
        );
    }

    const hasDataStudio = Boolean(activeDatasetProfile || filePreview?.table);
    const canProfileDataset = Boolean(workspace && filePreview?.fileType === 'data');

    return (
        <div className="data-operations-panel">
            <div className="data-operations-column primary">
                <div className="bottom-section-heading">
                    <strong>Data Studio</strong>
                    <small>{datasetProfiles.length} profiles available for this workspace.</small>
                    <Button disabled={!canProfileDataset || isProfilingDataset} onClick={onProfileDataset} variant="subtle">
                        {isProfilingDataset ? 'Profiling...' : 'Profile dataset'}
                    </Button>
                </div>
                {hasDataStudio ? (
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
                        onRebuildDependency={onRebuildDatasetDependency}
                        rebuildingDependencyId={rebuildingDatasetDependencyId}
                        profiles={filePreview?.table?.profiles ?? activeDatasetProfile?.profiles ?? []}
                        query={datasetQuery}
                        queryLabel={datasetQueryLabel}
                        queryResult={datasetQueryResult}
                        sqlQuery={datasetSQLQuery}
                        sqlLabel={datasetSQLQueryLabel}
                        sqlResult={datasetSQLQueryResult}
                        savedQueries={savedDatasetQueries}
                        savedSQLQueries={savedDatasetSQLQueries}
                        sqlRuns={datasetSQLRuns}
                        dependencies={datasetDependencies}
                        table={filePreview?.table ?? null}
                        isQueryingSQL={isQueryingDatasetSQL}
                        isExportingSQL={isExportingDatasetSQL}
                        isSavingSQL={isSavingDatasetSQLQuery}
                        onSQLChange={onDatasetSQLQueryChange}
                        onSQLLabelChange={onDatasetSQLQueryLabelChange}
                        onSQLQuery={onQueryDatasetSQL}
                        onSQLExport={onExportDatasetSQL}
                        onSQLSave={onSaveDatasetSQLQuery}
                    />
                ) : (
                    <EmptyState
                        detail="Select a CSV, spreadsheet, or saved profile to work with data."
                        icon={brandAssets.icons.data}
                        title="No active dataset"
                    />
                )}
            </div>

            <div className="data-operations-column">
                <div className="bottom-section-heading">
                    <strong>Operations</strong>
                    <small>Read-only service, database, and workspace metadata surfaces.</small>
                </div>
                <OperationsInspector preview={filePreview} workspace={workspace} />
                {filePreview?.fileType === 'database' && (
                    <SQLiteConnectorPanel
                        isQuerying={isQueryingSQLiteConnector}
                        onChange={onSQLiteConnectorQueryChange}
                        onQuery={onQuerySQLiteConnector}
                        query={sqliteConnectorQuery}
                        result={sqliteConnectorResult}
                    />
                )}
                {workspaceFreshness && (
                    <WorkspaceFreshnessPanel
                        isRefreshing={isRefreshingStaleContext}
                        onRefreshContext={onRefreshStaleContext}
                        status={workspaceFreshness}
                    />
                )}
            </div>

            <div className="data-operations-column">
                <div className="bottom-section-heading">
                    <strong>Metadata</strong>
                    <small>SQLite metadata mirror and history search.</small>
                </div>
                {sqliteStatus && <MetadataStorePanel status={sqliteStatus} />}
                {metadataBrowser && (
                    <MetadataBrowserPanel
                        browser={metadataBrowser}
                        isSearching={isSearchingMetadata}
                        onQueryChange={onMetadataSearchQueryChange}
                        onSearch={onSearchMetadata}
                        query={metadataSearchQuery}
                        results={metadataSearchResults}
                    />
                )}
                <div className="metadata-action-row">
                    <Button disabled={isPreparingMetadataStore} onClick={onPrepareMetadataStore} variant="subtle">
                        {isPreparingMetadataStore ? 'Preparing...' : 'Prepare metadata'}
                    </Button>
                    <Button onClick={onInspectMetadata} variant="subtle">Inspect metadata</Button>
                </div>
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

function MetadataBrowserPanel({
    browser,
    isSearching,
    onQueryChange,
    onSearch,
    query,
    results,
}: {
    browser: MetadataBrowser;
    isSearching: boolean;
    onQueryChange: (value: string) => void;
    onSearch: () => void;
    query: string;
    results: MetadataSearchResult[];
}) {
    const [columnQuery, setColumnQuery] = useState('');
    const [selectedTable, setSelectedTable] = useState(browser.tables[0]?.name ?? '');
    const normalizedQuery = columnQuery.trim().toLowerCase();
    const selected = browser.tables.find((table) => table.name === selectedTable) ?? browser.tables[0] ?? null;
    const visibleColumns = selected?.columns.filter((column) => {
        if (!normalizedQuery) {
            return true;
        }
        return column.name.toLowerCase().includes(normalizedQuery) || column.type.toLowerCase().includes(normalizedQuery);
    }) ?? [];
    const visibleColumnIndexes = visibleColumns.map((column) => selected?.columns.findIndex((item) => item.name === column.name) ?? -1).filter((index) => index >= 0);
    const sampleText = selected ? selected.sampleRows.map((row) => visibleColumnIndexes.map((index) => row[index] ?? '').join('\t')).join('\n') : '';

    useEffect(() => {
        setSelectedTable((current) => browser.tables.some((table) => table.name === current) ? current : browser.tables[0]?.name ?? '');
    }, [browser]);

    return (
        <div className="metadata-store-panel metadata-browser-panel">
            <strong>Metadata Browser</strong>
            <small>{browser.message}</small>
            <div className="metadata-browser-controls">
                <select aria-label="Metadata table" onChange={(event) => setSelectedTable(event.target.value)} value={selected?.name ?? ''}>
                    {browser.tables.map((table) => (
                        <option key={table.name} value={table.name}>{table.name} / {table.rowCount}</option>
                    ))}
                </select>
                <input aria-label="Column search" onChange={(event) => setColumnQuery(event.target.value)} placeholder="Search columns" value={columnQuery} />
                <Button disabled={!sampleText} onClick={() => void navigator.clipboard?.writeText(sampleText)} variant="subtle">Copy rows</Button>
            </div>
            <div className="metadata-browser-controls">
                <input
                    aria-label="Metadata history search"
                    onChange={(event) => onQueryChange(event.target.value)}
                    onKeyDown={(event) => {
                        if (event.key === 'Enter') {
                            onSearch();
                        }
                    }}
                    placeholder="Search chats, artifacts, tools"
                    value={query}
                />
                <Button disabled={isSearching || !query.trim()} onClick={onSearch} variant="subtle">
                    {isSearching ? 'Searching...' : 'Search history'}
                </Button>
            </div>
            {results.length > 0 && (
                <div className="metadata-history-results">
                    {results.map((result) => (
                        <p key={`${result.kind}-${result.id}`}>
                            <strong>{result.kind}</strong> {result.title} <small>{result.target}</small>
                            <span>{result.snippet}</span>
                        </p>
                    ))}
                </div>
            )}
            {selected && (
                <>
                    <div className="metadata-column-grid">
                        {visibleColumns.map((column) => (
                            <span key={column.name}><strong>{column.name}</strong><small>{column.type || 'TEXT'}</small></span>
                        ))}
                    </div>
                    {selected.sampleRows.length > 0 && (
                        <div className="metadata-sample">
                            {selected.sampleRows.map((row, rowIndex) => (
                                <p key={`${selected.name}-${rowIndex}`}>
                                    {visibleColumnIndexes.map((index) => row[index] ?? '').slice(0, 6).join(' | ')}
                                </p>
                            ))}
                        </div>
                    )}
                </>
            )}
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

function SQLiteConnectorPanel({
    isQuerying,
    onChange,
    onQuery,
    query,
    result,
}: {
    isQuerying: boolean;
    onChange: (value: string) => void;
    onQuery: () => void;
    query: string;
    result: SQLiteQueryResult | null;
}) {
    return (
        <div className="metadata-store-panel sqlite-connector-panel">
            <strong>SQLite Connector</strong>
            <small>Read-only workspace database query surface.</small>
            <textarea aria-label="Workspace SQLite query" onChange={(event) => onChange(event.target.value)} value={query} />
            <Button disabled={isQuerying || !query.trim()} onClick={onQuery} variant="subtle">
                {isQuerying ? 'Querying...' : 'Run read-only query'}
            </Button>
            {result && (
                <SortableDataTable
                    pageSize={8}
                    table={{columns: result.columns, rows: result.rows, profiles: [], totalRows: result.totalRows, truncated: result.rows.length < result.totalRows}}
                    title={result.engine}
                />
            )}
        </div>
    );
}

function WorkspaceFreshnessPanel({
    isRefreshing,
    onRefreshContext,
    status,
}: {
    isRefreshing: boolean;
    onRefreshContext: () => void;
    status: WorkspaceFreshnessStatus;
}) {
    const changed = safeWorkspaceChanges(status.changed);
    const staleArtifacts = safeStringArray(status.staleArtifacts);
    const staleDatasets = safeStringArray(status.staleDatasets);
    if (changed.length === 0 && staleArtifacts.length === 0 && staleDatasets.length === 0) {
        return null;
    }
    return (
        <div className="metadata-store-panel">
            <div className="panel-toolbar">
                <strong>Workspace Watcher</strong>
                <Button disabled={changed.length === 0 || isRefreshing} onClick={onRefreshContext} variant="subtle">
                    {isRefreshing ? 'Refreshing...' : 'Refresh context'}
                </Button>
            </div>
            <small>{status.message}</small>
            {changed.slice(0, 5).map((change) => (
                <p key={`${change.kind}-${change.relPath}`}>{change.kind}: {change.relPath}</p>
            ))}
            {staleArtifacts.length > 0 && (
                <small>Stale artifacts: {staleArtifacts.slice(0, 4).join(', ')}</small>
            )}
            {staleDatasets.length > 0 && (
                <small>Dataset refresh needed: {staleDatasets.slice(0, 4).join(', ')}</small>
            )}
        </div>
    );
}

function safeWorkspaceChanges(value: unknown) {
    return Array.isArray(value)
        ? value.filter((change): change is {relPath: string; kind: string} => {
            return Boolean(change && typeof change.relPath === 'string' && change.relPath.trim().length > 0);
        })
        : [];
}

function safeStringArray(value: unknown) {
    return Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string') : [];
}
