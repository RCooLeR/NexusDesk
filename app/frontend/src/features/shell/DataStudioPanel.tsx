import {useEffect, useMemo, useState} from 'react';
import {Button} from '../../components/ui';
import type {ColumnProfile, DatasetChartResult, DatasetDependency, DatasetProfile, DatasetQueryResult, DatasetSQLQueryResult, SavedDatasetQuery, SQLRun, TablePreview} from '../../types';

type SQLNotebookCell = {
    id: string;
    kind: 'sql' | 'chart';
    label: string;
    sql: string;
};

type SQLResultTab = 'rows' | 'summary' | 'history';

type DataStudioPanelProps = {
    activeDatasetProfile: DatasetProfile | null;
    chartCategory: string;
    chartPreview: DatasetChartResult | null;
    chartType: string;
    chartValue: string;
    columns: string[];
    isCreatingChart: boolean;
    isCreatingSummary: boolean;
    isExporting: boolean;
    isPreviewingChart: boolean;
    isQuerying: boolean;
    isQueryingSQL: boolean;
    isExportingSQL: boolean;
    isSavingSQL: boolean;
    isSavingQuery: boolean;
    onChartCategoryChange: (value: string) => void;
    onChartTypeChange: (value: string) => void;
    onChartValueChange: (value: string) => void;
    onCreateChart: () => void;
    onCreateSummary: () => void;
    onExportQuery: () => void;
    onPreviewChart: () => void;
    onQuery: () => void;
    onQueryChange: (value: string) => void;
    onSQLChange: (value: string) => void;
    onSQLLabelChange: (value: string) => void;
    onSQLQuery: (sql?: string) => void;
    onSQLExport: () => void;
    onSQLSave: () => void;
    onQueryLabelChange: (value: string) => void;
    onSaveQuery: () => void;
    onRebuildDependency: (id: string) => void;
    rebuildingDependencyId: string;
    profiles: ColumnProfile[];
    query: string;
    queryLabel: string;
    queryResult: DatasetQueryResult | null;
    sqlQuery: string;
    sqlLabel: string;
    sqlResult: DatasetSQLQueryResult | null;
    savedQueries: SavedDatasetQuery[];
    savedSQLQueries: SavedDatasetQuery[];
    sqlRuns: SQLRun[];
    dependencies: DatasetDependency[];
    table: TablePreview | null;
};

export function DataStudioPanel({
    activeDatasetProfile,
    chartCategory,
    chartPreview,
    chartType,
    chartValue,
    columns,
    isCreatingChart,
    isCreatingSummary,
    isExporting,
    isPreviewingChart,
    isQuerying,
    isQueryingSQL,
    isExportingSQL,
    isSavingSQL,
    isSavingQuery,
    onChartCategoryChange,
    onChartTypeChange,
    onChartValueChange,
    onCreateChart,
    onCreateSummary,
    onExportQuery,
    onPreviewChart,
    onQuery,
    onQueryChange,
    onSQLChange,
    onSQLLabelChange,
    onSQLQuery,
    onSQLExport,
    onSQLSave,
    onQueryLabelChange,
    onSaveQuery,
    onRebuildDependency,
    rebuildingDependencyId,
    profiles,
    query,
    queryLabel,
    queryResult,
    sqlQuery,
    sqlLabel,
    sqlResult,
    savedQueries,
    savedSQLQueries,
    sqlRuns,
    dependencies,
    table,
}: DataStudioPanelProps) {
    return (
        <>
            {activeDatasetProfile && <DatasetProfileSummary profile={activeDatasetProfile} />}
            <DatasetQueryPanel
                chartCategory={chartCategory}
                chartPreview={chartPreview}
                chartType={chartType}
                chartValue={chartValue}
                columns={columns}
                isCreatingChart={isCreatingChart}
                isExporting={isExporting}
                isPreviewingChart={isPreviewingChart}
                isSaving={isSavingQuery}
                label={queryLabel}
                onChartCategoryChange={onChartCategoryChange}
                onChartTypeChange={onChartTypeChange}
                onChartValueChange={onChartValueChange}
                onCreateChart={onCreateChart}
                onChange={onQueryChange}
                onExport={onExportQuery}
                onLabelChange={onQueryLabelChange}
                onPreviewChart={onPreviewChart}
                onQuery={onQuery}
                onSave={onSaveQuery}
                query={query}
                result={queryResult}
                sqlQuery={sqlQuery}
                sqlResult={sqlResult}
                savedQueries={savedQueries}
                isQuerying={isQuerying}
                isQueryingSQL={isQueryingSQL}
                isExportingSQL={isExportingSQL}
                isSavingSQL={isSavingSQL}
                onSQLChange={onSQLChange}
                onSQLLabelChange={onSQLLabelChange}
                onSQLQuery={onSQLQuery}
                onSQLExport={onSQLExport}
                onSQLSave={onSQLSave}
                profiles={profiles}
                savedSQLQueries={savedSQLQueries}
                onRebuildDependency={onRebuildDependency}
                rebuildingDependencyId={rebuildingDependencyId}
                sqlRuns={sqlRuns}
                dependencies={dependencies}
                sqlLabel={sqlLabel}
            />
            <DatasetChartPanel
                categoryColumn={chartCategory}
                chartType={chartType}
                columns={columns}
                isCreating={isCreatingChart}
                isPreviewing={isPreviewingChart}
                onCategoryChange={onChartCategoryChange}
                onChartTypeChange={onChartTypeChange}
                onCreate={onCreateChart}
                onPreview={onPreviewChart}
                onValueChange={onChartValueChange}
                preview={chartPreview}
                profiles={profiles}
                valueColumn={chartValue}
            />
            {table && <SortableDataTable table={table} title="Table Preview" />}
            <Button disabled={!table || isCreatingSummary} onClick={onCreateSummary} variant="subtle">
                {isCreatingSummary ? 'Summarizing...' : 'Create dataset summary'}
            </Button>
        </>
    );
}

