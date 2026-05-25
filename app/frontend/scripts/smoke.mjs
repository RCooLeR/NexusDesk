import {existsSync, readFileSync} from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

const checks = [
    {
        file: 'src/features/shell/NexusShell.tsx',
        terms: [
            'AskLLMStreamContextPack',
            'PreviewFileWrite',
            'PreviewFileDelete',
            'ApplyFileDelete',
            'PreviewFileMove',
            'ApplyFileMove',
            'ProfileDataset',
            'QueryDataset',
            'CreateDatasetChartArtifact',
            'CreateDatasetQueryArtifact',
            'CreateDatasetSQLArtifact',
            'CreateDatasetSummaryArtifact',
            'PreviewDatasetChart',
            'SaveDatasetQuery',
            'ListDatasetQueries',
            'ListDatasetDependencies',
            'ListDatasetSQLRuns',
            'GetArtifactMetadata',
            'ListAgentTools',
            'ListAgentToolRuns',
            'PreviewAgentTool',
            'ExecuteAgentTool',
            'QueryDatasetSQL',
            'QueryWorkspaceSQLite',
            'SearchMetadata',
            'ExportArtifactLineageJSON',
            'EnsureSQLiteMetadataStore',
            'InspectMetadataStore',
            'GetArtifactLineage',
            'CheckWorkspaceFreshness',
            'CompareArtifacts',
            'ArchiveArtifact',
            'DeleteArtifact',
            'CreateScanReportArtifact',
            'ListApprovals',
            'SearchWorkspace',
            'GetGitStatus',
            'GetGitFileDiff',
            'ApprovalRequestModal',
            'requestApproval',
            'QuickOpenPalette',
            'CommandPalette',
            'commandActions',
            'isCommandPaletteOpen',
            'commandPaletteQuery',
            'selectQuickOpenNode',
            'saveActiveDraftShortcut',
            'selectAdjacentTab',
            'Close Active Tab',
            'Next Editor Tab',
            'startNewFileDraft',
            'New File',
            'Delete Active File',
            'deleteActiveFile',
            'Rename Or Move Active File',
            'moveActiveFile',
            'defaultNewFileContent',
            'dirtyDraftPaths',
            'editingFilePaths',
            'writeProposals',
            'contextPackPaths',
            'pinProjectContext',
            'openTabs',
            'listenForChatStream',
            'CreateChatMarkdownArtifact',
            'PreviewChatContextPack',
            'summarizeSelectedContext',
            'createDatasetChart',
            'createDatasetSummary',
            'previewDatasetChart',
            'saveCurrentDatasetQuery',
            'exportDatasetQuery',
            'createScanReportArtifact',
            'archiveActiveArtifact',
            'deleteActiveArtifact',
            'querySelectedDatasetSQL',
            'exportDatasetSQL',
            'prepareSQLiteMetadataStore',
            'inspectMetadataStore',
            'executeAgentTool',
            'dryRunAgentTool',
            'buildAgentToolPlan',
            'renderStudioPanel',
            'useResizablePanels',
            'useStudioNavigation',
            'settingsForSelectedModel',
            'Model request queued',
            'First token received',
            'refreshGitStatus',
            'summarizeGitDiff',
            'draftGitCommitMessage',
            'selectedGitDiffPromptContext',
            'handleTreeContextAction',
            'main-studio-panel',
            'showTabs: false',
        ],
    },
    {
        file: 'src/api/wailsClient.ts',
        terms: ['wailsjs/go/main/App', 'wailsjs/runtime/runtime', 'EventsOn'],
    },
    {
        file: 'src/features/shell/useResizablePanels.ts',
        terms: ['useResizablePanels', 'startNavigatorResize', 'startAgentResize', 'startBottomResize', 'beginResize', 'clamp', 'localStorage', 'nexus:resizable-panels'],
    },
    {
        file: 'src/features/shell/useStudioNavigation.ts',
        terms: ['useStudioNavigation', 'studioRouteSurfaceTab', 'changeStudioRoute', 'changeBottomStudioTab', 'mainStudioTabForRoute', 'localStorage', 'nexus:studio-navigation', "'git'"],
    },
    {
        file: 'src/features/shell/AgentToolPlanCard.tsx',
        terms: ['AgentToolPlanCard', 'Tool Plan', 'tool-plan-list', 'requiresApproval', 'Dry run', 'Execute', 'tool-run-list', 'Replay dry run', 'Diff target'],
    },
    {
        file: 'src/features/shell/CodeStudioPanel.tsx',
        terms: ['CodeStudioPanel', 'Workbench', 'Project Session', 'Repository', 'Code Queues', 'selectedGitChangePath', 'onSelectGitChange', 'stagedFiles', 'unstagedFiles', 'Refresh git', 'Commands'],
    },
    {
        file: 'src/features/shell/GitDiffPanel.tsx',
        terms: ['GitDiffPanel', 'Working Tree Diff', 'Staged Diff', 'Unstaged Diff', 'selectedGitChangePath', 'selectedGitFileDiff', 'diffMode', 'changes', 'Diff Only', 'collectChangedRows', 'hunkTargets', 'Previous hunk', 'Next hunk', 'Summarize diff', 'Draft commit', 'isGeneratingGitInsight', 'git-diff-panel', 'git-diff-split', 'git-diff-changes', 'git-diff-view', 'onSelectGitChange', 'Refresh git', 'buildGitChangeTree', 'git-change-tree'],
    },
    {
        file: 'src/features/shell/LLMSettingsCard.tsx',
        terms: ['recommendedModelOptions', '<select', 'maxContextTokens', 'responseReserveTokens', 'num_ctx', 'num_predict', 'max_tokens', 'probe-runtime'],
    },
    {
        file: 'src/features/shell/llmModelCatalog.ts',
        terms: ['recommendedModelOptions', 'qwen3:8b', 'gpt-oss:20b', 'gemma4:26b', 'settingsForSelectedModel', 'settingsWithRuntimeContext', 'responseReserveForContext'],
    },
    {
        file: 'src/features/shell/WorkbenchPanel.tsx',
        terms: ['editor-tabs', 'markdownViewMode', 'markdown-view-toggle', 'markdown-document-preview', 'Summarize', 'onSummarizeContext', 'onSelectTab', 'onCloseTab', 'onDeleteFile', 'onMoveFile', 'onPinProjectContext', 'file-write-editor', 'MonacoFileEditor', 'MonacoCodePreview', 'editor-find', 'findInputRef', 'dirty-indicator', 'dirtyTabPaths', 'countFindMatches'],
    },
    {
        file: 'src/features/shell/DataStudioPanel.tsx',
        terms: ['DatasetQueryPanel', 'DatasetChartPanel', 'DatasetChartPreview', 'SortableDataTable', 'table-pager', 'chart-config-list', 'Read-only SQL', 'DuckDB-compatible SQL query', 'Export SQL'],
    },
    {
        file: 'src/features/shell/ArtifactMetadataPanel.tsx',
        terms: ['ArtifactMetadataPanel', 'artifact-chart-preview', 'Configuration', 'Open source', 'Archive', 'Delete'],
    },
    {
        file: 'src/features/shell/ArtifactStudioPanel.tsx',
        terms: ['ArtifactStudioPanel', 'ArtifactComparisonPanel', 'ArtifactLineagePanel', 'artifact-studio-panel', 'onSelectArtifact', 'Export JSON'],
    },
    {
        file: 'src/features/shell/ApprovalLogPanel.tsx',
        terms: ['ApprovalLogPanel', 'approval-log-row'],
    },
    {
        file: 'src/features/shell/ApprovalRequestModal.tsx',
        terms: ['ApprovalRequestModal', 'Approval Required', 'risk-dot'],
    },
    {
        file: 'src/features/shell/OperationsInspector.tsx',
        terms: ['OperationsInspector', 'Read-only inspector', 'docker-compose', 'parseComposeServices', 'compose-service-list'],
    },
    {
        file: 'src/features/shell/MonacoFileEditor.tsx',
        terms: ['MonacoFileEditor', 'loadMonaco', 'languageForFile', 'nexus-light', 'KeyCode.KeyS'],
    },
    {
        file: 'src/features/shell/MonacoCodePreview.tsx',
        terms: ['MonacoCodePreview', 'readOnly', 'updateSearchDecorations', 'monaco-find-highlight'],
    },
    {
        file: 'src/features/shell/monacoRuntime.ts',
        terms: ['monaco-editor', 'MonacoEnvironment', 'loadMonaco', 'languageForFile', 'nexus-light', 'basic-languages/go/go.contribution', 'language/typescript/monaco.contribution'],
    },
    {
        file: 'src/features/shell/HighlightedCode.tsx',
        terms: ['searchQuery', 'find-highlight', 'renderTokenText'],
    },
    {
        file: 'src/features/shell/QuickOpenPalette.tsx',
        terms: ['QuickOpenPalette', 'quick-open-result', 'scoreQuickOpenEntry', 'maxQuickOpenResults', 'ArrowDown', 'ArrowUp'],
    },
    {
        file: 'src/features/shell/CommandPalette.tsx',
        terms: ['CommandPalette', 'command-result', 'scoreCommand', 'maxCommandResults', 'ArrowDown', 'ArrowUp'],
    },
    {
        file: 'src/brand/assets.ts',
        terms: ['productBrand', 'Nexus Augentic Studio', 'Agentic work. Augmented by context.', 'logoHorizontalDark', 'StudioRouteId', "code: 'code'", 'Workbench', 'Data & Analytics', 'Artifacts', 'Settings', 'studioRouteSurfaceTab', 'studioRoutePrimarySurface'],
    },
    {
        file: 'src/features/shell/WorkspaceRail.tsx',
        terms: ['activeRoute', 'onRouteChange', 'data-studio-route', 'Main studio menu', 'studioRoutePrimarySurface'],
    },
    {
        file: 'src/features/shell/WorkspaceNavigator.tsx',
        terms: ['workspace-search', 'project-tree', 'TreeNodeButton', 'tree-indent-guide', 'tree-node-badge', 'search-results', 'search-result-group', 'workspace.root', 'Expand all', 'Collapse all', 'Save scan', 'TreeContextAction', 'tree-context-menu'],
    },
    {
        file: 'src/features/shell/AgentChatCard.tsx',
        terms: ['ChatMessageContent', 'recommendedModelOptions', 'chat-card-header', 'Save answer', 'textarea', 'composer-shell', 'composer-controls', 'Submit mode', 'onModelChange', 'onRunAgent', 'Agent', 'Clear pack', 'staleSourcePaths', 'Context changed since this answer was created.'],
    },
    {
        file: 'src/features/shell/BottomStudioPanel.tsx',
        terms: ['BottomStudioPanel', 'drawerTabs', 'Git', 'Approvals', 'Activity', 'GitDiffPanel', 'onSummarizeGitDiff', 'onDraftGitCommitMessage', 'CodeStudioPanel', 'DataOperationsPanel', 'LLMSettingsCard', 'AgentToolPlanCard', 'ArtifactStudioPanel', 'ApprovalLogPanel', 'ToolTimeline', 'bottom-tabbar', 'showTabs'],
    },
    {
        file: 'src/features/shell/DataOperationsPanel.tsx',
        terms: ['DataOperationsPanel', 'Data & Analytics', 'Operations', 'Metadata', 'Profile dataset', 'OperationsInspector', 'MetadataBrowserPanel', 'SQLiteConnectorPanel', 'WorkspaceFreshnessPanel'],
    },
    {
        file: 'src/features/shell/ChatMessageContent.tsx',
        terms: ['parseMarkdownBlocks', 'ChatTable', 'chat-markdown', 'chat-code-block'],
    },
    {
        file: 'src/App.css',
        terms: [
            '.app-shell',
            '.workbench',
            '.navigator-resizer',
            '.agent-resizer',
            '.bottom-panel-resizer',
            '.bottom-studio-panel',
            '.main-studio-panel',
            '.code-studio-panel',
            '.code-studio-metrics',
            '.code-studio-row',
            '.code-studio-toolbar',
            '.git-change-tree',
            '.git-change-file',
            '.git-diff-panel',
            '.git-diff-controls',
            '.git-diff-changes',
            '.git-diff-split',
            '.git-diff-view',
            '.tree-context-menu',
            '.bottom-tabbar',
            '.settings-page',
            '.settings-number-grid',
            '.data-operations-panel',
            '.composer-shell',
            '.composer-controls',
            '.composer-submit',
            '.artifact-studio-panel',
            '.artifact-studio-list',
            '.artifact-lineage-panel',
            '.file-preview',
            '.context-pack-list',
            '.context-pack-preview',
            '.chat-markdown',
            '.chat-table',
            '.file-write-editor',
            '.monaco-file-editor',
            '.monaco-code-preview',
            '.monaco-find-highlight',
            '.dataset-profile-summary',
            '.dataset-chart-panel',
            '.dataset-filter-row',
            '.dataset-chart-preview',
            '.artifact-metadata-panel',
            '.approval-log-panel',
            '.approval-modal',
            '.operations-inspector-panel',
            '.compose-service-list',
            '.agent-tool-plan-card',
            '.tool-plan-list',
            '.tool-run-list',
            '.artifact-action-row',
            '.artifact-comparison-panel',
            '.metadata-store-panel',
            '.metadata-browser-panel',
            '.metadata-browser-controls',
            '.metadata-column-grid',
            '.metadata-history-results',
            '.lineage-filter-row',
            '.lineage-graph-layout',
            '.lineage-node',
            '.sqlite-connector-panel',
            '.dataset-lineage-history',
            '.stale-source-warning',
            '.dataset-sql-panel',
            '.sortable-data-table',
            '.project-tree',
            '.tree-indent-guide',
            '.tree-node-main',
            '.tree-node-badge',
            '.search-result-group',
            '.quick-open',
            '.quick-open-result',
            '.command-palette',
            '.command-result',
            '.command-shortcut',
            '.editor-find',
            '.dirty-indicator',
            '.find-highlight',
        ],
    },
    {
        file: 'wailsjs/go/main/App.d.ts',
        terms: ['AskLLMContextPack', 'RunAgent', 'PreviewFileWrite', 'ApplyFileDelete', 'ApplyFileMove', 'ProfileDataset', 'CreateDatasetChartArtifact', 'CreateDatasetQueryArtifact', 'CreateDatasetSQLArtifact', 'CreateDatasetSummaryArtifact', 'CreateChatMarkdownArtifact', 'CreateScanReportArtifact', 'PreviewChatContextPack', 'PreviewDatasetChart', 'SaveDatasetQuery', 'SaveDatasetSQLQuery', 'ListDatasetSQLQueries', 'ListDatasetDependencies', 'ListDatasetSQLRuns', 'RefreshStaleContext', 'SearchMetadata', 'QueryWorkspaceSQLite', 'ExportArtifactLineageJSON', 'GetGitStatus', 'GetGitFileDiff', 'ListApprovals', 'ListAgentTools', 'ListAgentToolRuns', 'PreviewAgentTool', 'ExecuteAgentTool', 'QueryDatasetSQL', 'EnsureSQLiteMetadataStore', 'InspectMetadataStore', 'GetArtifactLineage', 'CheckWorkspaceFreshness', 'CompareArtifacts', 'ArchiveArtifact', 'DeleteArtifact'],
    },
    {
        file: '../app_metadata.go',
        terms: ['metadataMirrorData', 'mirrorMetadataStore', 'recordDatasetDependency', 'recordSQLRun', 'datasetViews', 'hashForID'],
    },
    {
        file: '../app_git.go',
        terms: ['GetGitStatus', 'GetGitFileDiff', 'gitFileDiff', 'cleanGitRelPath', 'parseGitStatus', 'gitDiffMaxBytes'],
    },
    {
        file: '../internal/llm/chat.go',
        terms: ['MaxTokens', 'max_tokens', 'num_predict', 'num_ctx'],
    },
    {
        file: 'scripts/visual-smoke.mjs',
        terms: ['playwright', 'installNexusMocks', 'desktop.png', 'mobile.png', 'visual-baselines', 'manifest.json', 'navigator-resize', 'project-tree', 'dataMainSurface', 'code-route', 'tool-run-detail', 'metadata-browser', 'metadata-history'],
    },
    {
        file: 'scripts/visual-fixtures.mjs',
        terms: ['installNexusMocks', 'ListDatasetDependencies', 'ListDatasetSQLRuns', 'SearchMetadata', 'QueryWorkspaceSQLite', 'ExportArtifactLineageJSON', 'ImportArtifactLineageJSON', 'dependencies'],
    },
    {
        file: 'dist/index.html',
        terms: ['<script type="module"', '<div id="root">'],
    },
];

const failures = [];

for (const check of checks) {
    const target = path.join(root, check.file);
    if (!existsSync(target)) {
        failures.push(`${check.file} is missing`);
        continue;
    }

    const content = readFileSync(target, 'utf8');
    for (const term of check.terms) {
        if (!content.includes(term)) {
            failures.push(`${check.file} does not contain ${term}`);
        }
    }
}

if (failures.length > 0) {
    console.error('Nexus frontend smoke failed:');
    for (const failure of failures) {
        console.error(`- ${failure}`);
    }
    process.exit(1);
}

console.log(`Nexus frontend smoke passed (${checks.length} files checked).`);
