import {useEffect, useMemo, useState} from 'react';
import {Button} from '../../components/ui';
import type {ColumnProfile, DatasetChartResult, DatasetDependency, DatasetProfile, DatasetQueryResult, DatasetSQLQueryResult, SavedDatasetQuery, SQLRun, TablePreview} from '../../types';

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
    onSQLQuery: () => void;
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
                columns={columns}
                isExporting={isExporting}
                isSaving={isSavingQuery}
                label={queryLabel}
                onChange={onQueryChange}
                onExport={onExportQuery}
                onLabelChange={onQueryLabelChange}
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
    columns,
    isExporting,
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
    onExport,
    onLabelChange,
    onQuery,
    onSQLChange,
    onSQLLabelChange,
    onSQLQuery,
    onSQLExport,
    onSQLSave,
    onSave,
    savedSQLQueries,
    sqlRuns,
    dependencies,
    sqlLabel,
    onRebuildDependency,
    rebuildingDependencyId,
}: {
    columns: string[];
    isExporting: boolean;
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
    onExport: () => void;
    onLabelChange: (value: string) => void;
    onQuery: () => void;
    onSQLChange: (value: string) => void;
    onSQLLabelChange: (value: string) => void;
    onSQLQuery: () => void;
    onSQLExport: () => void;
    onSQLSave: () => void;
    onSave: () => void;
    savedSQLQueries: SavedDatasetQuery[];
    sqlRuns: SQLRun[];
    dependencies: DatasetDependency[];
    sqlLabel: string;
    onRebuildDependency: (id: string) => void;
    rebuildingDependencyId: string;
}) {
    const [filterColumn, setFilterColumn] = useState(columns[0] ?? '');
    const [filterValue, setFilterValue] = useState('');

    useEffect(() => {
        setFilterColumn((current) => columns.includes(current) ? current : columns[0] ?? '');
    }, [columns]);

    function applyFilter() {
        if (!filterColumn) {
            return;
        }
        onChange(filterValue.trim() ? `${filterColumn}=${filterValue.trim()}` : filterColumn);
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
                            <button key={`${saved.relPath}-${saved.query}`} onClick={() => onSQLChange(saved.query)} title={saved.query}>
                                {saved.label}
                            </button>
                        ))}
                    </div>
                )}
                <textarea
                    aria-label="DuckDB-compatible SQL query"
                    onChange={(event) => onSQLChange(event.target.value)}
                    placeholder="select * from dataset where spend > 10 order by spend desc limit 20"
                    value={sqlQuery}
                />
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
                    <Button disabled={isQueryingSQL} onClick={onSQLQuery} variant="subtle">
                        {isQueryingSQL ? 'Running...' : 'Run SQL'}
                    </Button>
                    <Button disabled={isExportingSQL || !sqlQuery.trim()} onClick={onSQLExport} variant="subtle">
                        {isExportingSQL ? 'Exporting...' : 'Export SQL'}
                    </Button>
                </div>
                {sqlResult && (
                    <div className="dataset-query-result">
                        <small>{sqlResult.message}</small>
                        <SortableDataTable
                            pageSize={12}
                            table={{
                                columns: sqlResult.columns,
                                rows: sqlResult.rows,
                                profiles: [],
                                totalRows: sqlResult.matchedRows,
                                truncated: sqlResult.rows.length < sqlResult.matchedRows,
                            }}
                            title="SQL Result"
                        />
                    </div>
                )}
                {(sqlRuns.length > 0 || dependencies.length > 0) && (
                    <div className="dataset-lineage-history">
                        {sqlRuns.slice(0, 3).map((run) => (
                            <p key={run.id}><strong>{run.status}</strong> {run.engine} / {run.rows} rows <small>{run.artifact || run.message}</small></p>
                        ))}
                        {dependencies.slice(0, 4).map((item) => (
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
                    </div>
                )}
            </div>
        </div>
    );
}

function canRebuildDependency(kind: string) {
    return ['filter-export', 'sql-report', 'chart', 'summary'].includes(kind);
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