function DatasetQueryPanel({
    chartCategory,
    chartPreview,
    chartType,
    chartValue,
    columns,
    isCreatingChart,
    isExporting,
    isPreviewingChart,
    isSaving,
    label,
    query,
    result,
    sqlQuery,
    sqlResult,
    savedQueries,
    isQuerying,
    isQueryingSQL,
    isExportingSQL,
    isSavingSQL,
    onChange,
    onChartCategoryChange,
    onChartTypeChange,
    onChartValueChange,
    onCreateChart,
    onExport,
    onLabelChange,
    onPreviewChart,
    onQuery,
    onSQLChange,
    onSQLLabelChange,
    onSQLQuery,
    onSQLExport,
    onSQLSave,
    onSave,
    profiles,
    savedSQLQueries,
    sqlRuns,
    dependencies,
    sqlLabel,
    onRebuildDependency,
    rebuildingDependencyId,
}: {
    chartCategory: string;
    chartPreview: DatasetChartResult | null;
    chartType: string;
    chartValue: string;
    columns: string[];
    isCreatingChart: boolean;
    isExporting: boolean;
    isPreviewingChart: boolean;
    isSaving: boolean;
    label: string;
    query: string;
    result: DatasetQueryResult | null;
    sqlQuery: string;
    sqlResult: DatasetSQLQueryResult | null;
    savedQueries: SavedDatasetQuery[];
    isQuerying: boolean;
    isQueryingSQL: boolean;
    isExportingSQL: boolean;
    isSavingSQL: boolean;
    onChange: (value: string) => void;
    onChartCategoryChange: (value: string) => void;
    onChartTypeChange: (value: string) => void;
    onChartValueChange: (value: string) => void;
    onCreateChart: () => void;
    onExport: () => void;
    onLabelChange: (value: string) => void;
    onPreviewChart: () => void;
    onQuery: () => void;
    onSQLChange: (value: string) => void;
    onSQLLabelChange: (value: string) => void;
    onSQLQuery: (sql?: string) => void;
    onSQLExport: () => void;
    onSQLSave: () => void;
    onSave: () => void;
    profiles: ColumnProfile[];
    savedSQLQueries: SavedDatasetQuery[];
    sqlRuns: SQLRun[];
    dependencies: DatasetDependency[];
    sqlLabel: string;
    onRebuildDependency: (id: string) => void;
    rebuildingDependencyId: string;
}) {
    const [filterColumn, setFilterColumn] = useState(columns[0] ?? '');
    const [filterValue, setFilterValue] = useState('');
    const [sqlCells, setSQLCells] = useState<SQLNotebookCell[]>(() => [newSQLNotebookCell(sqlQuery)]);
    const [activeSQLCellId, setActiveSQLCellId] = useState(sqlCells[0]?.id ?? '');
    const [activeSQLResultTab, setActiveSQLResultTab] = useState<SQLResultTab>('rows');
    const activeSQLCell = sqlCells.find((cell) => cell.id === activeSQLCellId) ?? sqlCells[0];

    useEffect(() => {
        setFilterColumn((current) => columns.includes(current) ? current : columns[0] ?? '');
    }, [columns]);

    useEffect(() => {
        setSQLCells((current) => current.map((cell) => cell.id === activeSQLCellId && cell.kind === 'sql' ? {...cell, sql: sqlQuery} : cell));
    }, [activeSQLCellId, sqlQuery]);

    function applyFilter() {
        if (!filterColumn) {
            return;
        }
        onChange(filterValue.trim() ? `${filterColumn}=${filterValue.trim()}` : filterColumn);
    }

    function addSQLCell() {
        const nextCell = newSQLNotebookCell('select * from dataset limit 20', sqlCells.length + 1, 'sql');
        setSQLCells((current) => [...current, nextCell]);
        setActiveSQLCellId(nextCell.id);
        onSQLChange(nextCell.sql);
    }

    function addChartCell() {
        const nextCell = newSQLNotebookCell('', sqlCells.length + 1, 'chart');
        setSQLCells((current) => [...current, nextCell]);
        setActiveSQLCellId(nextCell.id);
    }

    function deleteSQLCell(cellId: string) {
        if (sqlCells.length <= 1) {
            return;
        }
        const nextCells = sqlCells.filter((cell) => cell.id !== cellId);
        setSQLCells(nextCells);
        if (cellId === activeSQLCellId) {
            const nextCell = nextCells[0];
            setActiveSQLCellId(nextCell.id);
            onSQLChange(nextCell.sql);
        }
    }

    function selectSQLCell(cell: SQLNotebookCell) {
        setActiveSQLCellId(cell.id);
        onSQLChange(cell.sql);
    }

    function updateActiveSQL(value: string) {
        setSQLCells((current) => current.map((cell) => cell.id === activeSQLCellId ? {...cell, kind: 'sql', sql: value} : cell));
        onSQLChange(value);
    }

    function applySavedSQL(value: string) {
        updateActiveSQL(value);
    }

    function runActiveSQLCell() {
        if (activeSQLCell?.kind === 'chart') {
            onPreviewChart();
            return;
        }
        onSQLQuery(activeSQLCell?.sql ?? sqlQuery);
    }

    return (
        <div className="dataset-query-panel">
            <strong>Dataset Query</strong>
            <div className="dataset-filter-row">
                <select aria-label="Filter column" onChange={(event) => setFilterColumn(event.target.value)} value={filterColumn}>
                    {columns.map((column) => (
                        <option key={column} value={column}>{column}</option>
                    ))}
                </select>
                <input
                    aria-label="Filter value"
                    onChange={(event) => setFilterValue(event.target.value)}
                    onKeyDown={(event) => {
                        if (event.key === 'Enter') {
                            applyFilter();
                        }
                    }}
                    placeholder="Filter value"
                    value={filterValue}
                />
                <Button disabled={!filterColumn} onClick={applyFilter} variant="subtle">Apply</Button>
            </div>
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
                <Button disabled={!result || isExporting} onClick={onExport} variant="subtle">
                    {isExporting ? 'Exporting...' : 'Export'}
                </Button>
            </div>
            {savedQueries.length > 0 && (
                <div className="saved-query-list" aria-label="Saved dataset queries">
                    {savedQueries.map((saved) => (
                        <button key={`${saved.relPath}-${saved.query}`} onClick={() => onChange(saved.query)} title={saved.query || 'First rows'}>
                            {saved.label}
                        </button>
                    ))}
                </div>
            )}
            <div className="query-save-row">
                <input
                    aria-label="Saved query label"
                    onChange={(event) => onLabelChange(event.target.value)}
                    placeholder="Label"
                    value={label}
                />
                <Button disabled={isSaving} onClick={onSave} variant="subtle">
                    {isSaving ? 'Saving...' : 'Save query'}
                </Button>
            </div>
            {result && (
                <div className="dataset-query-result">
                    <small>{result.message}</small>
                    <SortableDataTable
                        pageSize={12}
                        table={{
                            columns: result.columns,
                            rows: result.rows,
                            profiles: [],
                            totalRows: result.matchedRows,
                            truncated: result.rows.length < result.matchedRows,
                        }}
                        title="Query Result"
                    />
                </div>
            )}
            <div className="dataset-sql-panel">
                <strong>Read-only SQL</strong>
                {savedSQLQueries.length > 0 && (
                    <div className="saved-query-list" aria-label="Saved SQL snippets">
                        {savedSQLQueries.map((saved) => (
                            <button key={`${saved.relPath}-${saved.query}`} onClick={() => applySavedSQL(saved.query)} title={saved.query}>
                                {saved.label}
                            </button>
                        ))}
                    </div>
                )}
                <div className="sql-notebook" aria-label="SQL notebook cells">
                    <div className="sql-notebook-tabs">
                        {sqlCells.map((cell, index) => (
                            <button
                                aria-pressed={cell.id === activeSQLCellId}
                                className={cell.id === activeSQLCellId ? 'active' : ''}
                                key={cell.id}
                                onClick={() => selectSQLCell(cell)}
                                type="button"
                            >
                                <span>{cell.label || `Cell ${index + 1}`}</span>
                                <small>{cell.kind === 'chart' ? 'chart' : cell.sql.trim() ? 'SQL' : 'empty'}</small>
                            </button>
                        ))}
                        <Button onClick={addSQLCell} variant="subtle">Add cell</Button>
                        <Button onClick={addChartCell} variant="subtle">Add chart</Button>
                    </div>
                    <div className="sql-notebook-toolbar">
                        <input
                            aria-label="SQL cell label"
                            onChange={(event) => {
                                const value = event.target.value;
                                setSQLCells((current) => current.map((cell) => cell.id === activeSQLCellId ? {...cell, label: value} : cell));
                            }}
                            placeholder="Cell label"
                            value={activeSQLCell?.label ?? ''}
                        />
                        <Button disabled={sqlCells.length <= 1} onClick={() => deleteSQLCell(activeSQLCellId)} variant="subtle">Delete cell</Button>
                        <Button
                            disabled={activeSQLCell?.kind === 'chart' ? isPreviewingChart || columns.length === 0 : isQueryingSQL || !(activeSQLCell?.sql ?? '').trim()}
                            onClick={runActiveSQLCell}
                            variant="subtle"
                        >
                            {activeSQLCell?.kind === 'chart' ? isPreviewingChart ? 'Previewing...' : 'Preview chart' : isQueryingSQL ? 'Running...' : 'Run cell'}
                        </Button>
                    </div>
                </div>
                {activeSQLCell?.kind === 'chart' ? (
                    <div className="sql-chart-cell">
                        <DatasetChartPanel
                            categoryColumn={chartCategory}
                            chartType={chartType}
                            columns={columns}
                            isCreating={isCreatingChart}
                            isPreviewing={isPreviewingChart}
                            onCategoryChange={onChartCategoryChange}
                            onChartTypeChange={onChartTypeChange}
                            onCreate={onCreateChart}
                            onPreview={onPreviewChart}
                            onValueChange={onChartValueChange}
                            preview={chartPreview}
                            profiles={profiles}
                            valueColumn={chartValue}
                        />
                    </div>
                ) : (
                    <textarea
                        aria-label="DuckDB-compatible SQL query"
                        onChange={(event) => updateActiveSQL(event.target.value)}
                        placeholder="select * from dataset where spend > 10 order by spend desc limit 20"
                        value={sqlQuery}
                    />
                )}
                {activeSQLCell?.kind !== 'chart' && (
                    <>
                        <div className="query-save-row">
                            <input
                                aria-label="Saved SQL label"
                                onChange={(event) => onSQLLabelChange(event.target.value)}
                                placeholder="SQL label"
                                value={sqlLabel}
                            />
                            <Button disabled={isSavingSQL || !sqlQuery.trim()} onClick={onSQLSave} variant="subtle">
                                {isSavingSQL ? 'Saving...' : 'Save SQL'}
                            </Button>
                        </div>
                        <div className="dataset-query-row">
                            <Button disabled={isQueryingSQL} onClick={() => onSQLQuery()} variant="subtle">
                                {isQueryingSQL ? 'Running...' : 'Run SQL'}
                            </Button>
                            <Button disabled={isExportingSQL || !sqlQuery.trim()} onClick={onSQLExport} variant="subtle">
                                {isExportingSQL ? 'Exporting...' : 'Export SQL'}
                            </Button>
                        </div>
                    </>
                )}
                {(sqlResult || sqlRuns.length > 0 || dependencies.length > 0) && (
                    <SQLResultTabs
                        activeTab={activeSQLResultTab}
                        dependencies={dependencies}
                        onRebuildDependency={onRebuildDependency}
                        onTabChange={setActiveSQLResultTab}
                        rebuildingDependencyId={rebuildingDependencyId}
                        result={sqlResult}
                        runs={sqlRuns}
                    />
                )}
            </div>
        </div>
    );
}

