import {existsSync, readFileSync} from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

const checks = [
    {
        file: 'src/features/shell/NexusDeskShell.tsx',
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
            'CreateDatasetSummaryArtifact',
            'PreviewDatasetChart',
            'SaveDatasetQuery',
            'ListDatasetQueries',
            'GetArtifactMetadata',
            'ListAgentTools',
            'ArchiveArtifact',
            'DeleteArtifact',
            'CreateScanReportArtifact',
            'ListApprovals',
            'SearchWorkspace',
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
            'buildAgentToolPlan',
        ],
    },
    {
        file: 'src/features/shell/AgentToolPlanCard.tsx',
        terms: ['AgentToolPlanCard', 'Tool Plan', 'tool-plan-list', 'requiresApproval'],
    },
    {
        file: 'src/features/shell/LLMSettingsCard.tsx',
        terms: ['recommendedModelOptions', 'qwen3:8b', 'gpt-oss:20b', 'gemma4:26b', '<select', 'probe-runtime'],
    },
    {
        file: 'src/features/shell/WorkbenchPanel.tsx',
        terms: ['editor-tabs', 'markdownViewMode', 'markdown-view-toggle', 'markdown-document-preview', 'studio-mode-strip', 'resolveStudioMode', 'Data Studio', 'Summarize', 'onSummarizeContext', 'onSelectTab', 'onCloseTab', 'onDeleteFile', 'onMoveFile', 'onPinProjectContext', 'DataStudioPanel', 'OperationsInspector', 'onExportDatasetQuery', 'dataset-query-csv', 'file-write-editor', 'MonacoFileEditor', 'MonacoCodePreview', 'editor-find', 'findInputRef', 'dirty-indicator', 'dirtyTabPaths', 'countFindMatches'],
    },
    {
        file: 'src/features/shell/DataStudioPanel.tsx',
        terms: ['DatasetQueryPanel', 'DatasetChartPanel', 'DatasetChartPreview', 'SortableDataTable', 'table-pager', 'chart-config-list'],
    },
    {
        file: 'src/features/shell/ArtifactMetadataPanel.tsx',
        terms: ['ArtifactMetadataPanel', 'artifact-chart-preview', 'Configuration', 'Open source', 'Archive', 'Delete'],
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
        terms: ['MonacoFileEditor', 'loadMonaco', 'languageForFile', 'nexusdesk-light', 'KeyCode.KeyS'],
    },
    {
        file: 'src/features/shell/MonacoCodePreview.tsx',
        terms: ['MonacoCodePreview', 'readOnly', 'updateSearchDecorations', 'monaco-find-highlight'],
    },
    {
        file: 'src/features/shell/monacoRuntime.ts',
        terms: ['monaco-editor', 'MonacoEnvironment', 'loadMonaco', 'languageForFile', 'nexusdesk-light'],
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
        terms: ['Code Studio', 'AI Assistant', 'Data Studio', 'Document Studio', 'Ops Studio'],
    },
    {
        file: 'src/features/shell/WorkspaceNavigator.tsx',
        terms: ['workspace-search', 'search-results', 'search-result-group', 'ScanStatusDetails', 'scanStatusSummary', 'Expand all', 'Collapse all', 'Save scan'],
    },
    {
        file: 'src/features/shell/AgentChatCard.tsx',
        terms: ['ChatMessageContent', 'chat-card-header', 'Save answer', 'textarea', 'context-pack-list', 'context-pack-preview', 'onRemoveContextPath', 'Clear pack'],
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
            '.artifact-action-row',
            '.sortable-data-table',
            '.scan-status-details',
            '.search-result-group',
            '.studio-mode-strip',
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
        terms: ['AskLLMContextPack', 'PreviewFileWrite', 'ApplyFileDelete', 'ApplyFileMove', 'ProfileDataset', 'CreateDatasetChartArtifact', 'CreateDatasetQueryArtifact', 'CreateDatasetSummaryArtifact', 'CreateChatMarkdownArtifact', 'CreateScanReportArtifact', 'PreviewChatContextPack', 'PreviewDatasetChart', 'SaveDatasetQuery', 'ListApprovals', 'ListAgentTools', 'ArchiveArtifact', 'DeleteArtifact'],
    },
    {
        file: 'scripts/visual-smoke.mjs',
        terms: ['playwright', 'desktop.png', 'mobile.png'],
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
    console.error('NexusDesk frontend smoke failed:');
    for (const failure of failures) {
        console.error(`- ${failure}`);
    }
    process.exit(1);
}

console.log(`NexusDesk frontend smoke passed (${checks.length} files checked).`);
