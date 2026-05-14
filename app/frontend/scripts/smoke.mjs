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
            'ProfileDataset',
            'QueryDataset',
            'SearchWorkspace',
            'QuickOpenPalette',
            'CommandPalette',
            'commandActions',
            'isCommandPaletteOpen',
            'commandPaletteQuery',
            'selectQuickOpenNode',
            'saveActiveDraftShortcut',
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
        ],
    },
    {
        file: 'src/features/shell/LLMSettingsCard.tsx',
        terms: ['recommendedModelOptions', 'qwen3:8b', 'gpt-oss:20b', 'gemma4:26b', '<select', 'probe-runtime'],
    },
    {
        file: 'src/features/shell/WorkbenchPanel.tsx',
        terms: ['editor-tabs', 'markdownViewMode', 'markdown-view-toggle', 'markdown-document-preview', 'studio-mode-strip', 'resolveStudioMode', 'Data Studio', 'Summarize', 'onSummarizeContext', 'onSelectTab', 'onCloseTab', 'onPinProjectContext', 'DatasetQueryPanel', 'file-write-editor', 'MonacoFileEditor', 'editor-find', 'dirty-indicator', 'dirtyTabPaths', 'countFindMatches'],
    },
    {
        file: 'src/features/shell/MonacoFileEditor.tsx',
        terms: ['monaco-editor', 'MonacoEnvironment', 'loadMonaco', 'languageForFile', 'nexusdesk-light', 'KeyCode.KeyS'],
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
        terms: ['workspace-search', 'search-results', 'Expand all', 'Collapse all'],
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
            '.dataset-profile-summary',
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
        terms: ['AskLLMContextPack', 'PreviewFileWrite', 'ProfileDataset', 'CreateChatMarkdownArtifact', 'PreviewChatContextPack'],
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