function canRebuildDependency(kind: string) {
    return ['filter-export', 'sql-report', 'chart', 'summary'].includes(kind);
}

function newSQLNotebookCell(sql = '', index = 1, kind: SQLNotebookCell['kind'] = 'sql'): SQLNotebookCell {
    return {
        id: `sql-cell-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
        kind,
        label: kind === 'chart' ? `Chart ${index}` : `Cell ${index}`,
        sql,
    };
}

function SQLResultTabs({
    activeTab,
    dependencies,
    onRebuildDependency,
    onTabChange,
    rebuildingDependencyId,
    result,
    runs,
}: {
    activeTab: SQLResultTab;
    dependencies: DatasetDependency[];
    onRebuildDependency: (id: string) => void;
    onTabChange: (tab: SQLResultTab) => void;
    rebuildingDependencyId: string;
    result: DatasetSQLQueryResult | null;
    runs: SQLRun[];
}) {
    const tabs: Array<{id: SQLResultTab; label: string; disabled?: boolean}> = [
        {id: 'rows', label: 'Rows', disabled: !result},
        {id: 'summary', label: 'Summary', disabled: !result},
        {id: 'history', label: 'History', disabled: runs.length === 0 && dependencies.length === 0},
    ];
    const currentTab = tabs.some((tab) => tab.id === activeTab && !tab.disabled)
        ? activeTab
        : tabs.find((tab) => !tab.disabled)?.id ?? 'rows';

    return (
        <div className="sql-result-tabs">
            <div className="sql-result-tab-list" role="tablist" aria-label="SQL result tabs">
                {tabs.map((tab) => (
                    <button
                        aria-selected={currentTab === tab.id}
                        className={currentTab === tab.id ? 'active' : ''}
                        disabled={tab.disabled}
                        key={tab.id}
                        onClick={() => onTabChange(tab.id)}
                        role="tab"
                        type="button"
                    >
                        {tab.label}
                    </button>
                ))}
            </div>
            {currentTab === 'rows' && result && (
                <div className="dataset-query-result" role="tabpanel">
                    <small>{result.message}</small>
                    <SortableDataTable
                        pageSize={12}
                        table={{
                            columns: result.columns,
                            rows: result.rows,
                            profiles: [],
                            totalRows: result.matchedRows,
                            truncated: result.rows.length < result.matchedRows,
                        }}
                        title="SQL Result"
                    />
                </div>
            )}
            {currentTab === 'summary' && result && (
                <div className="sql-result-summary" role="tabpanel">
                    <p><strong>{result.engine}</strong><small>{result.relPath}</small></p>
                    <p><strong>{result.matchedRows}</strong><small>matched rows</small></p>
                    <p><strong>{result.rows.length}</strong><small>preview rows</small></p>
                    <pre>{result.sql}</pre>
                    <small>{result.message}</small>
                </div>
            )}
            {currentTab === 'history' && (
                <div className="dataset-lineage-history" role="tabpanel">
                    {runs.slice(0, 5).map((run) => (
                        <p key={run.id}><strong>{run.status}</strong> {run.engine} / {run.rows} rows <small>{run.artifact || run.message}</small></p>
                    ))}
                    {dependencies.slice(0, 6).map((item) => (
                        <p className="dataset-lineage-row" key={item.id}>
                            <span><strong>{item.kind}</strong> {item.target || item.artifact || item.query}</span>
                            {canRebuildDependency(item.kind) ? (
                                <Button
                                    className="dataset-lineage-rebuild"
                                    disabled={rebuildingDependencyId === item.id}
                                    onClick={() => onRebuildDependency(item.id)}
                                    variant="subtle"
                                >
                                    {rebuildingDependencyId === item.id ? 'Rebuilding...' : 'Rebuild'}
                                </Button>
                            ) : null}
                        </p>
                    ))}
                    {runs.length === 0 && dependencies.length === 0 && <small>No SQL history or lineage yet.</small>}
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
    isPreviewing,
    onCategoryChange,
    onChartTypeChange,
    onCreate,
    onPreview,
    onValueChange,
    preview,
    profiles,
    valueColumn,
}: {
    categoryColumn: string;
    chartType: string;
    columns: string[];
    isCreating: boolean;
    isPreviewing: boolean;
    onCategoryChange: (value: string) => void;
    onChartTypeChange: (value: string) => void;
    onCreate: () => void;
    onPreview: () => void;
    onValueChange: (value: string) => void;
    preview: DatasetChartResult | null;
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
                        <option value="line">Line</option>
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
                <Button disabled={isPreviewing || columns.length === 0 || !categoryColumn} onClick={onPreview} variant="subtle">
                    {isPreviewing ? 'Previewing...' : 'Preview'}
                </Button>
            </div>
            {preview && <DatasetChartPreview preview={preview} />}
        </div>
    );
}

function DatasetChartPreview({preview}: {preview: DatasetChartResult}) {
    const maxValue = Math.max(...preview.points.map((point) => point.value), 1);

    return (
        <div className="dataset-chart-preview" aria-label="Dataset chart preview">
            <small>{preview.message}</small>
            <dl className="chart-config-list">
                <div><dt>Type</dt><dd>{preview.chartType}</dd></div>
                <div><dt>Category</dt><dd>{preview.categoryColumn}</dd></div>
                <div><dt>Value</dt><dd>{preview.valueColumn || 'row count'}</dd></div>
            </dl>
            {preview.points.map((point) => (
                <div className="chart-preview-row" key={point.label}>
                    <span>{point.label}</span>
                    <i style={{width: `${Math.max(4, (point.value / maxValue) * 100)}%`}} />
                    <strong>{formatChartPoint(point.value)}</strong>
                </div>
            ))}
        </div>
    );
}

function DatasetProfileSummary({profile}: {profile: DatasetProfile}) {
    return (
        <div className="dataset-profile-summary">
            <strong>{profile.name}</strong>
            <small>{profile.kind}</small>
            {profile.kind === 'xlsx' ? (
                <>
                    <p>{profile.workbook?.sheets?.length ?? profile.sheets.length} sheets, {profile.workbook?.formulaCount ?? 0} formulas</p>
                    {profile.workbook?.tableRanges?.length > 0 && (
                        <small>{profile.workbook.tableRanges.length} tables: {profile.workbook.tableRanges.map((table) => `${table.sheet}:${table.ref}`).join(', ')}</small>
                    )}
                    {profile.workbook?.namedRanges?.length > 0 && (
                        <small>{profile.workbook.namedRanges.length} named ranges</small>
                    )}
                    {profile.workbook?.pivotTables?.length > 0 && (
                        <small>{profile.workbook.pivotTables.length} pivots</small>
                    )}
                </>
            ) : profile.kind === 'parquet' ? (
                <>
                    <p>{formatBytes(profile.parquet?.fileSize ?? 0)} file, {formatBytes(profile.parquet?.footerMetadataBytes ?? 0)} footer metadata</p>
                    <small>{profile.parquet?.message || profile.message}</small>
                </>
            ) : profile.kind === 'log' ? (
                <>
                    <p>{profile.log?.sampledLines ?? 0} sampled lines{profile.log?.truncated ? ', bounded sample' : ''}</p>
                    <small>{logLevelSummary(profile.log?.levelCounts)}</small>
                    {(profile.log?.timestampedLines ?? 0) > 0 && <small>{profile.log.timestampedLines} timestamped lines</small>}
                    {(profile.log?.stackTraceLines ?? 0) > 0 && <small>{profile.log.stackTraceLines} stack trace lines</small>}
                    {(profile.log?.topPatterns?.length ?? 0) > 0 && (
                        <small>{profile.log.topPatterns.slice(0, 2).map((pattern) => `${pattern.count}x ${pattern.pattern}`).join(' | ')}</small>
                    )}
                </>
            ) : (
                <p>{profile.rows} rows, {profile.columns} columns</p>
            )}
        </div>
    );
}

export function SortableDataTable({pageSize = 20, table, title}: {pageSize?: number; table: TablePreview; title?: string}) {
    const [sort, setSort] = useState<{column: number; direction: 'asc' | 'desc'} | null>(null);
    const [page, setPage] = useState(0);

    useEffect(() => {
        setPage(0);
    }, [table]);

    const rows = useMemo(() => {
        const nextRows = [...table.rows];
        if (!sort) {
            return nextRows;
        }
        nextRows.sort((left, right) => compareCellValues(left[sort.column] ?? '', right[sort.column] ?? '', sort.direction));
        return nextRows;
    }, [sort, table.rows]);

    const pageCount = Math.max(1, Math.ceil(rows.length / pageSize));
    const safePage = Math.min(page, pageCount - 1);
    const visibleRows = rows.slice(safePage * pageSize, safePage * pageSize + pageSize);

    function toggleSort(columnIndex: number) {
        setSort((current) => {
            if (!current || current.column !== columnIndex) {
                return {column: columnIndex, direction: 'asc'};
            }
            if (current.direction === 'asc') {
                return {column: columnIndex, direction: 'desc'};
            }
            return null;
        });
    }

    return (
        <div className="sortable-data-table">
            <div className="table-toolbar">
                <strong>{title ?? 'Table'}</strong>
                <small>{table.totalRows} rows{table.truncated ? ', bounded preview' : ''}</small>
            </div>
            {table.profiles.length > 0 && (
                <div className="csv-profile-strip" aria-label="Column profile">
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
            <div className="csv-preview" aria-label={`${title ?? 'Dataset'} table preview`}>
                <table>
                    <thead>
                        <tr>
                            {table.columns.map((column, index) => (
                                <th key={`${column}-${index}`}>
                                    <button onClick={() => toggleSort(index)}>
                                        {column || `Column ${index + 1}`}
                                        {sort?.column === index ? (sort.direction === 'asc' ? ' ↑' : ' ↓') : ''}
                                    </button>
                                </th>
                            ))}
                        </tr>
                    </thead>
                    <tbody>
                        {visibleRows.map((row, rowIndex) => (
                            <tr key={`${safePage}-${rowIndex}`}>
                                {table.columns.map((_, columnIndex) => (
                                    <td key={columnIndex}>{row[columnIndex] ?? ''}</td>
                                ))}
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
            {pageCount > 1 && (
                <div className="table-pager">
                    <Button disabled={safePage === 0} onClick={() => setPage((current) => Math.max(0, current - 1))} variant="subtle">Prev</Button>
                    <small>Page {safePage + 1} of {pageCount}</small>
                    <Button disabled={safePage >= pageCount - 1} onClick={() => setPage((current) => Math.min(pageCount - 1, current + 1))} variant="subtle">Next</Button>
                </div>
            )}
        </div>
    );
}

function compareCellValues(left: string, right: string, direction: 'asc' | 'desc') {
    const leftNumber = Number(left);
    const rightNumber = Number(right);
    const bothNumeric = Number.isFinite(leftNumber) && Number.isFinite(rightNumber) && left.trim() !== '' && right.trim() !== '';
    const result = bothNumeric
        ? leftNumber - rightNumber
        : left.localeCompare(right, undefined, {numeric: true, sensitivity: 'base'});
    return direction === 'asc' ? result : -result;
}

function formatChartPoint(value: number) {
    if (!Number.isFinite(value)) {
        return '0';
    }
    return Number.isInteger(value) ? value.toString() : value.toFixed(2);
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

function logLevelSummary(levelCounts?: Record<string, number>) {
    if (!levelCounts) {
        return 'No levels detected';
    }
    const parts = ['FATAL', 'ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE']
        .map((level) => [level, levelCounts[level] ?? 0] as const)
        .filter(([, count]) => count > 0)
        .map(([level, count]) => `${count} ${level.toLowerCase()}`);
    return parts.length > 0 ? parts.join(', ') : 'No levels detected';
}
