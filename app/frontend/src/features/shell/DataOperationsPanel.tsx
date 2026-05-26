import {useEffect, useState} from 'react';
import {FontAwesomeIcon} from '@fortawesome/react-fontawesome';
import {brandAssets, capabilityIconByTitle} from '../../brand/assets';
import {Button, EmptyState, StatusBadge} from '../../components/ui';
import type {Capability, ConnectorMetadata, DatasetChartResult, DatasetDependency, DatasetProfile, DatasetQueryResult, DatasetSQLQueryResult, FileNode, FilePreview, MetadataBrowser, MetadataSearchResult, SavedDatasetQuery, SQLRun, SQLiteMetadataStatus, SQLiteQueryResult, WorkspaceFreshnessStatus, WorkspaceSnapshot} from '../../types';
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
    isInspectingSQLiteConnector: boolean;
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
    onInspectSQLiteConnector: () => void;
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
    sqliteConnectorMetadata: ConnectorMetadata | null;
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
    isInspectingSQLiteConnector,
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
    onInspectSQLiteConnector,
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
    sqliteConnectorMetadata,
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
    const dataSources = buildDataSourceCards(workspace.nodes, datasetProfiles, filePreview?.relPath ?? '');

    return (
        <div className="data-operations-panel">
            <div className="data-operations-column primary">
                <div className="bottom-section-heading">
                    <strong>Data & Analytics</strong>
                    <small>{datasetProfiles.length} profiles available for this workspace.</small>
                    <Button disabled={!canProfileDataset || isProfilingDataset} onClick={onProfileDataset} variant="subtle">
                        {isProfilingDataset ? 'Profiling...' : 'Profile dataset'}
                    </Button>
                </div>
                <DataSourceCards sources={dataSources} />
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
                        isInspecting={isInspectingSQLiteConnector}
                        metadata={sqliteConnectorMetadata}
                        onChange={onSQLiteConnectorQueryChange}
                        onInspect={onInspectSQLiteConnector}
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

type DataSourceCard = {
    relPath: string;
    name: string;
    category: string;
    status: 'profiled' | 'detected' | 'planned' | 'guidance';
    detail: string;
    meta: string;
    active: boolean;
};

function DataSourceCards({sources}: {sources: DataSourceCard[]}) {
    if (sources.length === 0) {
        return (
            <div className="data-source-panel">
                <strong>Data Sources</strong>
                <small>No dataset-like files detected in the bounded workspace tree.</small>
            </div>
        );
    }
    return (
        <div className="data-source-panel">
            <div className="data-source-heading">
                <strong>Data Sources</strong>
                <small>{sources.length} detected in the bounded workspace tree.</small>
            </div>
            <div className="data-source-grid">
                {sources.slice(0, 24).map((source) => (
                    <div className={source.active ? 'data-source-card active' : 'data-source-card'} key={source.relPath}>
                        <span>{source.category}</span>
                        <strong title={source.relPath}>{source.name}</strong>
                        <small>{source.detail}</small>
                        <StatusBadge tone={source.status === 'profiled' ? 'success' : source.status === 'guidance' ? 'warning' : 'neutral'}>
                            {source.status}
                        </StatusBadge>
                        <small>{source.meta}</small>
                    </div>
                ))}
            </div>
            {sources.length > 24 && <small>{sources.length - 24} more sources hidden by the card cap.</small>}
        </div>
    );
}

function buildDataSourceCards(nodes: FileNode[], profiles: DatasetProfile[], activeRelPath: string): DataSourceCard[] {
    const profiled = new Map(profiles.map((profile) => [profile.relPath, profile]));
    return nodes
        .filter((node) => node.kind === 'file')
        .map((node) => dataSourceFromNode(node, profiled.get(node.relPath), activeRelPath))
        .filter((source): source is DataSourceCard => Boolean(source))
        .sort(compareDataSources);
}

function dataSourceFromNode(node: FileNode, profile: DatasetProfile | undefined, activeRelPath: string): DataSourceCard | null {
    const extension = fileExtension(node.name);
    const active = node.relPath === activeRelPath;
    if (['csv', 'tsv', 'json', 'jsonl', 'ndjson'].includes(extension)) {
        return sourceCard(node, profile, active, 'table file', profile ? `${profile.rows} rows, ${profile.columns} columns` : 'Preview, profile, query, chart, and summarize.');
    }
    if (extension === 'xlsx') {
        const formulaCount = profile?.workbook?.formulaCount ?? 0;
        const tableCount = profile?.workbook?.tableRanges?.length ?? 0;
        return sourceCard(node, profile, active, 'workbook', profile ? `${profile.sheets.length} sheets, ${formulaCount} formulas, ${tableCount} tables` : 'Profile sheets, formulas, tables, named ranges, and pivots.');
    }
    if (extension === 'xls') {
        return plannedSourceCard(node, active, 'legacy workbook', 'Convert to XLSX or CSV before profiling.');
    }
    if (extension === 'parquet') {
        return sourceCard(node, profile, active, 'parquet', profile ? `${formatBytes(profile.parquet?.footerMetadataBytes ?? 0)} footer, ${formatBytes(profile.parquet?.dataBytes ?? 0)} data` : 'Footer metadata inspection available.');
    }
    if (['sqlite', 'sqlite3', 'db'].includes(extension)) {
        return sourceCard(node, profile, active, 'sqlite file', 'Read-only connector available separately from dataset profiles.');
    }
    if (['sql', 'dump', 'bak'].includes(extension)) {
        return plannedSourceCard(node, active, 'database dump', 'Dump classification detected; sandbox import is planned.');
    }
    if (['zip', 'gz', 'tgz', 'tar', 'bz2', 'xz', '7z'].includes(extension)) {
        return plannedSourceCard(node, active, 'compressed export', 'Archive/export detection only; import workflow is planned.');
    }
    if (['log', 'out', 'trace'].includes(extension) || node.name.toLowerCase().includes('log')) {
        return sourceCard(node, profile, active, 'log file', profile ? `${profile.log?.sampledLines ?? 0} sampled lines, ${levelSummary(profile.log?.levelCounts)}` : 'Profile levels, timestamps, stack traces, and repeated patterns.');
    }
    return null;
}

function sourceCard(node: FileNode, profile: DatasetProfile | undefined, active: boolean, category: string, detail: string): DataSourceCard {
    return {
        relPath: node.relPath,
        name: node.name,
        category,
        status: profile ? 'profiled' : 'detected',
        detail,
        meta: node.meta || node.relPath,
        active,
    };
}

function plannedSourceCard(node: FileNode, active: boolean, category: string, detail: string): DataSourceCard {
    return {
        relPath: node.relPath,
        name: node.name,
        category,
        status: category === 'legacy workbook' ? 'guidance' : 'planned',
        detail,
        meta: node.meta || node.relPath,
        active,
    };
}

function compareDataSources(left: DataSourceCard, right: DataSourceCard) {
    if (left.active !== right.active) {
        return left.active ? -1 : 1;
    }
    const statusOrder = {profiled: 0, detected: 1, guidance: 2, planned: 3};
    if (statusOrder[left.status] !== statusOrder[right.status]) {
        return statusOrder[left.status] - statusOrder[right.status];
    }
    return left.relPath.localeCompare(right.relPath, undefined, {sensitivity: 'base'});
}

function fileExtension(name: string) {
    const index = name.lastIndexOf('.');
    return index >= 0 ? name.slice(index + 1).toLowerCase() : '';
}

function formatBytes(value: number) {
    if (!Number.isFinite(value) || value <= 0) {
        return '0 B';
    }
    const units = ['B', 'KB', 'MB', 'GB'];
    let current = value;
    let unitIndex = 0;
    while (current >= 1024 && unitIndex < units.length - 1) {
        current /= 1024;
        unitIndex += 1;
    }
    return `${current >= 10 || unitIndex === 0 ? current.toFixed(0) : current.toFixed(1)} ${units[unitIndex]}`;
}

function levelSummary(levelCounts?: Record<string, number>) {
    if (!levelCounts) {
        return 'no levels yet';
    }
    const parts = ['ERROR', 'WARN', 'INFO', 'DEBUG']
        .map((level) => [level, levelCounts[level] ?? 0] as const)
        .filter(([, count]) => count > 0)
        .map(([level, count]) => `${count} ${level.toLowerCase()}`);
    return parts.length > 0 ? parts.join(', ') : 'no levels detected';
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
    isInspecting,
    isQuerying,
    metadata,
    onChange,
    onInspect,
    onQuery,
    query,
    result,
}: {
    isInspecting: boolean;
    isQuerying: boolean;
    metadata: ConnectorMetadata | null;
    onChange: (value: string) => void;
    onInspect: () => void;
    onQuery: () => void;
    query: string;
    result: SQLiteQueryResult | null;
}) {
    return (
        <div className="metadata-store-panel sqlite-connector-panel">
            <strong>SQLite Connector</strong>
            <small>Read-only workspace database query surface.</small>
            <div className="metadata-action-row">
                <Button disabled={isInspecting} onClick={onInspect} variant="subtle">
                    {isInspecting ? 'Inspecting...' : 'Inspect schema'}
                </Button>
            </div>
            {metadata && <ConnectorMetadataPanel metadata={metadata} />}
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

function ConnectorMetadataPanel({metadata}: {metadata: ConnectorMetadata}) {
    const objects = [...metadata.tables, ...metadata.views];
    return (
        <div className="connector-metadata-panel">
            <small>{metadata.message}</small>
            <div className="metadata-dataset-views">
                <strong>{metadata.engine}{metadata.readOnly ? ' / read-only' : ''}</strong>
                {objects.slice(0, 8).map((table) => (
                    <p key={`${table.type}-${table.name}`}>
                        {table.type}: {table.name} / {table.rowCount} rows
                        <small>{table.columns.map((column) => `${column.name}:${column.type || 'ANY'}${column.primaryKey ? ' pk' : ''}`).slice(0, 6).join(', ')}</small>
                    </p>
                ))}
                {metadata.indexes.length > 0 && (
                    <small>{metadata.indexes.length} indexes: {metadata.indexes.slice(0, 4).map((index) => `${index.name}(${index.columns.join(', ')})`).join(', ')}</small>
                )}
            </div>
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
