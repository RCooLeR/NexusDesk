import type {AgentToolDescriptor, AgentToolPlanItem, AgentToolRunRecord, ApprovalRecord, ArtifactComparison, ArtifactLineage, ArtifactMetadata, Capability, DatasetChartResult, DatasetDependency, DatasetProfile, DatasetQueryResult, DatasetSQLQueryResult, FilePreview, LLMProbeResult, LLMSettings, MetadataBrowser, MetadataSearchResult, SavedDatasetQuery, SQLRun, SQLiteMetadataStatus, SQLiteQueryResult, ToolEvent, WorkspaceArtifact, WorkspaceFreshnessStatus, WorkspaceSnapshot} from '../../types';
import {AgentToolPlanCard} from './AgentToolPlanCard';
import {ApprovalLogPanel} from './ApprovalLogPanel';
import {ArtifactStudioPanel} from './ArtifactStudioPanel';
import {DataOperationsPanel} from './DataOperationsPanel';
import {LLMSettingsCard} from './LLMSettingsCard';
import {ToolTimeline} from './ToolTimeline';

type BottomStudioTab = 'settings' | 'data' | 'tools' | 'artifacts' | 'approvals' | 'activity';

type BottomStudioPanelProps = {
    activeTab: BottomStudioTab;
    agentTools: AgentToolDescriptor[];
    agentToolPlan: AgentToolPlanItem[];
    agentToolRuns: AgentToolRunRecord[];
    approvalRecords: ApprovalRecord[];
    artifacts: WorkspaceArtifact[];
    artifactComparison: ArtifactComparison | null;
    artifactLineage: ArtifactLineage | null;
    artifactMetadata: ArtifactMetadata | null;
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
    isArchivingArtifact: boolean;
    isCreatingDatasetChart: boolean;
    isCreatingDatasetSummary: boolean;
    isDeletingArtifact: boolean;
    isExportingDatasetQuery: boolean;
    isExportingDatasetSQL: boolean;
    isPreparingMetadataStore: boolean;
    isProfilingDataset: boolean;
    isPreviewingDatasetChart: boolean;
    isQueryingDataset: boolean;
    isQueryingDatasetSQL: boolean;
    isQueryingSQLiteConnector: boolean;
    isRefreshingStaleContext: boolean;
    isRunningAgentTool: boolean;
    isSavingDatasetQuery: boolean;
    isSavingDatasetSQLQuery: boolean;
    isSavingSettings: boolean;
    isSearchingMetadata: boolean;
    isTestingConnection: boolean;
    metadataBrowser: MetadataBrowser | null;
    metadataSearchQuery: string;
    metadataSearchResults: MetadataSearchResult[];
    onArchiveArtifact: () => void;
    onCompareAgentToolRunTarget: (run: AgentToolRunRecord) => void;
    onCompareArtifact: () => void;
    onCreateDatasetChart: () => void;
    onCreateDatasetSummary: () => void;
    onDatasetChartCategoryChange: (content: string) => void;
    onDatasetChartTypeChange: (content: string) => void;
    onDatasetChartValueChange: (content: string) => void;
    onDatasetQueryChange: (content: string) => void;
    onDatasetQueryLabelChange: (content: string) => void;
    onDatasetSQLQueryChange: (content: string) => void;
    onDatasetSQLQueryLabelChange: (content: string) => void;
    onDeleteArtifact: () => void;
    onDryRunAgentTool: (item: AgentToolPlanItem) => void;
    onExecuteAgentTool: (item: AgentToolPlanItem) => void;
    onExportDatasetQuery: () => void;
    onExportDatasetSQL: () => void;
    onExportLineage: () => void;
    onInspectMetadata: () => void;
    onMetadataSearchQueryChange: (content: string) => void;
    onOpenArtifactSource: () => void;
    onOpenLineageSource: (relPath: string) => void;
    onPrepareMetadataStore: () => void;
    onProfileDataset: () => void;
    onPreviewDatasetChart: () => void;
    onQueryDataset: () => void;
    onQueryDatasetSQL: () => void;
    onQuerySQLiteConnector: () => void;
    onRebuildDatasetDependency: (dependencyId: string) => void;
    onRefreshAgentPlan: () => void;
    onRefreshLineage: () => void;
    onRefreshStaleContext: () => void;
    onReplayAgentToolRun: (run: AgentToolRunRecord) => void;
    onSaveDatasetQuery: () => void;
    onSaveDatasetSQLQuery: () => void;
    onSaveSettings: () => void;
    onSelectArtifact: (artifact: WorkspaceArtifact) => void;
    onSettingsDraftChange: (field: keyof LLMSettings, value: string) => void;
    onSearchMetadata: () => void;
    onSQLiteConnectorQueryChange: (content: string) => void;
    onTabChange: (tab: BottomStudioTab) => void;
    onTestConnection: () => void;
    probeResult: LLMProbeResult | null;
    rebuildingDatasetDependencyId: string;
    savedDatasetQueries: SavedDatasetQuery[];
    savedDatasetSQLQueries: SavedDatasetQuery[];
    settingsDraft: LLMSettings;
    settingsStatus: string;
    sqliteConnectorQuery: string;
    sqliteConnectorResult: SQLiteQueryResult | null;
    sqliteStatus: SQLiteMetadataStatus | null;
    toolEvents: ToolEvent[];
    workspace: WorkspaceSnapshot | null;
    workspaceFreshness: WorkspaceFreshnessStatus | null;
};

