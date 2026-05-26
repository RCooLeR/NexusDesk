import type {AgentToolDescriptor, AgentToolPlanItem, AgentToolRunRecord, ApprovalRecord, ArtifactComparison, ArtifactLineage, ArtifactMetadata, Capability, ConnectorMetadata, ConnectorProfile, DatasetChartResult, DatasetDependency, DatasetProfile, DatasetQueryResult, DatasetSQLQueryResult, FilePreview, GitFileAction, GitFileActionPreview, GitFileDiff, GitHunkActionPreview, GitHunkActionRequest, GitStatus, LLMProbeResult, LLMSettings, MetadataBrowser, MetadataSearchResult, SavedDatasetQuery, SQLRun, SQLiteMetadataStatus, SQLiteQueryResult, ToolEvent, WorkspaceArtifact, WorkspaceFreshnessStatus, WorkspaceProblemSummary, WorkspaceSearchResult, WorkspaceSnapshot, WorkspaceTask, WorkspaceTaskRunResult, WorkspaceTaskSummary} from '../../types';
import {AgentToolPlanCard} from './AgentToolPlanCard';
import {ApprovalLogPanel} from './ApprovalLogPanel';
import {ArtifactStudioPanel} from './ArtifactStudioPanel';
import {CodeStudioPanel} from './CodeStudioPanel';
import {ConnectorProfilesCard} from './ConnectorProfilesCard';
import {DataOperationsPanel} from './DataOperationsPanel';
import {GitDiffPanel} from './GitDiffPanel';
import {LLMSettingsCard} from './LLMSettingsCard';
import {ToolTimeline} from './ToolTimeline';

type BottomStudioTab = 'code' | 'settings' | 'data' | 'tools' | 'artifacts' | 'git' | 'approvals' | 'activity';

type BottomStudioPanelProps = {
    activeTab: BottomStudioTab;
    activeFile: string;
    ariaLabel?: string;
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
    connectorProfileDraft: ConnectorProfile;
    connectorProfiles: ConnectorProfile[];
    connectorProfilesStatus: string;
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
    gitFileActionPreview: GitFileActionPreview | null;
    gitHunkActionPreview: GitHunkActionPreview | null;
    gitStatus: GitStatus | null;
    selectedGitChangePath: string;
    selectedGitFileDiff: GitFileDiff | null;
    dirtyTabPaths: string[];
    isArchivingArtifact: boolean;
    isCreatingDatasetChart: boolean;
    isCreatingDatasetSummary: boolean;
    isDeletingArtifact: boolean;
    isExportingDatasetQuery: boolean;
    isExportingDatasetSQL: boolean;
    isGeneratingGitInsight: boolean;
    isApplyingGitFileAction: boolean;
    isApplyingGitHunkAction: boolean;
    isLoadingGitFileDiff: boolean;
    isLoadingWorkspaceProblems: boolean;
    isLoadingWorkspaceTasks: boolean;
    isReviewingCode: boolean;
    isRunningWorkspaceTask: boolean;
    isPreviewingGitFileAction: boolean;
    isPreviewingGitHunkAction: boolean;
    isPreparingMetadataStore: boolean;
    isInspectingSQLiteConnector: boolean;
    isProfilingDataset: boolean;
    isPreviewingDatasetChart: boolean;
    isQueryingDataset: boolean;
    isQueryingDatasetSQL: boolean;
    isQueryingSQLiteConnector: boolean;
    isRefreshingStaleContext: boolean;
    isRunningAgentTool: boolean;
    isSavingDatasetQuery: boolean;
    isSavingDatasetSQLQuery: boolean;
    isSavingConnectorProfile: boolean;
    isSavingSettings: boolean;
    isSearchingMetadata: boolean;
    isSearchingWorkspace: boolean;
    isTestingConnection: boolean;
    metadataBrowser: MetadataBrowser | null;
    metadataSearchQuery: string;
    metadataSearchResults: MetadataSearchResult[];
    onArchiveArtifact: () => void;
    onApplyAssistantPatch: () => void;
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
    onDraftGitCommitMessage: () => void;
    onDraftPrDescription: () => void;
    onDraftPrSummary: () => void;
    onExecuteAgentTool: (item: AgentToolPlanItem) => void;
    onExportDatasetQuery: () => void;
    onExportDatasetSQL: () => void;
    onExportLineage: () => void;
    onExplainDependencyGraph: () => void;
    onGenerateTests: () => void;
    onInspectMetadata: () => void;
    onInspectSQLiteConnector: () => void;
    onInspectDataSource: (relPath: string) => void;
    onClearWorkspaceSearch: () => void;
    onMetadataSearchQueryChange: (content: string) => void;
    onOpenDataSource: (relPath: string) => void;
    onOpenArtifactSource: () => void;
    onProposePatch: () => void;
    onSelectGitChange: (path: string) => void;
    onOpenLineageSource: (relPath: string) => void;
    onPrepareMetadataStore: () => void;
    onProfileDataSource: (relPath: string) => void;
    onProfileDataset: () => void;
    onPreviewDatasetChart: () => void;
    onPreviewGitFileAction: (action: GitFileAction) => void;
    onApplyGitFileAction: (action: GitFileAction) => void;
    onPreviewGitHunkAction: (request: GitHunkActionRequest) => void;
    onApplyGitHunkAction: (request: GitHunkActionRequest) => void;
    onOpenCommandPalette: () => void;
    onQueryDataset: () => void;
    onQueryDatasetSQL: () => void;
    onCancelSQLiteConnectorQuery: () => void;
    onQuerySQLiteConnector: () => void;
    onRebuildDatasetDependency: (dependencyId: string) => void;
    onRefreshAgentPlan: () => void;
    onRefreshGitStatus: () => void;
    onRefreshLineage: () => void;
    onRefreshStaleContext: () => void;
    onRefreshWorkspaceProblems: () => void;
    onRefreshWorkspaceTasks: () => void;
    onReviewCurrentFile: () => void;
    onReviewGitDiff: () => void;
    onRunWorkspaceTask: (task: WorkspaceTask) => void;
    onReplayAgentToolRun: (run: AgentToolRunRecord) => void;
    onSaveDatasetQuery: () => void;
    onSaveDatasetSQLQuery: () => void;
    onSaveConnectorProfile: () => void;
    onDeleteConnectorProfile: (id: string) => void;
    onSaveSettings: () => void;
    onSelectArtifact: (artifact: WorkspaceArtifact) => void;
    onSettingsDraftChange: (field: keyof LLMSettings, value: string) => void;
    onConnectorProfileDraftChange: (field: keyof ConnectorProfile, value: string | number | boolean) => void;
    onSearchMetadata: () => void;
    onSearchWorkspace: () => void;
    onSelectSearchResult: (result: WorkspaceSearchResult) => void;
    onSQLiteConnectorQueryChange: (content: string) => void;
    onSQLiteConnectorQueryLabelChange: (content: string) => void;
    onPreviewSQLiteSchemaObject: (objectName: string) => void;
    onSQLiteConnectorResultLimitChange: (value: number) => void;
    onSaveSQLiteConnectorQuery: () => void;
    onSQLiteConnectorTimeoutSecondsChange: (value: number) => void;
    onSummarizeGitDiff: () => void;
    onTabChange: (tab: BottomStudioTab) => void;
    onTestConnection: () => void;
    onReplacePreviewChange: (value: string) => void;
    probeResult: LLMProbeResult | null;
    openTabs: FilePreview[];
    rebuildingDatasetDependencyId: string;
    savedDatasetQueries: SavedDatasetQuery[];
    savedDatasetSQLQueries: SavedDatasetQuery[];
    settingsDraft: LLMSettings;
    settingsStatus: string;
    sqliteConnectorQuery: string;
    sqliteConnectorQueryLabel: string;
    sqliteConnectorResultLimit: number;
    sqliteConnectorResult: SQLiteQueryResult | null;
    sqliteConnectorMetadata: ConnectorMetadata | null;
    sqliteConnectorTimeoutSeconds: number;
    savedSQLiteConnectorQueries: SavedDatasetQuery[];
    isSavingSQLiteConnectorQuery: boolean;
    sqliteStatus: SQLiteMetadataStatus | null;
    toolEvents: ToolEvent[];
    workspace: WorkspaceSnapshot | null;
    workspaceFreshness: WorkspaceFreshnessStatus | null;
    workspaceProblems: WorkspaceProblemSummary | null;
    workspaceSearchQuery: string;
    workspaceSearchRegex: boolean;
    workspaceSearchResults: WorkspaceSearchResult[];
    workspaceReplacePreview: string;
    workspaceTaskRun: WorkspaceTaskRunResult | null;
    workspaceTasks: WorkspaceTaskSummary | null;
    onWorkspaceSearchRegexChange: (value: boolean) => void;
    onWorkspaceSearchQueryChange: (value: string) => void;
    className?: string;
    showTabs?: boolean;
};