const tabs: Array<{id: BottomStudioTab; label: string}> = [
    {id: 'settings', label: 'Settings'},
    {id: 'data', label: 'Data'},
    {id: 'tools', label: 'Tools'},
    {id: 'artifacts', label: 'Artifacts'},
    {id: 'approvals', label: 'Approvals'},
    {id: 'activity', label: 'Activity'},
];

export function BottomStudioPanel({
    activeTab,
    agentTools,
    agentToolPlan,
    agentToolRuns,
    approvalRecords,
    artifacts,
    artifactComparison,
    artifactLineage,
    artifactMetadata,
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
    isArchivingArtifact,
    isCreatingDatasetChart,
    isCreatingDatasetSummary,
    isDeletingArtifact,
    isExportingDatasetQuery,
    isExportingDatasetSQL,
    isPreparingMetadataStore,
    isProfilingDataset,
    isPreviewingDatasetChart,
    isQueryingDataset,
    isQueryingDatasetSQL,
    isQueryingSQLiteConnector,
    isRefreshingStaleContext,
    isRunningAgentTool,
    isSavingDatasetQuery,
    isSavingDatasetSQLQuery,
    isSavingSettings,
    isSearchingMetadata,
    isTestingConnection,
    metadataBrowser,
    metadataSearchQuery,
    metadataSearchResults,
    onArchiveArtifact,
    onCompareAgentToolRunTarget,
    onCompareArtifact,
    onCreateDatasetChart,
    onCreateDatasetSummary,
    onDatasetChartCategoryChange,
    onDatasetChartTypeChange,
    onDatasetChartValueChange,
    onDatasetQueryChange,
    onDatasetQueryLabelChange,
    onDatasetSQLQueryChange,
    onDatasetSQLQueryLabelChange,
    onDeleteArtifact,
    onDryRunAgentTool,
    onExecuteAgentTool,
    onExportDatasetQuery,
    onExportDatasetSQL,
    onExportLineage,
    onInspectMetadata,
    onMetadataSearchQueryChange,
    onOpenArtifactSource,
    onOpenLineageSource,
    onPrepareMetadataStore,
    onProfileDataset,
    onPreviewDatasetChart,
    onQueryDataset,
    onQueryDatasetSQL,
    onQuerySQLiteConnector,
    onRebuildDatasetDependency,
    onRefreshAgentPlan,
    onRefreshLineage,
    onRefreshStaleContext,
    onReplayAgentToolRun,
    onSaveDatasetQuery,
    onSaveDatasetSQLQuery,
    onSaveSettings,
    onSelectArtifact,
    onSettingsDraftChange,
    onSearchMetadata,
    onSQLiteConnectorQueryChange,
    onTabChange,
    onTestConnection,
    probeResult,
    rebuildingDatasetDependencyId,
    savedDatasetQueries,
    savedDatasetSQLQueries,
    settingsDraft,
    settingsStatus,
    sqliteConnectorQuery,
    sqliteConnectorResult,
    sqliteStatus,
    toolEvents,
    workspace,
    workspaceFreshness,
}: BottomStudioPanelProps) {
    return (
        <section className="bottom-studio-panel" aria-label="Studio tools and settings">
            <div className="bottom-tabbar" role="tablist" aria-label="Studio drawer tabs">
                {tabs.map((tab) => (
                    <button
                        aria-selected={activeTab === tab.id}
                        className={activeTab === tab.id ? 'active' : ''}
                        key={tab.id}
                        onClick={() => onTabChange(tab.id)}
                        role="tab"
                    >
                        {tab.label}
                    </button>
                ))}
            </div>
            <div className="bottom-tab-content">
                {activeTab === 'settings' && (
                    <div className="settings-page">
                        <div className="settings-page-heading">
                            <strong>Settings</strong>
                            <small>Provider, model, runtime, and local assistant configuration.</small>
                        </div>
                        <LLMSettingsCard
                            isSavingSettings={isSavingSettings}
                            isTestingConnection={isTestingConnection}
                            onSaveSettings={onSaveSettings}
                            onSettingsDraftChange={onSettingsDraftChange}
                            onTestConnection={onTestConnection}
                            probeResult={probeResult}
                            settingsDraft={settingsDraft}
                            settingsStatus={settingsStatus}
                        />
                    </div>
                )}
                {activeTab === 'tools' && (
                    <AgentToolPlanCard
                        tools={agentTools}
                        planItems={agentToolPlan}
                        runs={agentToolRuns}
                        isRunning={isRunningAgentTool}
                        onDryRun={onDryRunAgentTool}
                        onExecute={onExecuteAgentTool}
                        onReplayRun={onReplayAgentToolRun}
                        onCompareRunTarget={onCompareAgentToolRunTarget}
                        onRefreshPlan={onRefreshAgentPlan}
                    />
                )}
                {activeTab === 'data' && (
                    <DataOperationsPanel
                        activeDatasetProfile={activeDatasetProfile}
                        capabilities={capabilities}
                        datasetProfiles={datasetProfiles}
                        datasetDependencies={datasetDependencies}
                        datasetSQLRuns={datasetSQLRuns}
                        datasetChartCategory={datasetChartCategory}
                        datasetChartPreview={datasetChartPreview}
                        datasetChartType={datasetChartType}
                        datasetChartValue={datasetChartValue}
                        datasetQuery={datasetQuery}
                        datasetQueryLabel={datasetQueryLabel}
                        datasetQueryResult={datasetQueryResult}
                        datasetSQLQuery={datasetSQLQuery}
                        datasetSQLQueryLabel={datasetSQLQueryLabel}
                        datasetSQLQueryResult={datasetSQLQueryResult}
                        filePreview={filePreview}
                        isCreatingDatasetChart={isCreatingDatasetChart}
                        isCreatingDatasetSummary={isCreatingDatasetSummary}
                        isExportingDatasetQuery={isExportingDatasetQuery}
                        isExportingDatasetSQL={isExportingDatasetSQL}
                        isPreparingMetadataStore={isPreparingMetadataStore}
                        isProfilingDataset={isProfilingDataset}
                        isPreviewingDatasetChart={isPreviewingDatasetChart}
                        isQueryingDataset={isQueryingDataset}
                        isQueryingDatasetSQL={isQueryingDatasetSQL}
                        isQueryingSQLiteConnector={isQueryingSQLiteConnector}
                        isRefreshingStaleContext={isRefreshingStaleContext}
                        isSavingDatasetQuery={isSavingDatasetQuery}
                        isSavingDatasetSQLQuery={isSavingDatasetSQLQuery}
                        isSearchingMetadata={isSearchingMetadata}
                        metadataBrowser={metadataBrowser}
                        metadataSearchQuery={metadataSearchQuery}
                        metadataSearchResults={metadataSearchResults}
                        onCreateDatasetChart={onCreateDatasetChart}
                        onCreateDatasetSummary={onCreateDatasetSummary}
                        onDatasetChartCategoryChange={onDatasetChartCategoryChange}
                        onDatasetChartTypeChange={onDatasetChartTypeChange}
                        onDatasetChartValueChange={onDatasetChartValueChange}
                        onDatasetQueryChange={onDatasetQueryChange}
                        onDatasetQueryLabelChange={onDatasetQueryLabelChange}
                        onDatasetSQLQueryChange={onDatasetSQLQueryChange}
                        onDatasetSQLQueryLabelChange={onDatasetSQLQueryLabelChange}
                        onExportDatasetQuery={onExportDatasetQuery}
                        onExportDatasetSQL={onExportDatasetSQL}
                        onInspectMetadata={onInspectMetadata}
                        onMetadataSearchQueryChange={onMetadataSearchQueryChange}
                        onPrepareMetadataStore={onPrepareMetadataStore}
                        onProfileDataset={onProfileDataset}
                        onPreviewDatasetChart={onPreviewDatasetChart}
                        onQueryDataset={onQueryDataset}
                        onQueryDatasetSQL={onQueryDatasetSQL}
                        onQuerySQLiteConnector={onQuerySQLiteConnector}
                        onRebuildDatasetDependency={onRebuildDatasetDependency}
                        onRefreshStaleContext={onRefreshStaleContext}
                        onSaveDatasetQuery={onSaveDatasetQuery}
                        onSaveDatasetSQLQuery={onSaveDatasetSQLQuery}
                        onSearchMetadata={onSearchMetadata}
                        onSQLiteConnectorQueryChange={onSQLiteConnectorQueryChange}
                        rebuildingDatasetDependencyId={rebuildingDatasetDependencyId}
                        savedDatasetQueries={savedDatasetQueries}
                        savedDatasetSQLQueries={savedDatasetSQLQueries}
                        sqliteConnectorQuery={sqliteConnectorQuery}
                        sqliteConnectorResult={sqliteConnectorResult}
                        sqliteStatus={sqliteStatus}
                        workspace={workspace}
                        workspaceFreshness={workspaceFreshness}
                    />
                )}
                {activeTab === 'artifacts' && (
                    <ArtifactStudioPanel
                        artifacts={artifacts}
                        artifactComparison={artifactComparison}
                        artifactLineage={artifactLineage}
                        artifactMetadata={artifactMetadata}
                        filePreview={filePreview}
                        isArchivingArtifact={isArchivingArtifact}
                        isDeletingArtifact={isDeletingArtifact}
                        onArchiveArtifact={onArchiveArtifact}
                        onCompareArtifact={onCompareArtifact}
                        onDeleteArtifact={onDeleteArtifact}
                        onExportLineage={onExportLineage}
                        onOpenArtifactSource={onOpenArtifactSource}
                        onOpenLineageSource={onOpenLineageSource}
                        onRefreshLineage={onRefreshLineage}
                        onSelectArtifact={onSelectArtifact}
                    />
                )}
                {activeTab === 'approvals' && <ApprovalLogPanel records={approvalRecords} />}
                {activeTab === 'activity' && <ToolTimeline events={toolEvents} />}
            </div>
        </section>
    );
}

export type {BottomStudioTab};