const drawerTabs: Array<{id: BottomStudioTab; label: string}> = [
    {id: 'git', label: 'Git'},
    {id: 'approvals', label: 'Approvals'},
    {id: 'activity', label: 'Activity'},
];

export function BottomStudioPanel({
    activeTab,
    activeFile,
    ariaLabel = 'Studio tools and settings',
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
    connectorProfileDraft,
    connectorProfiles,
    connectorProfilesStatus,
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
    gitFileActionPreview,
    gitHunkActionPreview,
    gitStatus,
    selectedGitChangePath,
    selectedGitFileDiff,
    dirtyTabPaths,
    isArchivingArtifact,
    isCreatingDatasetChart,
    isCreatingDatasetSummary,
    isDeletingArtifact,
    isExportingDatasetQuery,
    isExportingDatasetSQL,
    isGeneratingGitInsight,
    isApplyingGitFileAction,
    isApplyingGitHunkAction,
    isLoadingGitFileDiff,
    isLoadingWorkspaceProblems,
    isLoadingWorkspaceTasks,
    isReviewingCode,
    isRunningWorkspaceTask,
    isPreviewingGitFileAction,
    isPreviewingGitHunkAction,
    isPreparingMetadataStore,
    isInspectingSQLiteConnector,
    isProfilingDataset,
    isPreviewingDatasetChart,
    isQueryingDataset,
    isQueryingDatasetSQL,
    isQueryingSQLiteConnector,
    isRefreshingStaleContext,
    isRunningAgentTool,
    isSavingDatasetQuery,
    isSavingDatasetSQLQuery,
    isSavingConnectorProfile,
    isSavingSettings,
    isSearchingMetadata,
    isSearchingWorkspace,
    isTestingConnection,
    metadataBrowser,
    metadataSearchQuery,
    metadataSearchResults,
    onArchiveArtifact,
    onApplyAssistantPatch,
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
    onDraftGitCommitMessage,
    onDraftPrDescription,
    onDraftPrSummary,
    onExecuteAgentTool,
    onExportDatasetQuery,
    onExportDatasetSQL,
    onExportLineage,
    onExplainDependencyGraph,
    onGenerateTests,
    onInspectMetadata,
    onInspectSQLiteConnector,
    onInspectDataSource,
    onClearWorkspaceSearch,
    onMetadataSearchQueryChange,
    onOpenDataSource,
    onOpenArtifactSource,
    onProposePatch,
    onSelectGitChange,
    onOpenLineageSource,
    onPrepareMetadataStore,
    onProfileDataSource,
    onProfileDataset,
    onPreviewDatasetChart,
    onPreviewGitFileAction,
    onApplyGitFileAction,
    onPreviewGitHunkAction,
    onApplyGitHunkAction,
    onOpenCommandPalette,
    onQueryDataset,
    onQueryDatasetSQL,
    onCancelSQLiteConnectorQuery,
    onQuerySQLiteConnector,
    onRebuildDatasetDependency,
    onRefreshAgentPlan,
    onRefreshGitStatus,
    onRefreshLineage,
    onRefreshStaleContext,
    onRefreshWorkspaceProblems,
    onRefreshWorkspaceTasks,
    onReviewCurrentFile,
    onReviewGitDiff,
    onRunWorkspaceTask,
    onReplayAgentToolRun,
    onSaveDatasetQuery,
    onSaveDatasetSQLQuery,
    onSaveConnectorProfile,
    onDeleteConnectorProfile,
    onSaveSettings,
    onSelectArtifact,
    onSettingsDraftChange,
    onConnectorProfileDraftChange,
    onSearchMetadata,
    onSearchWorkspace,
    onSelectSearchResult,
    onSQLiteConnectorQueryChange,
    onSQLiteConnectorQueryLabelChange,
    onPreviewSQLiteSchemaObject,
    onSQLiteConnectorResultLimitChange,
    onSaveSQLiteConnectorQuery,
    onSQLiteConnectorTimeoutSecondsChange,
    onSummarizeGitDiff,
    onTabChange,
    onTestConnection,
    onReplacePreviewChange,
    probeResult,
    openTabs,
    rebuildingDatasetDependencyId,
    savedDatasetQueries,
    savedDatasetSQLQueries,
    settingsDraft,
    settingsStatus,
    sqliteConnectorQuery,
    sqliteConnectorQueryLabel,
    sqliteConnectorResultLimit,
    sqliteConnectorResult,
    sqliteConnectorMetadata,
    sqliteConnectorTimeoutSeconds,
    savedSQLiteConnectorQueries,
    isSavingSQLiteConnectorQuery,
    sqliteStatus,
    toolEvents,
    workspace,
    workspaceFreshness,
    workspaceProblems,
    workspaceSearchQuery,
    workspaceSearchRegex,
    workspaceSearchResults,
    workspaceReplacePreview,
    workspaceTaskRun,
    workspaceTasks,
    onWorkspaceSearchRegexChange,
    onWorkspaceSearchQueryChange,
    className = '',
    showTabs = true,
}: BottomStudioPanelProps) {
    return (
        <section className={['bottom-studio-panel', className].filter(Boolean).join(' ')} aria-label={ariaLabel}>
            {showTabs && (
                <div className="bottom-tabbar" role="tablist" aria-label="Studio drawer tabs">
                    {drawerTabs.map((tab) => (
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
            )}
            <div className="bottom-tab-content">
                {activeTab === 'code' && (
                    <CodeStudioPanel
                        activeFile={activeFile}
                        dirtyTabPaths={dirtyTabPaths}
                        filePreview={filePreview}
                        gitStatus={gitStatus}
                        selectedGitChangePath={selectedGitChangePath}
                        selectedGitFileDiff={selectedGitFileDiff}
                        isLoadingGitFileDiff={isLoadingGitFileDiff}
                        isLoadingWorkspaceProblems={isLoadingWorkspaceProblems}
                        isLoadingWorkspaceTasks={isLoadingWorkspaceTasks}
                        isReviewingCode={isReviewingCode}
                        isRunningWorkspaceTask={isRunningWorkspaceTask}
                        isSearchingWorkspace={isSearchingWorkspace}
                        openTabs={openTabs}
                        onClearWorkspaceSearch={onClearWorkspaceSearch}
                        onApplyAssistantPatch={onApplyAssistantPatch}
                        onDraftPrDescription={onDraftPrDescription}
                        onDraftPrSummary={onDraftPrSummary}
                        onExplainDependencyGraph={onExplainDependencyGraph}
                        onGenerateTests={onGenerateTests}
                        onOpenCommandPalette={onOpenCommandPalette}
                        onProposePatch={onProposePatch}
                        onRefreshGitStatus={onRefreshGitStatus}
                        onRefreshWorkspaceProblems={onRefreshWorkspaceProblems}
                        onRefreshWorkspaceTasks={onRefreshWorkspaceTasks}
                        onReviewCurrentFile={onReviewCurrentFile}
                        onReviewGitDiff={onReviewGitDiff}
                        onRunWorkspaceTask={onRunWorkspaceTask}
                        onSearchWorkspace={onSearchWorkspace}
                        onSelectGitChange={onSelectGitChange}
                        onSelectSearchResult={onSelectSearchResult}
                        onReplacePreviewChange={onReplacePreviewChange}
                        onWorkspaceSearchRegexChange={onWorkspaceSearchRegexChange}
                        onWorkspaceSearchQueryChange={onWorkspaceSearchQueryChange}
                        workspace={workspace}
                        workspaceFreshness={workspaceFreshness}
                        workspaceProblems={workspaceProblems}
                        workspaceSearchQuery={workspaceSearchQuery}
                        workspaceSearchRegex={workspaceSearchRegex}
                        workspaceSearchResults={workspaceSearchResults}
                        workspaceReplacePreview={workspaceReplacePreview}
                        workspaceTaskRun={workspaceTaskRun}
                        workspaceTasks={workspaceTasks}
                    />
                )}
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
                        <ConnectorProfilesCard
                            draft={connectorProfileDraft}
                            isSaving={isSavingConnectorProfile}
                            onDelete={onDeleteConnectorProfile}
                            onDraftChange={onConnectorProfileDraftChange}
                            onSave={onSaveConnectorProfile}
                            profiles={connectorProfiles}
                            status={connectorProfilesStatus}
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
                        isInspectingSQLiteConnector={isInspectingSQLiteConnector}
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
                        onInspectSQLiteConnector={onInspectSQLiteConnector}
                        onInspectDataSource={onInspectDataSource}
                        onMetadataSearchQueryChange={onMetadataSearchQueryChange}
                        onOpenDataSource={onOpenDataSource}
                        onPrepareMetadataStore={onPrepareMetadataStore}
                        onProfileDataSource={onProfileDataSource}
                        onProfileDataset={onProfileDataset}
                        onPreviewDatasetChart={onPreviewDatasetChart}
                        onQueryDataset={onQueryDataset}
                        onQueryDatasetSQL={onQueryDatasetSQL}
                        onCancelSQLiteConnectorQuery={onCancelSQLiteConnectorQuery}
                        onQuerySQLiteConnector={onQuerySQLiteConnector}
                        onRebuildDatasetDependency={onRebuildDatasetDependency}
                        onRefreshStaleContext={onRefreshStaleContext}
                        onSaveDatasetQuery={onSaveDatasetQuery}
                        onSaveDatasetSQLQuery={onSaveDatasetSQLQuery}
                        onSearchMetadata={onSearchMetadata}
                        onSQLiteConnectorQueryChange={onSQLiteConnectorQueryChange}
                        onSQLiteConnectorQueryLabelChange={onSQLiteConnectorQueryLabelChange}
                        onPreviewSQLiteSchemaObject={onPreviewSQLiteSchemaObject}
                        onSQLiteConnectorResultLimitChange={onSQLiteConnectorResultLimitChange}
                        onSaveSQLiteConnectorQuery={onSaveSQLiteConnectorQuery}
                        onSQLiteConnectorTimeoutSecondsChange={onSQLiteConnectorTimeoutSecondsChange}
                        rebuildingDatasetDependencyId={rebuildingDatasetDependencyId}
                        savedDatasetQueries={savedDatasetQueries}
                        savedDatasetSQLQueries={savedDatasetSQLQueries}
                        savedSQLiteConnectorQueries={savedSQLiteConnectorQueries}
                        sqliteConnectorQuery={sqliteConnectorQuery}
                        sqliteConnectorQueryLabel={sqliteConnectorQueryLabel}
                        sqliteConnectorResultLimit={sqliteConnectorResultLimit}
                        sqliteConnectorResult={sqliteConnectorResult}
                        sqliteConnectorMetadata={sqliteConnectorMetadata}
                        sqliteConnectorTimeoutSeconds={sqliteConnectorTimeoutSeconds}
                        isSavingSQLiteConnectorQuery={isSavingSQLiteConnectorQuery}
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
                {activeTab === 'git' && (
                    <GitDiffPanel
                        gitFileActionPreview={gitFileActionPreview}
                        gitHunkActionPreview={gitHunkActionPreview}
                        gitStatus={gitStatus}
                        selectedGitChangePath={selectedGitChangePath}
                        selectedGitFileDiff={selectedGitFileDiff}
                        isGeneratingGitInsight={isGeneratingGitInsight}
                        isApplyingGitFileAction={isApplyingGitFileAction}
                        isApplyingGitHunkAction={isApplyingGitHunkAction}
                        isLoadingGitFileDiff={isLoadingGitFileDiff}
                        isPreviewingGitFileAction={isPreviewingGitFileAction}
                        isPreviewingGitHunkAction={isPreviewingGitHunkAction}
                        onDraftCommitMessage={onDraftGitCommitMessage}
                        onPreviewGitFileAction={onPreviewGitFileAction}
                        onApplyGitFileAction={onApplyGitFileAction}
                        onPreviewGitHunkAction={onPreviewGitHunkAction}
                        onApplyGitHunkAction={onApplyGitHunkAction}
                        onDraftPrDescription={onDraftPrDescription}
                        onDraftPrSummary={onDraftPrSummary}
                        onGenerateTests={onGenerateTests}
                        onRefreshGitStatus={onRefreshGitStatus}
                        onReviewGitDiff={onReviewGitDiff}
                        onSelectGitChange={onSelectGitChange}
                        onSummarizeDiff={onSummarizeGitDiff}
                    />
                )}
                {activeTab === 'approvals' && <ApprovalLogPanel records={approvalRecords} />}
                {activeTab === 'activity' && <ToolTimeline events={toolEvents} />}
            </div>
        </section>
    );
}

export type {BottomStudioTab};
