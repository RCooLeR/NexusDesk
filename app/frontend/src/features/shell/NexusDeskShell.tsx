import {useEffect, useMemo, useRef, useState} from 'react';
import type {CSSProperties, MouseEvent as ReactMouseEvent} from 'react';
import {
    ApplyFileWrite,
    ApplyFileDelete,
    ApplyFileMove,
    AskLLM,
    AskLLMContextPack,
    AskLLMStream,
    AskLLMStreamContextPack,
    ClearRecentWorkspaces,
    ClearChatHistory,
    ArchiveArtifact,
    CompareArtifacts,
    CreateChatMarkdownArtifact,
    CreateDatasetChartArtifact,
    CreateDatasetQueryArtifact,
    CreateDatasetSummaryArtifact,
    CreateMarkdownReport,
    CreateScanReportArtifact,
    DeleteArtifact,
    EnsureSQLiteMetadataStore,
    ExecuteAgentTool,
    GetArtifactMetadata,
    GetChatHistory,
    ListAgentTools,
    ListAgentToolRuns,
    ListApprovals,
    ListDatasetQueries,
    ListDatasetProfiles,
    GetRecentWorkspaces,
    ListArtifacts,
    OpenWorkspace,
    PreviewFileDelete,
    PreviewFileMove,
    PreviewAgentTool,
    PreviewDatasetChart,
    PreviewChatContextPack,
    PreviewFileWrite,
    ProfileDataset,
    QueryDataset,
    QueryDatasetSQL,
    ReadWorkspaceFile,
    RemoveRecentWorkspace,
    RefreshWorkspace,
    SaveLLMSettings,
    SaveDatasetQuery,
    SearchWorkspace,
    SelectWorkspace,
    TestLLMConnection,
} from '../../../wailsjs/go/main/App';
import {EventsOn} from '../../../wailsjs/runtime/runtime';
import type {
    ApprovalRecord,
    AgentToolDescriptor,
    AgentToolPlanItem,
    AgentToolRunRecord,
    ArtifactComparison,
    ArtifactMetadata,
    ChatStreamEvent,
    ChatMessage,
    ContextPreview,
    DatasetChartResult,
    DatasetProfile,
    DatasetQueryResult,
    DatasetSQLQueryResult,
    FileNode,
    FilePreview,
    FileWriteProposal,
    LLMChatResult,
    LLMProbeResult,
    LLMSettings,
    MarkdownReport,
    RecentWorkspace,
    SavedDatasetQuery,
    SQLiteMetadataStatus,
    StartupState,
    ToolEvent,
    WorkspaceArtifact,
    WorkspaceSearchResult,
    WorkspaceOpenResult,
    WorkspaceSnapshot,
} from '../../types';
import {AgentPanel} from './AgentPanel';
import {ApprovalRequestModal, type ApprovalPrompt} from './ApprovalRequestModal';
import {CommandPalette, type CommandAction} from './CommandPalette';
import {QuickOpenPalette} from './QuickOpenPalette';
import {WorkbenchPanel} from './WorkbenchPanel';
import {WorkspaceNavigator} from './WorkspaceNavigator';
import {WorkspaceRail} from './WorkspaceRail';

type NexusDeskShellProps = {
    state: StartupState;
    workspace: WorkspaceSnapshot | null;
    recentWorkspaces: RecentWorkspace[];
    llmSettings: LLMSettings;
    onWorkspaceChange: (workspace: WorkspaceSnapshot) => void;
    onRecentWorkspacesChange: (workspaces: RecentWorkspace[]) => void;
    onLLMSettingsChange: (settings: LLMSettings) => void;
};

type SendPromptOptions = {
    clearComposer: boolean;
    contextPaths?: string[];
    saveArtifactSource?: string;
    saveArtifactTitle?: string;
};

type AssistantArtifactWriteRequest = {
    title: string;
    content: string;
    contextRelPath: string;
    prompt: string;
    model: string;
    source: string;
    sourcePaths: string[];
    eventTitle: string;
};

const chatStreamEventName = 'nexusdesk:chat-stream';
const navigatorMinWidth = 220;
const navigatorMaxWidth = 460;
const railWidth = 56;

export function NexusDeskShell({
    state,
    workspace,
    recentWorkspaces,
    llmSettings,
    onWorkspaceChange,
    onRecentWorkspacesChange,
    onLLMSettingsChange,
}: NexusDeskShellProps) {
    const [activeFile, setActiveFile] = useState('docs/08_DELIVERY_PLAN.md');
    const [workspaceStatus, setWorkspaceStatus] = useState('No workspace opened yet.');
    const [isOpeningWorkspace, setIsOpeningWorkspace] = useState(false);
    const [isRefreshingWorkspace, setIsRefreshingWorkspace] = useState(false);
    const [isManagingRecent, setIsManagingRecent] = useState(false);
    const [filePreview, setFilePreview] = useState<FilePreview | null>(null);
    const [openTabs, setOpenTabs] = useState<FilePreview[]>([]);
    const [isLoadingPreview, setIsLoadingPreview] = useState(false);
    const [expandedDirectories, setExpandedDirectories] = useState<Set<string>>(() => new Set());
    const [settingsDraft, setSettingsDraft] = useState<LLMSettings>(llmSettings);
    const [settingsStatus, setSettingsStatus] = useState('LLM provider not connected yet.');
    const [isSavingSettings, setIsSavingSettings] = useState(false);
    const [isTestingConnection, setIsTestingConnection] = useState(false);
    const [probeResult, setProbeResult] = useState<LLMProbeResult | null>(null);
    const [chatPrompt, setChatPrompt] = useState('');
    const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);
    const [chatStatus, setChatStatus] = useState('Select text context and ask the assistant.');
    const [contextPackPaths, setContextPackPaths] = useState<string[]>([]);
    const [contextPackPreview, setContextPackPreview] = useState<ContextPreview | null>(null);
    const [localToolEvents, setLocalToolEvents] = useState<ToolEvent[]>(state.toolEvents);
    const [artifacts, setArtifacts] = useState<WorkspaceArtifact[]>([]);
    const [datasetProfiles, setDatasetProfiles] = useState<DatasetProfile[]>([]);
    const [activeDatasetProfile, setActiveDatasetProfile] = useState<DatasetProfile | null>(null);
    const [datasetQuery, setDatasetQuery] = useState('');
    const [datasetSQLQuery, setDatasetSQLQuery] = useState('');
    const [datasetQueryLabel, setDatasetQueryLabel] = useState('');
    const [datasetQueryResult, setDatasetQueryResult] = useState<DatasetQueryResult | null>(null);
    const [datasetSQLQueryResult, setDatasetSQLQueryResult] = useState<DatasetSQLQueryResult | null>(null);
    const [savedDatasetQueries, setSavedDatasetQueries] = useState<SavedDatasetQuery[]>([]);
    const [isQueryingDataset, setIsQueryingDataset] = useState(false);
    const [isQueryingDatasetSQL, setIsQueryingDatasetSQL] = useState(false);
    const [isSavingDatasetQuery, setIsSavingDatasetQuery] = useState(false);
    const [datasetChartType, setDatasetChartType] = useState('bar');
    const [datasetChartCategory, setDatasetChartCategory] = useState('');
    const [datasetChartValue, setDatasetChartValue] = useState('');
    const [datasetChartPreview, setDatasetChartPreview] = useState<DatasetChartResult | null>(null);
    const [isPreviewingDatasetChart, setIsPreviewingDatasetChart] = useState(false);
    const [isCreatingDatasetChart, setIsCreatingDatasetChart] = useState(false);
    const [isCreatingDatasetSummary, setIsCreatingDatasetSummary] = useState(false);
    const [isExportingDatasetQuery, setIsExportingDatasetQuery] = useState(false);
    const [artifactMetadata, setArtifactMetadata] = useState<ArtifactMetadata | null>(null);
    const [approvalRecords, setApprovalRecords] = useState<ApprovalRecord[]>([]);
    const [workspaceSearchQuery, setWorkspaceSearchQuery] = useState('');
    const [workspaceSearchResults, setWorkspaceSearchResults] = useState<WorkspaceSearchResult[]>([]);
    const [isSearchingWorkspace, setIsSearchingWorkspace] = useState(false);
    const [isSendingPrompt, setIsSendingPrompt] = useState(false);
    const [isCreatingReport, setIsCreatingReport] = useState(false);
    const [isCreatingScanReport, setIsCreatingScanReport] = useState(false);
    const [isSavingChatArtifact, setIsSavingChatArtifact] = useState(false);
    const [isSummarizingContext, setIsSummarizingContext] = useState(false);
    const [isProfilingDataset, setIsProfilingDataset] = useState(false);
    const [isPreviewingWrite, setIsPreviewingWrite] = useState(false);
    const [isApplyingWrite, setIsApplyingWrite] = useState(false);
    const [isDeletingFile, setIsDeletingFile] = useState(false);
    const [isMovingFile, setIsMovingFile] = useState(false);
    const [isArchivingArtifact, setIsArchivingArtifact] = useState(false);
    const [isDeletingArtifact, setIsDeletingArtifact] = useState(false);
    const [artifactComparison, setArtifactComparison] = useState<ArtifactComparison | null>(null);
    const [sqliteStatus, setSQLiteStatus] = useState<SQLiteMetadataStatus | null>(null);
    const [isPreparingMetadataStore, setIsPreparingMetadataStore] = useState(false);
    const [editingFilePaths, setEditingFilePaths] = useState<string[]>([]);
    const [fileDrafts, setFileDrafts] = useState<Record<string, string>>({});
    const [writeProposals, setWriteProposals] = useState<Record<string, FileWriteProposal>>({});
    const [navigatorWidth, setNavigatorWidth] = useState(280);
    const [isQuickOpenOpen, setIsQuickOpenOpen] = useState(false);
    const [quickOpenQuery, setQuickOpenQuery] = useState('');
    const [isCommandPaletteOpen, setIsCommandPaletteOpen] = useState(false);
    const [commandPaletteQuery, setCommandPaletteQuery] = useState('');
    const [approvalPrompt, setApprovalPrompt] = useState<ApprovalPrompt | null>(null);
    const [agentTools, setAgentTools] = useState<AgentToolDescriptor[]>([]);
    const [agentToolPlan, setAgentToolPlan] = useState<AgentToolPlanItem[]>([]);
    const [agentToolRuns, setAgentToolRuns] = useState<AgentToolRunRecord[]>([]);
    const [isRunningAgentTool, setIsRunningAgentTool] = useState(false);
    const approvalResolverRef = useRef<((approved: boolean) => void) | null>(null);
    const fileDraft = activeFile ? fileDrafts[activeFile] ?? '' : '';
    const writeProposal = activeFile ? writeProposals[activeFile] ?? null : null;
    const isEditingFile = Boolean(activeFile && editingFilePaths.includes(activeFile));
    const dirtyTabPaths = dirtyDraftPaths(fileDrafts, openTabs);

    useEffect(() => {
        setSettingsDraft(llmSettings);
        setSettingsStatus(llmSettings.updatedAt ? 'LLM settings loaded from local config.' : 'LLM provider not connected yet.');
    }, [llmSettings]);

    useEffect(() => {
        void refreshAgentTools();
    }, []);

    useEffect(() => {
        setAgentToolPlan(buildAgentToolPlan(agentTools, filePreview, artifactMetadata, activeFile));
    }, [activeFile, agentTools, artifactMetadata, filePreview]);

    useEffect(() => {
        if (!workspace || contextPackPaths.length === 0) {
            setContextPackPreview(null);
            return;
        }

        let cancelled = false;
        async function previewContextPack() {
            try {
                const preview = await PreviewChatContextPack(contextPackPaths);
                if (!cancelled) {
                    setContextPackPreview(preview);
                }
            } catch (error) {
                if (!cancelled) {
                    const message = error instanceof Error ? error.message : 'Context pack preview is unavailable.';
                    setContextPackPreview({
                        roots: contextPackPaths,
                        files: [],
                        fileCount: 0,
                        truncated: false,
                        message,
                    });
                }
            }
        }

        void previewContextPack();
        return () => {
            cancelled = true;
        };
    }, [contextPackPaths, workspace]);

    useEffect(() => {
        const columns = currentDatasetColumns(filePreview, activeDatasetProfile);
        if (columns.length === 0) {
            setDatasetChartCategory('');
            setDatasetChartValue('');
            setDatasetChartPreview(null);
            return;
        }

        setDatasetChartCategory((current) => columns.includes(current) ? current : columns[0]);
        setDatasetChartValue((current) => {
            if (current === '' || columns.includes(current)) {
                return current;
            }
            return '';
        });
        setDatasetChartPreview(null);
    }, [activeDatasetProfile, filePreview]);

    useEffect(() => {
        setArtifactMetadata(null);
        if (!filePreview?.relPath || !isArtifactPath(filePreview.relPath)) {
            return;
        }

        let cancelled = false;
        const relPath = filePreview.relPath;
        async function loadMetadata() {
            try {
                const metadata = await GetArtifactMetadata(relPath);
                if (!cancelled) {
                    setArtifactMetadata(metadata);
                }
            } catch {
                if (!cancelled) {
                    setArtifactMetadata(null);
                }
            }
        }

        void loadMetadata();
        return () => {
            cancelled = true;
        };
    }, [filePreview?.relPath]);

    useEffect(() => {
        setSavedDatasetQueries([]);
        if (!filePreview?.table) {
            return;
        }
        void refreshSavedDatasetQueries(filePreview.relPath);
    }, [filePreview?.relPath, filePreview?.table]);

    useEffect(() => {
        function handleGlobalKeyDown(event: KeyboardEvent) {
            const target = event.target as HTMLElement | null;
            const isTyping = target?.tagName === 'INPUT' || target?.tagName === 'TEXTAREA' || target?.isContentEditable;
            const isCommandPaletteShortcut = (event.ctrlKey || event.metaKey) && event.shiftKey && event.key.toLowerCase() === 'p';
            const isQuickOpenShortcut = (event.ctrlKey || event.metaKey) && !event.shiftKey && event.key.toLowerCase() === 'p';

            if (isCommandPaletteShortcut) {
                event.preventDefault();
                setIsCommandPaletteOpen(true);
                setIsQuickOpenOpen(false);
                setCommandPaletteQuery('');
                return;
            }

            if (isQuickOpenShortcut) {
                event.preventDefault();
                setIsCommandPaletteOpen(false);
                setIsQuickOpenOpen(true);
                setQuickOpenQuery('');
                return;
            }

            if (event.key === 'Escape' && isCommandPaletteOpen) {
                event.preventDefault();
                setIsCommandPaletteOpen(false);
                return;
            }

            if (event.key === 'Escape' && isQuickOpenOpen) {
                event.preventDefault();
                setIsQuickOpenOpen(false);
                return;
            }

            if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's') {
                event.preventDefault();
                void saveActiveDraftShortcut();
                return;
            }

            if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'n') {
                event.preventDefault();
                startNewFileDraft();
                return;
            }

            if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'w') {
                if (openTabs.some((tab) => tab.relPath === activeFile)) {
                    event.preventDefault();
                    closeOpenTab(activeFile);
                }
                return;
            }

            if ((event.ctrlKey || event.metaKey) && event.key === 'Tab') {
                if (openTabs.length > 1) {
                    event.preventDefault();
                    selectAdjacentTab(event.shiftKey ? -1 : 1);
                }
                return;
            }

            if (isTyping) {
                return;
            }

            if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
                event.preventDefault();
                setIsCommandPaletteOpen(false);
                setIsQuickOpenOpen(true);
                setQuickOpenQuery('');
            }
        }

        window.addEventListener('keydown', handleGlobalKeyDown);
        return () => window.removeEventListener('keydown', handleGlobalKeyDown);
    }, [activeFile, fileDraft, isApplyingWrite, isCommandPaletteOpen, isEditingFile, isPreviewingWrite, isQuickOpenOpen, openTabs, writeProposal]);

    const selectedMeta = useMemo(() => {
        if (workspace) {
            if (filePreview?.relPath === activeFile) {
                return previewMeta(filePreview);
            }

            return workspace.nodes.find((node) => node.relPath === activeFile)?.meta ?? workspace.root;
        }

        return state.workspaceItems.find((item) => activeFile.startsWith(item.name))?.meta ?? 'Selected planning source';
    }, [activeFile, filePreview, state.workspaceItems, workspace]);

    const workspaceNodes = useMemo(() => {
        if (!workspace) {
            return [];
        }

        return workspace.nodes.filter((node) => isWorkspaceNodeVisible(node, expandedDirectories));
    }, [expandedDirectories, workspace]);

    const canSaveLatestAssistantArtifact = useMemo(() => {
        return Boolean(latestAssistantMessage(chatMessages)) && !isSendingPrompt;
    }, [chatMessages, isSendingPrompt]);

    const commandActions = buildCommandActions();

    function pushToolEvent(title: string, detail: string) {
        setLocalToolEvents((current) => [
            {time: new Date().toLocaleTimeString(), title, detail},
            ...current,
        ].slice(0, 12));
    }

    function buildCommandActions(): CommandAction[] {
        const selectedContextRelPath = selectedTextContextRelPath();
        const hasWorkspace = Boolean(workspace);
        const hasActivePreview = Boolean(workspace && filePreview);
        const hasDirtyActiveDraft = Boolean(isEditingFile && filePreview && dirtyTabPaths.includes(filePreview.relPath));
        const canEditSelectedFile = Boolean(workspace && filePreview?.kind === 'file' && !filePreview.table);
        const canDeleteSelectedFile = Boolean(workspace && filePreview?.kind === 'file' && !isEditingFile);
        const canMoveSelectedFile = Boolean(workspace && filePreview?.kind === 'file' && !isEditingFile);
        const canUseSelectedContext = Boolean(selectedContextRelPath) && !isSendingPrompt;
        const canUseSelectedDataset = Boolean(workspace && filePreview?.fileType === 'data');
        const canChartSelectedDataset = Boolean(workspace && filePreview?.table && datasetChartCategory);
        const canExportDatasetQuery = Boolean(workspace && filePreview?.table && datasetQueryResult);

        return [
            {
                detail: 'Choose a project folder and index its safe workspace tree.',
                group: 'Workspace',
                id: 'workspace.open',
                run: () => void openWorkspace(),
                title: 'Open Folder',
            },
            {
                detail: hasWorkspace ? 'Rescan the current workspace and preserve the selected file when possible.' : 'Open a workspace before refreshing.',
                disabled: !hasWorkspace || isRefreshingWorkspace,
                group: 'Workspace',
                id: 'workspace.refresh',
                run: () => void refreshWorkspace(),
                title: 'Refresh Workspace',
            },
            {
                detail: 'Find and open workspace files, folders, datasets, artifacts, or loaded tabs.',
                disabled: !hasWorkspace,
                group: 'Workspace',
                id: 'workspace.quick-open',
                run: () => {
                    setQuickOpenQuery('');
                    setIsQuickOpenOpen(true);
                },
                shortcut: 'Ctrl+P',
                title: 'Quick Open',
            },
            {
                detail: workspaceSearchQuery.trim() ? `Search workspace paths and previewable text for "${workspaceSearchQuery.trim()}".` : 'Enter a query in the workspace search field first.',
                disabled: !hasWorkspace || !workspaceSearchQuery.trim() || isSearchingWorkspace,
                group: 'Workspace',
                id: 'workspace.search',
                run: () => void searchWorkspace(),
                title: 'Search Workspace',
            },
            {
                detail: hasWorkspace ? 'Expand every indexed directory in the project tree.' : 'Open a workspace before expanding folders.',
                disabled: !hasWorkspace,
                group: 'Workspace',
                id: 'workspace.expand-all',
                run: expandAllDirectories,
                title: 'Expand All Folders',
            },
            {
                detail: hasWorkspace ? 'Collapse the project tree back to the root view.' : 'Open a workspace before collapsing folders.',
                disabled: !hasWorkspace,
                group: 'Workspace',
                id: 'workspace.collapse-all',
                run: collapseAllDirectories,
                title: 'Collapse All Folders',
            },
            {
                detail: hasActivePreview ? `Reload ${activeFile} from disk.` : 'Select a workspace file before reloading its preview.',
                disabled: !hasActivePreview || isLoadingPreview,
                group: 'Editor',
                id: 'editor.reload-preview',
                run: () => void refreshSelectedPreview(),
                title: 'Reload Current Preview',
            },
            {
                detail: hasWorkspace ? 'Create a workspace-relative text/code file through the draft and diff flow.' : 'Open a workspace before creating files.',
                disabled: !hasWorkspace,
                group: 'Editor',
                id: 'editor.new-file',
                run: startNewFileDraft,
                shortcut: 'Ctrl+N',
                title: 'New File',
            },
            {
                detail: openTabs.some((tab) => tab.relPath === activeFile) ? `Close ${activeFile}.` : 'Open a file tab before closing the active tab.',
                disabled: !openTabs.some((tab) => tab.relPath === activeFile),
                group: 'Editor',
                id: 'editor.close-tab',
                run: () => closeOpenTab(activeFile),
                shortcut: 'Ctrl+W',
                title: 'Close Active Tab',
            },
            {
                detail: openTabs.length > 1 ? 'Move to the next open editor tab.' : 'Open at least two tabs before cycling tabs.',
                disabled: openTabs.length < 2,
                group: 'Editor',
                id: 'editor.next-tab',
                run: () => selectAdjacentTab(1),
                shortcut: 'Ctrl+Tab',
                title: 'Next Editor Tab',
            },
            {
                detail: openTabs.length > 1 ? 'Move to the previous open editor tab.' : 'Open at least two tabs before cycling tabs.',
                disabled: openTabs.length < 2,
                group: 'Editor',
                id: 'editor.previous-tab',
                run: () => selectAdjacentTab(-1),
                shortcut: 'Ctrl+Shift+Tab',
                title: 'Previous Editor Tab',
            },
            {
                detail: canEditSelectedFile ? `Start a safe edit draft for ${activeFile}.` : 'Select a text/code file before editing.',
                disabled: !canEditSelectedFile,
                group: 'Editor',
                id: 'editor.start-edit',
                run: startFileEdit,
                title: 'Edit Current File',
            },
            {
                detail: canDeleteSelectedFile ? `Delete ${activeFile} after backend preview and confirmation.` : 'Select a saved file and close any active draft before deleting.',
                disabled: !canDeleteSelectedFile || isDeletingFile,
                group: 'Editor',
                id: 'editor.delete-file',
                run: () => void deleteActiveFile(),
                title: 'Delete Active File',
            },
            {
                detail: canMoveSelectedFile ? `Rename or move ${activeFile} without overwriting an existing file.` : 'Select a saved file and close any active draft before renaming.',
                disabled: !canMoveSelectedFile || isMovingFile,
                group: 'Editor',
                id: 'editor.move-file',
                run: () => void moveActiveFile(),
                title: 'Rename Or Move Active File',
            },
            {
                detail: hasDirtyActiveDraft ? 'Preview or apply the active file draft through the write safety flow.' : 'Change the active edit draft before saving.',
                disabled: !hasDirtyActiveDraft || isPreviewingWrite || isApplyingWrite,
                group: 'Editor',
                id: 'editor.save-draft',
                run: () => void saveActiveDraftShortcut(),
                shortcut: 'Ctrl+S',
                title: 'Preview Or Apply Active Draft',
            },
            {
                detail: selectedContextRelPath ? `Pin ${selectedContextRelPath} into the chat context pack.` : 'Select a file, extracted document, dataset, directory, or workspace context first.',
                disabled: !selectedContextRelPath,
                group: 'AI Assistant',
                id: 'context.pin-selected',
                run: pinSelectedContext,
                title: 'Pin Current Context',
            },
            {
                detail: hasWorkspace ? 'Pin the workspace root so chat can stream a bounded project context.' : 'Open a workspace before pinning project context.',
                disabled: !hasWorkspace,
                group: 'AI Assistant',
                id: 'context.pin-project',
                run: pinProjectContext,
                title: 'Pin Project Context',
            },
            {
                detail: contextPackPaths.length > 0 ? 'Remove every pinned context item from the chat pack.' : 'Pin context before clearing the pack.',
                disabled: contextPackPaths.length === 0,
                group: 'AI Assistant',
                id: 'context.clear-pack',
                run: () => {
                    setContextPackPaths([]);
                    setChatStatus('Context pack cleared.');
                    pushToolEvent('Context pack cleared', 'Pinned chat context reset.');
                },
                title: 'Clear Context Pack',
            },
            {
                detail: selectedContextRelPath ? `Ask the model to explain ${selectedContextRelPath}.` : 'Select explainable context first.',
                disabled: !canUseSelectedContext,
                group: 'AI Assistant',
                id: 'chat.explain-context',
                run: () => void explainSelectedContext(),
                title: 'Explain Current Context',
            },
            {
                detail: selectedContextRelPath ? `Summarize ${selectedContextRelPath} and save the result as Markdown.` : 'Select summarizable context first.',
                disabled: !canUseSelectedContext || isSummarizingContext,
                group: 'AI Assistant',
                id: 'chat.summarize-context',
                run: () => void summarizeSelectedContext(),
                title: 'Summarize Current Context',
            },
            {
                detail: canSaveLatestAssistantArtifact ? 'Save the latest assistant answer as a Markdown artifact.' : 'Generate an assistant answer before saving it.',
                disabled: !hasWorkspace || !canSaveLatestAssistantArtifact || isSavingChatArtifact,
                group: 'Artifacts',
                id: 'artifacts.save-answer',
                run: () => void saveLatestAssistantArtifact(),
                title: 'Save Latest Assistant Answer',
            },
            {
                detail: hasWorkspace ? 'Create a timestamped Markdown report from the selected preview or workspace.' : 'Open a workspace before creating reports.',
                disabled: !hasWorkspace || isCreatingReport,
                group: 'Artifacts',
                id: 'artifacts.create-report',
                run: () => void createMarkdownReport(),
                title: 'Create Markdown Report',
            },
            {
                detail: canUseSelectedDataset ? `Profile ${activeFile} and persist dataset metadata.` : 'Select a CSV or workbook dataset first.',
                disabled: !canUseSelectedDataset || isProfilingDataset,
                group: 'Data Studio',
                id: 'data.profile',
                run: () => void profileSelectedDataset(),
                title: 'Profile Current Dataset',
            },
            {
                detail: canUseSelectedDataset ? 'Run the bounded dataset query/filter currently in the data panel.' : 'Select a CSV dataset before querying.',
                disabled: !canUseSelectedDataset || isQueryingDataset,
                group: 'Data Studio',
                id: 'data.query',
                run: () => void querySelectedDataset(),
                title: 'Query Current Dataset',
            },
            {
                detail: canChartSelectedDataset ? `Create a ${datasetChartType} chart from ${activeFile}.` : 'Select a CSV dataset and chart category first.',
                disabled: !canChartSelectedDataset || isCreatingDatasetChart,
                group: 'Data Studio',
                id: 'data.create-chart',
                run: () => void createDatasetChart(),
                title: 'Create Dataset Chart',
            },
            {
                detail: canExportDatasetQuery ? `Export the current ${activeFile} query result as a CSV artifact.` : 'Run a CSV dataset query before exporting the result.',
                disabled: !canExportDatasetQuery || isExportingDatasetQuery,
                group: 'Data Studio',
                id: 'data.export-query',
                run: () => void exportDatasetQuery(),
                title: 'Export Dataset Query',
            },
            {
                detail: chatMessages.length > 0 ? 'Clear persisted chat messages for the active workspace.' : 'No chat messages are loaded.',
                disabled: chatMessages.length === 0,
                group: 'AI Assistant',
                id: 'chat.clear-history',
                run: () => void clearChatHistory(),
                title: 'Clear Chat History',
            },
            {
                detail: recentWorkspaces.length > 0 ? 'Remove every recent workspace entry from local config.' : 'No recent workspace entries are stored.',
                disabled: recentWorkspaces.length === 0 || isManagingRecent,
                group: 'Workspace',
                id: 'workspace.clear-recent',
                run: () => void clearRecentWorkspaces(),
                title: 'Clear Recent Workspaces',
            },
        ];
    }

    function selectFallbackItem(name: string) {
        setActiveFile(`${name}/`);
        setFilePreview(null);
    }

    async function selectWorkspaceNode(node: FileNode) {
        if (node.kind === 'directory') {
            toggleDirectory(node.relPath);
        }
        await previewWorkspaceNode(node, true);
    }

    function toggleDirectory(relPath: string) {
        setExpandedDirectories((current) => {
            const next = new Set(current);
            if (next.has(relPath)) {
                next.delete(relPath);
            } else {
                next.add(relPath);
            }
            return next;
        });
    }

    async function previewWorkspaceNode(node: FileNode, updateActiveFile: boolean) {
        if (updateActiveFile) {
            setActiveFile(node.relPath);
        }
        setDatasetQueryResult(null);

        if (node.kind === 'directory') {
            setIsLoadingPreview(false);
            const directoryPreview = createDirectoryPreview(node);
            setFilePreview(directoryPreview);
            return;
        }

        setFilePreview(null);
        setIsLoadingPreview(true);
        try {
            const preview = await ReadWorkspaceFile(node.relPath);
            setFilePreview(preview);
            upsertOpenTab(preview);
            setActiveDatasetProfile(datasetProfiles.find((profile) => profile.relPath === node.relPath) ?? null);
            pushToolEvent('Preview loaded', node.relPath);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            if (message.includes('undefined') || message.includes('window')) {
                const fallbackPreview: FilePreview = {
                    relPath: node.relPath,
                    name: node.name,
                    kind: 'unsupported',
                    fileType: node.fileType,
                    content: '',
                    text: '',
                    encoding: '',
                    truncated: false,
                    message: 'File previews are available in the desktop runtime.',
                    size: 0,
                };
                setFilePreview(fallbackPreview);
                upsertOpenTab(fallbackPreview);
                return;
            }
            const failedPreview: FilePreview = {
                relPath: node.relPath,
                name: node.name,
                kind: 'unsupported',
                fileType: node.fileType,
                content: '',
                text: '',
                encoding: '',
                truncated: false,
                message: message || 'Could not preview this file.',
                size: 0,
            };
            setFilePreview(failedPreview);
            upsertOpenTab(failedPreview);
        } finally {
            setIsLoadingPreview(false);
        }
    }

    function createDirectoryPreview(node: FileNode): FilePreview {
        return {
            relPath: node.relPath,
            name: node.name,
            kind: 'directory',
            fileType: node.fileType,
            content: '',
            text: '',
            encoding: '',
            truncated: false,
            message: 'Select a file inside this folder to preview its contents.',
            size: 0,
        };
    }

    function createNewFilePreview(relPath: string): FilePreview {
        return {
            relPath,
            name: fileNameFromRelPath(relPath),
            kind: 'file',
            fileType: fileTypeForRelPath(relPath),
            content: '',
            text: '',
            encoding: 'utf-8',
            truncated: false,
            message: 'New file draft. Preview the diff, then apply to create it in the workspace.',
            size: 0,
        };
    }

    function startFileEdit() {
        if (!filePreview || filePreview.kind !== 'file') {
            setWorkspaceStatus('Select a text file before editing.');
            return;
        }
        setEditingFilePaths((current) => current.includes(filePreview.relPath) ? current : [...current, filePreview.relPath]);
        setFileDrafts((current) => ({
            ...current,
            [filePreview.relPath]: current[filePreview.relPath] ?? filePreview.content,
        }));
        setWriteProposals((current) => omitKey(current, filePreview.relPath));
    }

    function startNewFileDraft() {
        if (!workspace) {
            setWorkspaceStatus('Open a workspace before creating files.');
            return;
        }

        const rawRelPath = window.prompt('New file path inside the workspace', suggestedNewFilePath(filePreview, activeFile));
        if (rawRelPath === null) {
            setWorkspaceStatus('New file creation cancelled.');
            return;
        }

        const relPath = normalizeNewFileRelPath(rawRelPath);
        if (!relPath) {
            setWorkspaceStatus('Enter a workspace-relative file path.');
            return;
        }
        if (relPath.includes('../') || relPath === '..' || relPath.startsWith('/')) {
            setWorkspaceStatus('New file path must stay inside the workspace.');
            return;
        }
        if (relPath.endsWith('/')) {
            setWorkspaceStatus('New file path must include a file name.');
            return;
        }
        if (relPath.toLowerCase().startsWith('.nexusdesk/')) {
            setWorkspaceStatus('Direct writes to NexusDesk metadata are not allowed.');
            return;
        }

        const existingNode = workspace.nodes.find((node) => node.relPath === relPath);
        if (existingNode) {
            void selectWorkspaceNode(existingNode);
            setWorkspaceStatus(`${relPath} already exists. Opened existing file instead.`);
            return;
        }

        const draft = defaultNewFileContent(relPath);
        const preview = createNewFilePreview(relPath);
        setActiveFile(relPath);
        setFilePreview(preview);
        upsertOpenTab(preview);
        setEditingFilePaths((current) => current.includes(relPath) ? current : [...current, relPath]);
        setFileDrafts((current) => ({...current, [relPath]: draft}));
        setWriteProposals((current) => omitKey(current, relPath));
        setActiveDatasetProfile(null);
        setWorkspaceStatus(`${relPath} draft ready. Preview diff before applying the create.`);
        pushToolEvent('New file draft', relPath);
    }

    function clearFileWriteDraft() {
        if (activeFile) {
            setEditingFilePaths((current) => current.filter((relPath) => relPath !== activeFile));
            setFileDrafts((current) => omitKey(current, activeFile));
            setWriteProposals((current) => omitKey(current, activeFile));
        }
        setIsPreviewingWrite(false);
        setIsApplyingWrite(false);
    }

    function requestApproval(prompt: ApprovalPrompt) {
        setApprovalPrompt(prompt);
        return new Promise<boolean>((resolve) => {
            approvalResolverRef.current = resolve;
        });
    }

    function resolveApproval(approved: boolean) {
        approvalResolverRef.current?.(approved);
        approvalResolverRef.current = null;
        setApprovalPrompt(null);
    }

    function updateFileDraft(content: string) {
        if (!activeFile) {
            return;
        }
        setFileDrafts((current) => ({
            ...current,
            [activeFile]: content,
        }));
        setWriteProposals((current) => omitKey(current, activeFile));
    }

    async function previewFileWrite() {
        if (!workspace || !filePreview) {
            setWorkspaceStatus('Open a workspace and select a file before previewing writes.');
            return;
        }

        setIsPreviewingWrite(true);
        try {
            const proposal = await PreviewFileWrite({relPath: filePreview.relPath, content: fileDraft});
            setWriteProposals((current) => ({
                ...current,
                [proposal.relPath]: proposal,
            }));
            setWorkspaceStatus(proposal.message);
            pushToolEvent('Write preview', proposal.relPath);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setWorkspaceStatus(message || 'Could not preview file write.');
        } finally {
            setIsPreviewingWrite(false);
        }
    }

    async function applyFileWrite() {
        if (!workspace || !filePreview || !writeProposal) {
            setWorkspaceStatus('Preview the file write before applying it.');
            return;
        }

        const approved = await requestApproval({
            action: 'Apply file write',
            confirmLabel: 'Apply write',
            message: writeProposal.message,
            risk: writeProposal.action === 'create' ? 'medium' : 'high',
            target: writeProposal.relPath,
        });
        if (!approved) {
            setWorkspaceStatus(`Write cancelled for ${writeProposal.relPath}.`);
            return;
        }

        setIsApplyingWrite(true);
        try {
            const proposal = await ApplyFileWrite({relPath: filePreview.relPath, content: fileDraft});
            const result = await RefreshWorkspace();
            if (result.selected) {
                onWorkspaceChange(result.snapshot);
                await selectWorkspaceFile(result.snapshot, proposal.relPath);
                await refreshApprovals();
            }
            setEditingFilePaths((current) => current.filter((relPath) => relPath !== proposal.relPath));
            setFileDrafts((current) => omitKey(current, proposal.relPath));
            setWriteProposals((current) => omitKey(current, proposal.relPath));
            setWorkspaceStatus(proposal.message);
            pushToolEvent('File write applied', proposal.relPath);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setWorkspaceStatus(message || 'Could not apply file write.');
        } finally {
            setIsApplyingWrite(false);
        }
    }

    async function deleteActiveFile() {
        if (!workspace || !filePreview || filePreview.kind !== 'file') {
            setWorkspaceStatus('Select a workspace file before deleting.');
            return;
        }
        if (isEditingFile) {
            setWorkspaceStatus('Close or cancel the active edit draft before deleting this file.');
            return;
        }

        const relPath = filePreview.relPath;
        setIsDeletingFile(true);
        setWorkspaceStatus(`Preparing delete preview for ${relPath}...`);
        try {
            const proposal = await PreviewFileDelete(relPath);
            const sizeLabel = proposal.size > 0 ? ` (${formatBytes(Number(proposal.size))})` : '';
            const approved = await requestApproval({
                action: 'Delete file',
                confirmLabel: 'Delete',
                message: `Delete ${proposal.relPath}${sizeLabel}. This removes the file from the workspace.`,
                risk: 'high',
                target: proposal.relPath,
            });
            if (!approved) {
                setWorkspaceStatus(`Delete cancelled for ${proposal.relPath}.`);
                return;
            }

            const deleted = await ApplyFileDelete(relPath);
            const result = await RefreshWorkspace();
            if (result.selected) {
                const selectedNode = selectNodeAfterWorkspaceUpdate(result.snapshot);
                onWorkspaceChange(result.snapshot);
                removeOpenTabState(deleted.relPath);
                await refreshArtifacts();
                await refreshApprovals();
                await refreshDatasetProfiles();
                setExpandedDirectories((current) => reconcileExpandedDirectories(current, result.snapshot, selectedNode));
                if (selectedNode) {
                    setActiveFile(selectedNode.relPath);
                    await previewWorkspaceNode(selectedNode, false);
                } else {
                    setActiveFile(result.snapshot.name);
                    setFilePreview(null);
                    setActiveDatasetProfile(null);
                }
            }
            setWorkspaceStatus(deleted.message);
            pushToolEvent('File deleted', deleted.relPath);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setWorkspaceStatus(message || 'Could not delete this file.');
        } finally {
            setIsDeletingFile(false);
        }
    }

    async function moveActiveFile() {
        if (!workspace || !filePreview || filePreview.kind !== 'file') {
            setWorkspaceStatus('Select a workspace file before renaming or moving.');
            return;
        }
        if (isEditingFile) {
            setWorkspaceStatus('Close or cancel the active edit draft before renaming this file.');
            return;
        }

        const sourceRelPath = filePreview.relPath;
        const rawTarget = window.prompt('Rename or move to workspace-relative path', sourceRelPath);
        if (rawTarget === null) {
            setWorkspaceStatus(`Rename cancelled for ${sourceRelPath}.`);
            return;
        }

        const targetRelPath = normalizeNewFileRelPath(rawTarget);
        if (!targetRelPath) {
            setWorkspaceStatus('Enter a workspace-relative target path.');
            return;
        }
        if (targetRelPath === sourceRelPath) {
            setWorkspaceStatus('Rename target is the same as the current file.');
            return;
        }
        if (targetRelPath.includes('../') || targetRelPath === '..' || targetRelPath.startsWith('/')) {
            setWorkspaceStatus('Rename target must stay inside the workspace.');
            return;
        }
        if (targetRelPath.toLowerCase().startsWith('.nexusdesk/')) {
            setWorkspaceStatus('Direct moves into NexusDesk metadata are not allowed.');
            return;
        }

        setIsMovingFile(true);
        setWorkspaceStatus(`Preparing move preview for ${sourceRelPath}...`);
        try {
            const proposal = await PreviewFileMove({sourceRelPath, targetRelPath});
            const sizeLabel = proposal.size > 0 ? ` (${formatBytes(Number(proposal.size))})` : '';
            const approved = await requestApproval({
                action: 'Rename or move file',
                confirmLabel: 'Move file',
                message: `Move ${proposal.sourceRelPath} to ${proposal.targetRelPath}${sizeLabel}.`,
                risk: 'high',
                target: proposal.targetRelPath,
            });
            if (!approved) {
                setWorkspaceStatus(`Rename cancelled for ${proposal.sourceRelPath}.`);
                return;
            }

            const moved = await ApplyFileMove({sourceRelPath, targetRelPath});
            const result = await RefreshWorkspace();
            if (result.selected) {
                onWorkspaceChange(result.snapshot);
                removeOpenTabState(moved.sourceRelPath);
                await refreshArtifacts();
                await refreshApprovals();
                await refreshDatasetProfiles();
                const selectedNode = findWorkspaceNode(result.snapshot, moved.targetRelPath) ?? selectNodeAfterWorkspaceUpdate(result.snapshot);
                setExpandedDirectories((current) => reconcileExpandedDirectories(current, result.snapshot, selectedNode));
                if (selectedNode) {
                    setActiveFile(selectedNode.relPath);
                    await previewWorkspaceNode(selectedNode, false);
                } else {
                    setActiveFile(result.snapshot.name);
                    setFilePreview(null);
                    setActiveDatasetProfile(null);
                }
            }
            setWorkspaceStatus(moved.message);
            pushToolEvent('File moved', `${moved.sourceRelPath} -> ${moved.targetRelPath}`);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setWorkspaceStatus(message || 'Could not rename or move this file.');
        } finally {
            setIsMovingFile(false);
        }
    }

    async function openWorkspace() {
        setIsOpeningWorkspace(true);
        setWorkspaceStatus('Waiting for folder selection...');

        try {
            const result = await SelectWorkspace();
            if (!(await applyWorkspaceResult(result, 'indexed'))) {
                setWorkspaceStatus('Workspace selection cancelled.');
                return;
            }
            await refreshRecentWorkspaces();
            pushToolEvent('Workspace opened', result.snapshot.name);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            if (message.includes('undefined') || message.includes('window')) {
                setWorkspaceStatus('Workspace picker is available in the desktop runtime.');
                return;
            }
            setWorkspaceStatus(message || 'Workspace picker is available in the desktop runtime.');
        } finally {
            setIsOpeningWorkspace(false);
        }
    }

    async function reopenWorkspace(recentWorkspace: RecentWorkspace) {
        setIsOpeningWorkspace(true);
        setWorkspaceStatus(`Opening ${recentWorkspace.name}...`);

        try {
            const result = await OpenWorkspace(recentWorkspace.path);
            if (await applyWorkspaceResult(result, 'indexed')) {
                await refreshRecentWorkspaces();
                pushToolEvent('Workspace reopened', result.snapshot.name);
            }
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            if (message.includes('undefined') || message.includes('window')) {
                setWorkspaceStatus('Recent workspaces are available in the desktop runtime.');
                return;
            }
            setWorkspaceStatus(message || `Could not open ${recentWorkspace.name}.`);
        } finally {
            setIsOpeningWorkspace(false);
        }
    }

    async function refreshWorkspace() {
        if (!workspace) {
            setWorkspaceStatus('Open a workspace before refreshing.');
            return;
        }

        setIsRefreshingWorkspace(true);
        setWorkspaceStatus(`Refreshing ${workspace.name}...`);

        try {
            const result = await RefreshWorkspace();
            if (!(await applyWorkspaceResult(result, 'refreshed'))) {
                setWorkspaceStatus('Open a workspace before refreshing.');
                return;
            }
            pushToolEvent('Workspace refreshed', result.snapshot.name);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            if (message.includes('undefined') || message.includes('window')) {
                setWorkspaceStatus('Workspace refresh is available in the desktop runtime.');
                return;
            }
            setWorkspaceStatus(message || 'Workspace refresh failed.');
        } finally {
            setIsRefreshingWorkspace(false);
        }
    }

    async function applyWorkspaceResult(result: WorkspaceOpenResult, verb: 'indexed' | 'refreshed') {
        if (!result.selected) {
            return false;
        }

        const rootChanged = workspace?.root !== result.snapshot.root;
        const selectedNode = selectNodeAfterWorkspaceUpdate(result.snapshot);

        onWorkspaceChange(result.snapshot);
        setWorkspaceSearchResults([]);
        if (rootChanged) {
            setOpenTabs([]);
            setEditingFilePaths([]);
            setFileDrafts({});
            setWriteProposals({});
        }
        await refreshChatHistory();
        await refreshArtifacts();
        await refreshApprovals();
        await refreshDatasetProfiles();
        await refreshAgentToolRuns();
        setExpandedDirectories((current) => reconcileExpandedDirectories(current, result.snapshot, selectedNode));
        if (selectedNode) {
            setActiveFile(selectedNode.relPath);
            await previewWorkspaceNode(selectedNode, false);
        } else {
            setActiveFile(result.snapshot.name);
            setFilePreview(null);
        }
        setWorkspaceStatus(`${result.snapshot.nodes.length} items ${verb} from ${result.snapshot.name}.`);
        return true;
    }

    function selectNodeAfterWorkspaceUpdate(snapshot: WorkspaceSnapshot) {
        const previousSelection = snapshot.nodes.find((node) => node.relPath === activeFile);
        if (previousSelection) {
            return previousSelection;
        }

        return snapshot.nodes.find((node) => node.kind === 'file') ?? snapshot.nodes[0] ?? null;
    }

    async function selectWorkspaceFile(snapshot: WorkspaceSnapshot, relPath: string) {
        const node = findWorkspaceNode(snapshot, relPath);
        if (!node) {
            setActiveFile(relPath);
            setFilePreview(null);
            return;
        }

        setActiveFile(node.relPath);
        await previewWorkspaceNode(node, false);
    }

    function upsertOpenTab(preview: FilePreview) {
        if (!preview.relPath || preview.kind === 'directory') {
            return;
        }

        setOpenTabs((current) => {
            const existing = current.findIndex((tab) => tab.relPath === preview.relPath);
            if (existing === -1) {
                return [...current, preview].slice(-8);
            }
            return current.map((tab, index) => index === existing ? preview : tab);
        });
    }

    function selectOpenTab(relPath: string) {
        const tab = openTabs.find((current) => current.relPath === relPath);
        if (!tab) {
            return;
        }
        setActiveFile(tab.relPath);
        setFilePreview(tab);
        setActiveDatasetProfile(datasetProfiles.find((profile) => profile.relPath === tab.relPath) ?? null);
    }

    function selectAdjacentTab(direction: 1 | -1) {
        if (openTabs.length === 0) {
            return;
        }

        const currentIndex = Math.max(openTabs.findIndex((tab) => tab.relPath === activeFile), 0);
        const nextIndex = (currentIndex + direction + openTabs.length) % openTabs.length;
        selectOpenTab(openTabs[nextIndex].relPath);
    }

    function removeOpenTabState(relPath: string) {
        setOpenTabs((current) => current.filter((tab) => tab.relPath !== relPath));
        setEditingFilePaths((current) => current.filter((path) => path !== relPath));
        setFileDrafts((current) => omitKey(current, relPath));
        setWriteProposals((current) => omitKey(current, relPath));
    }

    function closeOpenTab(relPath: string) {
        const tabIndex = openTabs.findIndex((tab) => tab.relPath === relPath);
        if (tabIndex === -1) {
            return;
        }
        if (dirtyTabPaths.includes(relPath) && !window.confirm(`Discard unsaved changes in ${relPath}?`)) {
            setWorkspaceStatus(`${relPath} is still open with unsaved changes.`);
            return;
        }

        const nextTabs = openTabs.filter((tab) => tab.relPath !== relPath);
        removeOpenTabState(relPath);
        if (activeFile !== relPath) {
            return;
        }

        const nextTab = nextTabs[Math.max(0, tabIndex - 1)] ?? nextTabs[0] ?? null;
        if (!nextTab) {
            setFilePreview(null);
            setActiveFile(workspace?.name ?? '');
            setActiveDatasetProfile(null);
            return;
        }

        setActiveFile(nextTab.relPath);
        setFilePreview(nextTab);
        setActiveDatasetProfile(datasetProfiles.find((profile) => profile.relPath === nextTab.relPath) ?? null);
    }

    async function saveActiveDraftShortcut() {
        if (!isEditingFile || !filePreview || !dirtyTabPaths.includes(filePreview.relPath)) {
            setWorkspaceStatus('No active edit draft to save.');
            return;
        }
        if (isPreviewingWrite || isApplyingWrite) {
            return;
        }
        if (writeProposal) {
            await applyFileWrite();
        } else {
            await previewFileWrite();
        }
    }

    async function refreshSelectedPreview() {
        if (!workspace) {
            setWorkspaceStatus('Open a workspace before refreshing a preview.');
            return;
        }

        const node = findWorkspaceNode(workspace, activeFile);
        if (!node) {
            setWorkspaceStatus(`${activeFile} is not available in the current workspace tree.`);
            return;
        }

        await previewWorkspaceNode(node, false);
        setWorkspaceStatus(`${node.relPath} preview reloaded.`);
    }

    async function searchWorkspace() {
        if (!workspace) {
            setWorkspaceStatus('Open a workspace before searching.');
            return;
        }
        const query = workspaceSearchQuery.trim();
        if (!query) {
            setWorkspaceSearchResults([]);
            return;
        }

        setIsSearchingWorkspace(true);
        setWorkspaceStatus(`Searching ${workspace.name}...`);
        try {
            const results = await SearchWorkspace(query);
            setWorkspaceSearchResults(results);
            setWorkspaceStatus(`${results.length} workspace matches for "${query}".`);
            pushToolEvent('Workspace search', `${results.length} matches for ${query}`);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setWorkspaceStatus(message || 'Workspace search failed.');
        } finally {
            setIsSearchingWorkspace(false);
        }
    }

    async function selectSearchResult(result: WorkspaceSearchResult) {
        if (!workspace) {
            return;
        }
        if (result.kind === 'chat') {
            setChatStatus(`Chat match: ${result.snippet}`);
            setWorkspaceStatus('Search match is in chat history.');
            return;
        }
        const node = findWorkspaceNode(workspace, result.relPath);
        if (!node) {
            setWorkspaceStatus(`${result.relPath} is not visible in the current workspace tree.`);
            return;
        }
        setExpandedDirectories((current) => {
            const next = new Set(current);
            getAncestorDirectories(node).forEach((relPath) => next.add(relPath));
            return next;
        });
        await selectWorkspaceNode(node);
    }

    async function selectQuickOpenNode(node: FileNode) {
        setExpandedDirectories((current) => {
            const next = new Set(current);
            getAncestorDirectories(node).forEach((relPath) => next.add(relPath));
            return next;
        });
        await selectWorkspaceNode(node);
        setWorkspaceStatus(`${node.relPath} opened from quick open.`);
        pushToolEvent('Quick open', node.relPath);
    }

    function expandAllDirectories() {
        if (!workspace) {
            return;
        }
        setExpandedDirectories(new Set(workspace.nodes.filter((node) => node.kind === 'directory').map((node) => node.relPath)));
    }

    function collapseAllDirectories() {
        setExpandedDirectories(new Set());
    }

    function reconcileExpandedDirectories(current: Set<string>, snapshot: WorkspaceSnapshot, selectedNode: FileNode | null) {
        const directoryPaths = new Set(snapshot.nodes.filter((node) => node.kind === 'directory').map((node) => node.relPath));
        const next = new Set<string>();

        current.forEach((relPath) => {
            if (directoryPaths.has(relPath)) {
                next.add(relPath);
            }
        });

        snapshot.nodes.forEach((node) => {
            if (node.kind === 'directory' && node.depth === 1) {
                next.add(node.relPath);
            }
        });

        getAncestorDirectories(selectedNode).forEach((relPath) => {
            if (directoryPaths.has(relPath)) {
                next.add(relPath);
            }
        });

        return next;
    }

    function getAncestorDirectories(node: FileNode | null) {
        if (!node) {
            return [];
        }

        const pathParts = node.relPath.split('/');
        const ancestorCount = node.kind === 'directory' ? pathParts.length : pathParts.length - 1;
        const ancestors: string[] = [];
        for (let index = 1; index <= ancestorCount; index += 1) {
            ancestors.push(pathParts.slice(0, index).join('/'));
        }
        return ancestors;
    }

    function isWorkspaceNodeVisible(node: FileNode, expanded: Set<string>) {
        const pathParts = node.relPath.split('/');
        for (let index = 1; index < pathParts.length; index += 1) {
            const ancestor = pathParts.slice(0, index).join('/');
            if (!expanded.has(ancestor)) {
                return false;
            }
        }
        return true;
    }

    async function refreshRecentWorkspaces() {
        try {
            onRecentWorkspacesChange(await GetRecentWorkspaces());
        } catch {
            onRecentWorkspacesChange([]);
        }
    }

    async function refreshArtifacts() {
        try {
            setArtifacts(await ListArtifacts());
        } catch {
            setArtifacts([]);
        }
    }

    async function refreshApprovals() {
        try {
            setApprovalRecords(await ListApprovals());
        } catch {
            setApprovalRecords([]);
        }
    }

    async function refreshDatasetProfiles() {
        try {
            const profiles = await ListDatasetProfiles();
            setDatasetProfiles(profiles);
            setActiveDatasetProfile((current) => profiles.find((profile) => profile.relPath === current?.relPath) ?? current);
        } catch {
            setDatasetProfiles([]);
        }
    }

    async function refreshSavedDatasetQueries(relPath: string) {
        try {
            setSavedDatasetQueries(await ListDatasetQueries(relPath));
        } catch {
            setSavedDatasetQueries([]);
        }
    }

    async function refreshAgentTools() {
        try {
            const tools = await ListAgentTools();
            setAgentTools(tools);
            setAgentToolPlan(buildAgentToolPlan(tools, filePreview, artifactMetadata, activeFile));
            await refreshAgentToolRuns();
        } catch {
            setAgentTools([]);
            setAgentToolPlan([]);
        }
    }

    async function refreshAgentToolRuns() {
        try {
            setAgentToolRuns(await ListAgentToolRuns());
        } catch {
            setAgentToolRuns([]);
        }
    }

    async function refreshChatHistory() {
        try {
            const messages = await GetChatHistory();
            setChatMessages(messages);
            setChatStatus(messages.length > 0 ? `${messages.length} saved chat messages loaded.` : 'Select text context and ask the assistant.');
        } catch {
            setChatMessages([]);
            setChatStatus('Chat history is available in the desktop runtime.');
        }
    }

    async function clearChatHistory() {
        try {
            const messages = await ClearChatHistory();
            setChatMessages(messages);
            setChatStatus('Chat history cleared for this workspace.');
            pushToolEvent('Chat cleared', 'Workspace chat history reset.');
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setChatStatus(message || 'Could not clear chat history.');
        }
    }

    async function removeRecentWorkspace(recentWorkspace: RecentWorkspace) {
        setIsManagingRecent(true);
        setWorkspaceStatus(`Removing ${recentWorkspace.name} from recent workspaces...`);

        try {
            const nextWorkspaces = await RemoveRecentWorkspace(recentWorkspace.path);
            onRecentWorkspacesChange(nextWorkspaces);
            setWorkspaceStatus(`${recentWorkspace.name} removed from recent workspaces.`);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            if (message.includes('undefined') || message.includes('window')) {
                setWorkspaceStatus('Recent workspace management is available in the desktop runtime.');
                return;
            }
            setWorkspaceStatus(message || `Could not remove ${recentWorkspace.name}.`);
        } finally {
            setIsManagingRecent(false);
        }
    }

    async function clearRecentWorkspaces() {
        setIsManagingRecent(true);
        setWorkspaceStatus('Clearing recent workspaces...');

        try {
            const nextWorkspaces = await ClearRecentWorkspaces();
            onRecentWorkspacesChange(nextWorkspaces);
            setWorkspaceStatus('Recent workspaces cleared.');
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            if (message.includes('undefined') || message.includes('window')) {
                setWorkspaceStatus('Recent workspace management is available in the desktop runtime.');
                return;
            }
            setWorkspaceStatus(message || 'Could not clear recent workspaces.');
        } finally {
            setIsManagingRecent(false);
        }
    }

    function updateSettingsDraft(field: keyof LLMSettings, value: string) {
        setSettingsDraft((current) => ({
            ...current,
            [field]: value,
        }));
    }

    async function saveLLMSettings() {
        setIsSavingSettings(true);
        setSettingsStatus('Saving LLM settings...');

        try {
            const saved = await SaveLLMSettings(settingsDraft);
            onLLMSettingsChange(saved);
            setSettingsDraft(saved);
            setProbeResult(null);
            setSettingsStatus('LLM settings saved locally.');
            pushToolEvent('LLM settings saved', saved.model || saved.baseUrl);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            if (message.includes('undefined') || message.includes('window')) {
                setSettingsStatus('LLM settings save is available in the desktop runtime.');
                return;
            }
            setSettingsStatus(message || 'Could not save LLM settings.');
        } finally {
            setIsSavingSettings(false);
        }
    }

    async function testLLMConnection() {
        setIsTestingConnection(true);
        setProbeResult(null);
        setSettingsStatus('Testing LLM provider...');

        try {
            const result = await TestLLMConnection(settingsDraft);
            setProbeResult(result);
            if (result.ok) {
                const suffix = result.modelCount > 0 ? ` ${result.modelCount} models found.` : '';
                setSettingsStatus(`${result.message}${suffix}`);
            } else {
                setSettingsStatus(result.message || 'Provider did not accept the request.');
            }
            pushToolEvent('LLM connection tested', result.message);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            if (message.includes('undefined') || message.includes('window')) {
                setSettingsStatus('LLM connection test is available in the desktop runtime.');
                return;
            }
            setSettingsStatus(message || 'LLM connection test failed.');
        } finally {
            setIsTestingConnection(false);
        }
    }

    async function sendPrompt() {
        await sendPromptText(chatPrompt, {clearComposer: true});
    }

    async function explainSelectedContext() {
        if (!selectedTextContextRelPath()) {
            setChatStatus('Select a text, extracted document, or directory context before asking for an explanation.');
            return;
        }

        const prompt = [
            `Explain ${filePreview?.relPath}.`,
            'Cover purpose, important structures, notable dependencies, risks, and practical next steps.',
        ].join(' ');
        await sendPromptText(prompt, {clearComposer: false});
    }

    async function summarizeSelectedContext() {
        const contextRelPath = selectedTextContextRelPath();
        if (!contextRelPath) {
            setChatStatus('Select a text, extracted document, or directory context before summarizing.');
            return;
        }

        const prompt = [
            `Summarize ${contextRelPath}.`,
            'Return a concise Markdown summary with overview, key details, risks or gaps, and practical next actions.',
            'Stay grounded in the provided context and call out missing context if the source is incomplete.',
        ].join(' ');

        setIsSummarizingContext(true);
        try {
            await sendPromptText(prompt, {
                clearComposer: false,
                contextPaths: [contextRelPath],
                saveArtifactSource: 'NexusDesk summary',
                saveArtifactTitle: `Summary - ${contextRelPath}`,
            });
        } finally {
            setIsSummarizingContext(false);
        }
    }

    async function sendPromptText(rawPrompt: string, options: SendPromptOptions) {
        const prompt = rawPrompt.trim();
        if (!prompt) {
            setChatStatus('Write a prompt before sending.');
            return;
        }

        const selectedContextRelPath = selectedTextContextRelPath();
        const contextPaths = options.contextPaths ?? (contextPackPaths.length > 0 ? contextPackPaths : selectedContextRelPath ? [selectedContextRelPath] : []);
        const contextRelPath = contextPaths.length > 1 ? `pack: ${contextPaths.join(', ')}` : contextPaths[0] ?? '';
        const sourcePaths = contextPackPreview && contextPackPaths.length > 0 && !options.contextPaths
            ? contextPackPreview.files.map((file) => file.relPath)
            : sourcePathsFromContext(contextRelPath);
        const requestId = createRequestId();
        const userMessage: ChatMessage = {content: prompt, contextRelPath, sourcePaths, createdAt: new Date().toISOString(), role: 'user'};
        const assistantMessage: ChatMessage = {content: '', contextRelPath, sourcePaths, createdAt: new Date().toISOString(), role: 'assistant'};

        setIsSendingPrompt(true);
        setChatStatus(contextRelPath ? `Streaming with ${contextRelPath} as context...` : 'Streaming without selected file context...');
        setChatMessages((current) => [...current, userMessage, assistantMessage]);
        if (options.clearComposer) {
            setChatPrompt('');
        }

        const unsubscribe = listenForChatStream(requestId, assistantMessage.createdAt, contextRelPath);
        try {
            const result: LLMChatResult = contextPaths.length > 1
                ? (unsubscribe
                    ? await AskLLMStreamContextPack(prompt, contextPaths, requestId)
                    : await AskLLMContextPack(prompt, contextPaths))
                : (unsubscribe
                    ? await AskLLMStream(prompt, contextPaths[0] ?? '', requestId)
                    : await AskLLM(prompt, contextPaths[0] ?? ''));
            if (workspace) {
                await refreshChatHistory();
            } else {
                replaceChatMessage(assistantMessage.createdAt, result.message, result.contextRelPath, result.sourcePaths);
            }
            if (options.saveArtifactTitle && workspace) {
                const report = await writeAssistantArtifact({
                    content: result.message,
                    contextRelPath: result.contextRelPath,
                    eventTitle: 'Summary saved',
                    model: result.model,
                    prompt,
                    source: options.saveArtifactSource ?? 'NexusDesk chat',
                    sourcePaths: result.sourcePaths && result.sourcePaths.length > 0 ? result.sourcePaths : sourcePaths,
                    title: options.saveArtifactTitle,
                });
                setChatStatus(`${report.name} saved as a Markdown summary.`);
            } else {
                setChatStatus(result.contextRelPath ? `Answered with ${result.contextRelPath}.` : `Answered by ${result.model}.`);
            }
            pushToolEvent('Chat completed', result.contextRelPath || result.model);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            if (message.includes('undefined') || message.includes('window')) {
                replaceChatMessage(assistantMessage.createdAt, 'Chat is available in the desktop runtime.', '', []);
                setChatStatus('Chat is available in the desktop runtime.');
                return;
            }
            replaceChatMessage(assistantMessage.createdAt, message || 'The provider did not return a usable chat response.', '', []);
            setChatStatus(message || 'Chat request failed.');
        } finally {
            unsubscribe?.();
            setIsSendingPrompt(false);
        }
    }

    function selectedTextContextRelPath() {
        if ((filePreview?.kind === 'file' && filePreview.content) || (filePreview?.kind === 'pdf' && filePreview.text)) {
            return filePreview.relPath;
        }
        if (filePreview?.kind === 'directory') {
            return filePreview.relPath;
        }
        return '';
    }

    function pinSelectedContext() {
        const relPath = selectedTextContextRelPath();
        if (!relPath) {
            setChatStatus('Select text, CSV, extracted PDF, or directory context before pinning it.');
            return;
        }
        setContextPackPaths((current) => current.includes(relPath) ? current : [...current, relPath]);
        setChatStatus(`${relPath} pinned to the context pack.`);
        pushToolEvent('Context pinned', relPath);
    }

    function pinProjectContext() {
        if (!workspace) {
            setChatStatus('Open a workspace before pinning project context.');
            return;
        }
        setContextPackPaths((current) => current.includes('.') ? current : ['.', ...current]);
        setChatStatus('Workspace root pinned to the context pack.');
        pushToolEvent('Project context pinned', workspace.name);
    }

    function removeContextPath(relPath: string) {
        setContextPackPaths((current) => current.filter((path) => path !== relPath));
        setChatStatus(`${relPath} removed from the context pack.`);
    }

    async function createMarkdownReport() {
        if (!workspace) {
            setWorkspaceStatus('Open a workspace before creating reports.');
            return;
        }

        const sourceRelPath = filePreview?.relPath ?? '';
        setIsCreatingReport(true);
        setWorkspaceStatus(sourceRelPath ? `Creating report from ${sourceRelPath}...` : 'Creating workspace report...');

        try {
            const report: MarkdownReport = await CreateMarkdownReport(sourceRelPath);
            const result = await RefreshWorkspace();
            if (result.selected) {
                onWorkspaceChange(result.snapshot);
                await refreshArtifacts();
                await refreshApprovals();
                setExpandedDirectories((current) => reconcileExpandedDirectories(current, result.snapshot, findWorkspaceNode(result.snapshot, report.relPath)));
                await selectWorkspaceFile(result.snapshot, report.relPath);
                setWorkspaceStatus(`${report.name} created in .nexusdesk/artifacts.`);
                pushToolEvent('Report created', report.relPath);
            } else {
                setWorkspaceStatus(report.message);
            }
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            if (message.includes('undefined') || message.includes('window')) {
                setWorkspaceStatus('Report creation is available in the desktop runtime.');
                return;
            }
            setWorkspaceStatus(message || 'Could not create report artifact.');
        } finally {
            setIsCreatingReport(false);
        }
    }

    async function createScanReportArtifact() {
        if (!workspace) {
            setWorkspaceStatus('Open a workspace before creating scan reports.');
            return;
        }

        setIsCreatingScanReport(true);
        setWorkspaceStatus(`Saving scan report for ${workspace.name}...`);
        try {
            const report: MarkdownReport = await CreateScanReportArtifact();
            const result = await RefreshWorkspace();
            if (result.selected) {
                onWorkspaceChange(result.snapshot);
                await refreshArtifacts();
                await refreshApprovals();
                const reportNode = findWorkspaceNode(result.snapshot, report.relPath);
                setExpandedDirectories((current) => reconcileExpandedDirectories(current, result.snapshot, reportNode));
                await selectWorkspaceFile(result.snapshot, report.relPath);
                setWorkspaceStatus(`${report.name} saved in .nexusdesk/artifacts.`);
                pushToolEvent('Scan report saved', report.relPath);
            } else {
                setWorkspaceStatus(report.message);
            }
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setWorkspaceStatus(message || 'Could not save scan report.');
        } finally {
            setIsCreatingScanReport(false);
        }
    }

    async function saveLatestAssistantArtifact() {
        if (!workspace) {
            setChatStatus('Open a workspace before saving answers as artifacts.');
            return;
        }

        const latest = latestAssistantMessage(chatMessages);
        if (!latest) {
            setChatStatus('No assistant answer is ready to save yet.');
            return;
        }

        setIsSavingChatArtifact(true);
        setChatStatus('Saving latest assistant answer as Markdown...');

        try {
            const report = await writeAssistantArtifact({
                title: latestAssistantArtifactTitle(chatMessages),
                content: latest.content,
                contextRelPath: latest.contextRelPath,
                prompt: latestUserPromptForAssistant(chatMessages),
                model: settingsDraft.model,
                eventTitle: 'Answer saved',
                source: 'NexusDesk chat',
                sourcePaths: latest.sourcePaths && latest.sourcePaths.length > 0 ? latest.sourcePaths : sourcePathsFromContext(latest.contextRelPath),
            });
            setChatStatus(`${report.name} saved as a Markdown artifact.`);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            if (message.includes('undefined') || message.includes('window')) {
                setChatStatus('Saving chat artifacts is available in the desktop runtime.');
                return;
            }
            setChatStatus(message || 'Could not save the assistant answer.');
        } finally {
            setIsSavingChatArtifact(false);
        }
    }

    async function writeAssistantArtifact(request: AssistantArtifactWriteRequest) {
        const report: MarkdownReport = await CreateChatMarkdownArtifact({
            title: request.title,
            content: request.content,
            contextRelPath: request.contextRelPath,
            prompt: request.prompt,
            model: request.model,
            source: request.source,
            sourcePaths: request.sourcePaths,
        });
        const result = await RefreshWorkspace();
        if (result.selected) {
            onWorkspaceChange(result.snapshot);
            await refreshArtifacts();
            await refreshApprovals();
            setExpandedDirectories((current) => reconcileExpandedDirectories(current, result.snapshot, findWorkspaceNode(result.snapshot, report.relPath)));
            await selectWorkspaceFile(result.snapshot, report.relPath);
            setWorkspaceStatus(`${report.name} saved in .nexusdesk/artifacts.`);
        }
        pushToolEvent(request.eventTitle, report.relPath);
        return report;
    }

    async function profileSelectedDataset() {
        if (!workspace || !filePreview) {
            setWorkspaceStatus('Open a workspace and select a dataset before profiling.');
            return;
        }

        setIsProfilingDataset(true);
        try {
            const profile = await ProfileDataset(filePreview.relPath);
            await refreshDatasetProfiles();
            setActiveDatasetProfile(profile);
            setWorkspaceStatus(profile.message);
            pushToolEvent('Dataset profiled', profile.relPath);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setWorkspaceStatus(message || 'Could not profile this dataset.');
        } finally {
            setIsProfilingDataset(false);
        }
    }

    async function querySelectedDataset() {
        if (!workspace || !filePreview) {
            setWorkspaceStatus('Open a workspace and select a CSV dataset before querying.');
            return;
        }

        setIsQueryingDataset(true);
        try {
            const result = await QueryDataset(filePreview.relPath, datasetQuery);
            setDatasetQueryResult(result);
            setWorkspaceStatus(result.message);
            pushToolEvent('Dataset queried', `${result.relPath}: ${result.query || 'first rows'}`);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setWorkspaceStatus(message || 'Could not query this dataset.');
        } finally {
            setIsQueryingDataset(false);
        }
    }

    async function querySelectedDatasetSQL() {
        if (!workspace || !filePreview?.table) {
            setWorkspaceStatus('Open a workspace and select a CSV dataset before running SQL.');
            return;
        }

        setIsQueryingDatasetSQL(true);
        try {
            const result = await QueryDatasetSQL({relPath: filePreview.relPath, sql: datasetSQLQuery});
            setDatasetSQLQueryResult(result);
            setWorkspaceStatus(result.message);
            pushToolEvent('Dataset SQL queried', `${result.engine}: ${result.relPath}`);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setWorkspaceStatus(message || 'Could not run read-only SQL query.');
        } finally {
            setIsQueryingDatasetSQL(false);
        }
    }

    async function saveCurrentDatasetQuery() {
        if (!workspace || !filePreview?.table) {
            setWorkspaceStatus('Select a CSV dataset before saving a query.');
            return;
        }

        setIsSavingDatasetQuery(true);
        try {
            const saved = await SaveDatasetQuery(filePreview.relPath, datasetQuery, datasetQueryLabel);
            setDatasetQueryLabel('');
            await refreshSavedDatasetQueries(filePreview.relPath);
            setWorkspaceStatus(`Saved dataset query "${saved.label}".`);
            pushToolEvent('Dataset query saved', `${saved.relPath}: ${saved.label}`);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setWorkspaceStatus(message || 'Could not save dataset query.');
        } finally {
            setIsSavingDatasetQuery(false);
        }
    }

    async function previewDatasetChart() {
        if (!workspace || !filePreview?.table) {
            setWorkspaceStatus('Open a workspace and select a CSV dataset before previewing a chart.');
            return;
        }
        if (!datasetChartCategory) {
            setWorkspaceStatus('Choose a category column before previewing a chart.');
            return;
        }

        setIsPreviewingDatasetChart(true);
        setDatasetChartPreview(null);
        try {
            const preview = await PreviewDatasetChart({
                relPath: filePreview.relPath,
                chartType: datasetChartType,
                categoryColumn: datasetChartCategory,
                valueColumn: datasetChartValue,
            });
            setDatasetChartPreview(preview);
            setWorkspaceStatus(preview.message);
            pushToolEvent('Chart previewed', `${preview.relPath}: ${preview.chartType}`);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setWorkspaceStatus(message || 'Could not preview dataset chart.');
        } finally {
            setIsPreviewingDatasetChart(false);
        }
    }

    async function createDatasetSummary() {
        if (!workspace || !filePreview?.table) {
            setWorkspaceStatus('Open a workspace and select a CSV dataset before creating a summary.');
            return;
        }

        setIsCreatingDatasetSummary(true);
        setWorkspaceStatus(`Creating dataset summary from ${filePreview.relPath}...`);
        try {
            const report: MarkdownReport = await CreateDatasetSummaryArtifact(filePreview.relPath);
            const result = await RefreshWorkspace();
            if (result.selected) {
                onWorkspaceChange(result.snapshot);
                await refreshArtifacts();
                await refreshApprovals();
                setExpandedDirectories((current) => reconcileExpandedDirectories(current, result.snapshot, findWorkspaceNode(result.snapshot, report.relPath)));
                await selectWorkspaceFile(result.snapshot, report.relPath);
                setWorkspaceStatus(`${report.name} created in .nexusdesk/artifacts.`);
                pushToolEvent('Dataset summary created', report.relPath);
            } else {
                setWorkspaceStatus(report.message);
            }
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setWorkspaceStatus(message || 'Could not create dataset summary.');
        } finally {
            setIsCreatingDatasetSummary(false);
        }
    }

    async function createDatasetChart() {
        if (!workspace || !filePreview?.table) {
            setWorkspaceStatus('Open a workspace and select a CSV dataset before creating a chart.');
            return;
        }
        if (!datasetChartCategory) {
            setWorkspaceStatus('Choose a category column before creating a chart.');
            return;
        }

        setIsCreatingDatasetChart(true);
        setWorkspaceStatus(`Creating chart from ${filePreview.relPath}...`);
        try {
            const report: MarkdownReport = await CreateDatasetChartArtifact({
                relPath: filePreview.relPath,
                chartType: datasetChartType,
                categoryColumn: datasetChartCategory,
                valueColumn: datasetChartValue,
            });
            const result = await RefreshWorkspace();
            if (result.selected) {
                onWorkspaceChange(result.snapshot);
                await refreshArtifacts();
                await refreshApprovals();
                setExpandedDirectories((current) => reconcileExpandedDirectories(current, result.snapshot, findWorkspaceNode(result.snapshot, report.relPath)));
                await selectWorkspaceFile(result.snapshot, report.relPath);
                setWorkspaceStatus(`${report.name} created in .nexusdesk/artifacts.`);
                pushToolEvent('Chart created', report.relPath);
            } else {
                setWorkspaceStatus(report.message);
            }
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            if (message.includes('undefined') || message.includes('window')) {
                setWorkspaceStatus('Chart creation is available in the desktop runtime.');
                return;
            }
            setWorkspaceStatus(message || 'Could not create dataset chart.');
        } finally {
            setIsCreatingDatasetChart(false);
        }
    }

    async function exportDatasetQuery() {
        if (!workspace || !filePreview?.table) {
            setWorkspaceStatus('Open a workspace and select a CSV dataset before exporting a query.');
            return;
        }

        setIsExportingDatasetQuery(true);
        setWorkspaceStatus(`Exporting query result from ${filePreview.relPath}...`);
        try {
            const report: MarkdownReport = await CreateDatasetQueryArtifact(filePreview.relPath, datasetQuery);
            const result = await RefreshWorkspace();
            if (result.selected) {
                onWorkspaceChange(result.snapshot);
                await refreshArtifacts();
                await refreshApprovals();
                setExpandedDirectories((current) => reconcileExpandedDirectories(current, result.snapshot, findWorkspaceNode(result.snapshot, report.relPath)));
                await selectWorkspaceFile(result.snapshot, report.relPath);
                setWorkspaceStatus(`${report.name} exported in .nexusdesk/artifacts.`);
                pushToolEvent('Query exported', report.relPath);
            } else {
                setWorkspaceStatus(report.message);
            }
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            if (message.includes('undefined') || message.includes('window')) {
                setWorkspaceStatus('Dataset query export is available in the desktop runtime.');
                return;
            }
            setWorkspaceStatus(message || 'Could not export dataset query.');
        } finally {
            setIsExportingDatasetQuery(false);
        }
    }

    async function selectArtifact(artifact: WorkspaceArtifact) {
        if (!workspace) {
            setWorkspaceStatus('Open a workspace before selecting artifacts.');
            return;
        }

        const node = findWorkspaceNode(workspace, artifact.relPath);
        if (!node) {
            setWorkspaceStatus(`${artifact.name} is not visible in the current workspace tree. Refresh the workspace to reveal it.`);
            return;
        }

        await selectWorkspaceFile(workspace, artifact.relPath);
        setWorkspaceStatus(`${artifact.name} selected from artifacts.`);
    }

    async function openArtifactSource() {
        if (!workspace || !artifactMetadata?.contextRelPath) {
            setWorkspaceStatus('This artifact does not include source context.');
            return;
        }

        const source = artifactMetadata.contextRelPath;
        const node = findWorkspaceNode(workspace, source);
        if (!node) {
            setWorkspaceStatus(`${source} is not visible in the current workspace tree.`);
            return;
        }
        await selectWorkspaceFile(workspace, source);
        setWorkspaceStatus(`Opened source context ${source}.`);
        pushToolEvent('Artifact source opened', source);
    }

    async function archiveActiveArtifact() {
        if (!workspace || !filePreview || !isArtifactPath(filePreview.relPath)) {
            setWorkspaceStatus('Select an artifact before archiving it.');
            return;
        }

        const relPath = filePreview.relPath;
        const approved = await requestApproval({
            action: 'Archive artifact',
            confirmLabel: 'Archive',
            message: `Move ${relPath} to .nexusdesk/artifacts/archive.`,
            risk: 'medium',
            target: relPath,
        });
        if (!approved) {
            setWorkspaceStatus(`Archive cancelled for ${relPath}.`);
            return;
        }

        setIsArchivingArtifact(true);
        try {
            const archived = await ArchiveArtifact(relPath);
            const result = await RefreshWorkspace();
            if (result.selected) {
                onWorkspaceChange(result.snapshot);
                await refreshArtifacts();
                await refreshApprovals();
                const archiveNode = findWorkspaceNode(result.snapshot, archived.relPath);
                setExpandedDirectories((current) => reconcileExpandedDirectories(current, result.snapshot, archiveNode));
                await selectWorkspaceFile(result.snapshot, archived.relPath);
            }
            setWorkspaceStatus(archived.message);
            pushToolEvent('Artifact archived', archived.relPath);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setWorkspaceStatus(message || 'Could not archive artifact.');
        } finally {
            setIsArchivingArtifact(false);
        }
    }

    async function deleteActiveArtifact() {
        if (!workspace || !filePreview || !isArtifactPath(filePreview.relPath)) {
            setWorkspaceStatus('Select an artifact before deleting it.');
            return;
        }

        const relPath = filePreview.relPath;
        const sourceRelPath = artifactMetadata?.contextRelPath ?? '';
        const approved = await requestApproval({
            action: 'Delete artifact',
            confirmLabel: 'Delete',
            message: `Delete ${relPath} and its metadata sidecar from the workspace.`,
            risk: 'high',
            target: relPath,
        });
        if (!approved) {
            setWorkspaceStatus(`Delete cancelled for ${relPath}.`);
            return;
        }

        setIsDeletingArtifact(true);
        try {
            const deleted = await DeleteArtifact(relPath);
            const result = await RefreshWorkspace();
            if (result.selected) {
                onWorkspaceChange(result.snapshot);
                removeOpenTabState(deleted.relPath);
                await refreshArtifacts();
                await refreshApprovals();
                const sourceNode = sourceRelPath ? findWorkspaceNode(result.snapshot, sourceRelPath) : null;
                const selectedNode = sourceNode ?? selectNodeAfterWorkspaceUpdate(result.snapshot);
                setExpandedDirectories((current) => reconcileExpandedDirectories(current, result.snapshot, selectedNode));
                if (selectedNode) {
                    await previewWorkspaceNode(selectedNode, false);
                    setActiveFile(selectedNode.relPath);
                } else {
                    setActiveFile(result.snapshot.name);
                    setFilePreview(null);
                }
            }
            setWorkspaceStatus(deleted.message);
            pushToolEvent('Artifact deleted', deleted.relPath);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setWorkspaceStatus(message || 'Could not delete artifact.');
        } finally {
            setIsDeletingArtifact(false);
        }
    }

    async function compareActiveArtifactWithPrevious() {
        if (!workspace || !filePreview || !isArtifactPath(filePreview.relPath)) {
            setWorkspaceStatus('Select an artifact before comparing versions.');
            return;
        }

        const currentIndex = artifacts.findIndex((item) => item.relPath === filePreview.relPath);
        const previous = artifacts
            .slice(currentIndex + 1)
            .find((item) => item.kind === (artifactMetadata?.kind || item.kind));
        if (!previous) {
            setWorkspaceStatus('No earlier artifact of the same kind is available for comparison.');
            return;
        }

        try {
            const comparison = await CompareArtifacts(previous.relPath, filePreview.relPath);
            setArtifactComparison(comparison);
            setWorkspaceStatus(comparison.message);
            pushToolEvent('Artifacts compared', `${previous.name} -> ${filePreview.name}`);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setWorkspaceStatus(message || 'Could not compare artifacts.');
        }
    }

    async function prepareSQLiteMetadataStore() {
        if (!workspace) {
            setWorkspaceStatus('Open a workspace before preparing metadata storage.');
            return;
        }

        setIsPreparingMetadataStore(true);
        try {
            const status = await EnsureSQLiteMetadataStore();
            setSQLiteStatus(status);
            await refreshApprovals();
            setWorkspaceStatus(status.message);
            pushToolEvent('SQLite schema prepared', `${status.tables.length} metadata tables`);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setWorkspaceStatus(message || 'Could not prepare SQLite metadata schema.');
        } finally {
            setIsPreparingMetadataStore(false);
        }
    }

    async function dryRunAgentTool(item: AgentToolPlanItem) {
        await runAgentTool(item, false);
    }

    async function executeAgentTool(item: AgentToolPlanItem) {
        await runAgentTool(item, true);
    }

    async function runAgentTool(item: AgentToolPlanItem, execute: boolean) {
        if (!workspace) {
            setChatStatus('Open a workspace before running tools.');
            return;
        }
        if (execute && item.requiresApproval) {
            const approved = await requestApproval({
                action: `Execute ${item.title}`,
                confirmLabel: 'Execute',
                message: `Run ${item.toolName} for ${item.target}.`,
                risk: approvalRisk(item.risk),
                target: item.target,
            });
            if (!approved) {
                setChatStatus(`Tool execution cancelled for ${item.toolName}.`);
                return;
            }
        }

        setIsRunningAgentTool(true);
        try {
            const request = {
                toolName: item.toolName,
                target: item.target,
                inputs: agentToolInputs(item, datasetQuery),
                approved: execute,
                approvalId: '',
            };
            const record = execute ? await ExecuteAgentTool(request) : await PreviewAgentTool(request);
            await refreshAgentToolRuns();
            await refreshApprovals();
            if (execute && (item.toolName.startsWith('artifact.') || item.toolName === 'metadata.sqlite.prepare')) {
                const result = await RefreshWorkspace();
                if (result.selected) {
                    onWorkspaceChange(result.snapshot);
                    await refreshArtifacts();
                }
            }
            const summary = record.outputSummary || record.error || `${record.title} ${record.status}`;
            setChatStatus(summary);
            pushToolEvent(execute ? 'Tool executed' : 'Tool dry run', summary);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setChatStatus(message || 'Agent tool run failed.');
        } finally {
            setIsRunningAgentTool(false);
        }
    }

    function listenForChatStream(requestId: string, assistantCreatedAt: string, fallbackContextRelPath: string) {
        if (!isWailsRuntimeAvailable()) {
            return null;
        }

        try {
            return EventsOn(chatStreamEventName, (event: ChatStreamEvent) => {
                if (event.requestId !== requestId) {
                    return;
                }

                if (event.type === 'delta') {
                    appendChatDelta(assistantCreatedAt, event.delta, event.contextRelPath || fallbackContextRelPath, event.sourcePaths);
                }
                if (event.type === 'done') {
                    replaceChatMessage(assistantCreatedAt, event.message, event.contextRelPath || fallbackContextRelPath, event.sourcePaths);
                }
                if (event.type === 'error') {
                    replaceChatMessage(assistantCreatedAt, event.message || 'Streaming response failed.', '', []);
                }
            });
        } catch {
            return null;
        }
    }

    function appendChatDelta(createdAt: string, delta: string, contextRelPath: string, sourcePaths?: string[]) {
        if (!delta) {
            return;
        }

        setChatMessages((current) => current.map((message) => {
            if (message.createdAt !== createdAt) {
                return message;
            }
            return {
                ...message,
                content: `${message.content}${delta}`,
                contextRelPath,
                sourcePaths: sourcePaths && sourcePaths.length > 0 ? sourcePaths : message.sourcePaths,
            };
        }));
    }

    function replaceChatMessage(createdAt: string, content: string, contextRelPath: string, sourcePaths?: string[]) {
        setChatMessages((current) => current.map((message) => {
            if (message.createdAt !== createdAt) {
                return message;
            }
            return {
                ...message,
                content,
                contextRelPath,
                sourcePaths: sourcePaths && sourcePaths.length > 0 ? sourcePaths : message.sourcePaths,
            };
        }));
    }

    function startNavigatorResize(event: ReactMouseEvent<HTMLDivElement>) {
        event.preventDefault();

        function resize(moveEvent: MouseEvent) {
            setNavigatorWidth(clamp(moveEvent.clientX - railWidth, navigatorMinWidth, navigatorMaxWidth));
        }

        function stopResize() {
            document.body.style.cursor = '';
            document.body.style.userSelect = '';
            window.removeEventListener('mousemove', resize);
            window.removeEventListener('mouseup', stopResize);
        }

        document.body.style.cursor = 'col-resize';
        document.body.style.userSelect = 'none';
        window.addEventListener('mousemove', resize);
        window.addEventListener('mouseup', stopResize);
    }

    return (
        <div className="app-shell" style={{'--navigator-width': `${navigatorWidth}px`} as CSSProperties}>
            <WorkspaceRail />
            <QuickOpenPalette
                activeFile={activeFile}
                isOpen={isQuickOpenOpen}
                onClose={() => setIsQuickOpenOpen(false)}
                onQueryChange={setQuickOpenQuery}
                onSelectNode={(node) => void selectQuickOpenNode(node)}
                onSelectTab={selectOpenTab}
                openTabs={openTabs}
                query={quickOpenQuery}
                workspace={workspace}
            />
            <CommandPalette
                commands={commandActions}
                isOpen={isCommandPaletteOpen}
                onClose={() => setIsCommandPaletteOpen(false)}
                onQueryChange={setCommandPaletteQuery}
                query={commandPaletteQuery}
            />
            <ApprovalRequestModal
                onApprove={() => resolveApproval(true)}
                onCancel={() => resolveApproval(false)}
                prompt={approvalPrompt}
            />

            <WorkspaceNavigator
                activeFile={activeFile}
                buildStage={state.buildStage}
                expandedDirectories={expandedDirectories}
                isSearchingWorkspace={isSearchingWorkspace}
                isManagingRecent={isManagingRecent}
                isOpeningWorkspace={isOpeningWorkspace}
                isRefreshingWorkspace={isRefreshingWorkspace}
                isCreatingScanReport={isCreatingScanReport}
                onClearRecentWorkspaces={() => void clearRecentWorkspaces()}
                onClearWorkspaceSearch={() => setWorkspaceSearchResults([])}
                onCollapseAllDirectories={collapseAllDirectories}
                onCreateScanReport={() => void createScanReportArtifact()}
                onExpandAllDirectories={expandAllDirectories}
                onOpenWorkspace={() => void openWorkspace()}
                onRefreshWorkspace={() => void refreshWorkspace()}
                onRemoveRecentWorkspace={(recentWorkspace) => void removeRecentWorkspace(recentWorkspace)}
                onReopenWorkspace={(recentWorkspace) => void reopenWorkspace(recentWorkspace)}
                onSearchWorkspace={() => void searchWorkspace()}
                onSelectFallbackItem={selectFallbackItem}
                onSelectSearchResult={(result) => void selectSearchResult(result)}
                onSelectWorkspaceNode={(node) => void selectWorkspaceNode(node)}
                onWorkspaceSearchQueryChange={setWorkspaceSearchQuery}
                recentWorkspaces={recentWorkspaces}
                workspace={workspace}
                workspaceItems={state.workspaceItems}
                workspaceNodes={workspaceNodes}
                workspaceSearchQuery={workspaceSearchQuery}
                workspaceSearchResults={workspaceSearchResults}
                workspaceStatus={workspaceStatus}
            />

            <div
                aria-label="Resize workspace navigator"
                className="navigator-resizer"
                onMouseDown={startNavigatorResize}
                role="separator"
            />

            <WorkbenchPanel
                activeFile={activeFile}
                artifacts={artifacts}
                artifactMetadata={artifactMetadata}
                approvalRecords={approvalRecords}
                capabilities={state.capabilities}
                datasetProfiles={datasetProfiles}
                datasetQuery={datasetQuery}
                datasetQueryLabel={datasetQueryLabel}
                datasetQueryResult={datasetQueryResult}
                datasetSQLQuery={datasetSQLQuery}
                datasetSQLQueryResult={datasetSQLQueryResult}
                savedDatasetQueries={savedDatasetQueries}
                artifactComparison={artifactComparison}
                sqliteStatus={sqliteStatus}
                datasetChartPreview={datasetChartPreview}
                datasetChartCategory={datasetChartCategory}
                datasetChartType={datasetChartType}
                datasetChartValue={datasetChartValue}
                activeDatasetProfile={activeDatasetProfile}
                fileDraft={fileDraft}
                filePreview={filePreview}
                dirtyTabPaths={dirtyTabPaths}
                isCreatingReport={isCreatingReport}
                isDeletingFile={isDeletingFile}
                isMovingFile={isMovingFile}
                isProfilingDataset={isProfilingDataset}
                isQueryingDataset={isQueryingDataset}
                isQueryingDatasetSQL={isQueryingDatasetSQL}
                isSavingDatasetQuery={isSavingDatasetQuery}
                isPreparingMetadataStore={isPreparingMetadataStore}
                isPreviewingDatasetChart={isPreviewingDatasetChart}
                isCreatingDatasetChart={isCreatingDatasetChart}
                isCreatingDatasetSummary={isCreatingDatasetSummary}
                isExportingDatasetQuery={isExportingDatasetQuery}
                isArchivingArtifact={isArchivingArtifact}
                isDeletingArtifact={isDeletingArtifact}
                isSummarizingContext={isSummarizingContext}
                isEditingFile={isEditingFile}
                isApplyingWrite={isApplyingWrite}
                isLoadingPreview={isLoadingPreview}
                isPreviewingWrite={isPreviewingWrite}
                isSendingPrompt={isSendingPrompt}
                onApplyFileWrite={() => void applyFileWrite()}
                onCancelFileEdit={clearFileWriteDraft}
                onCreateReport={() => void createMarkdownReport()}
                onDatasetQueryChange={setDatasetQuery}
                onDatasetSQLQueryChange={setDatasetSQLQuery}
                onDatasetQueryLabelChange={setDatasetQueryLabel}
                onSaveDatasetQuery={() => void saveCurrentDatasetQuery()}
                onDatasetChartCategoryChange={(value) => {
                    setDatasetChartCategory(value);
                    setDatasetChartPreview(null);
                }}
                onDatasetChartTypeChange={(value) => {
                    setDatasetChartType(value);
                    setDatasetChartPreview(null);
                }}
                onDatasetChartValueChange={(value) => {
                    setDatasetChartValue(value);
                    setDatasetChartPreview(null);
                }}
                onDeleteFile={() => void deleteActiveFile()}
                onMoveFile={() => void moveActiveFile()}
                onFileDraftChange={updateFileDraft}
                onExplainContext={() => void explainSelectedContext()}
                onSummarizeContext={() => void summarizeSelectedContext()}
                onPinContext={pinSelectedContext}
                onPinProjectContext={pinProjectContext}
                onPreviewFileWrite={() => void previewFileWrite()}
                onProfileDataset={() => void profileSelectedDataset()}
                onQueryDataset={() => void querySelectedDataset()}
                onQueryDatasetSQL={() => void querySelectedDatasetSQL()}
                onPreviewDatasetChart={() => void previewDatasetChart()}
                onCreateDatasetChart={() => void createDatasetChart()}
                onCreateDatasetSummary={() => void createDatasetSummary()}
                onExportDatasetQuery={() => void exportDatasetQuery()}
                onArchiveArtifact={() => void archiveActiveArtifact()}
                onCompareArtifact={() => void compareActiveArtifactWithPrevious()}
                onCloseTab={closeOpenTab}
                onDeleteArtifact={() => void deleteActiveArtifact()}
                onOpenArtifactSource={() => void openArtifactSource()}
                onPrepareMetadataStore={() => void prepareSQLiteMetadataStore()}
                onSelectTab={selectOpenTab}
                onSelectArtifact={(artifact) => void selectArtifact(artifact)}
                onStartFileEdit={startFileEdit}
                onRefreshPreview={() => void refreshSelectedPreview()}
                openTabs={openTabs}
                selectedMeta={selectedMeta}
                writeProposal={writeProposal}
                workspace={workspace}
            />

            <AgentPanel
                chatMessages={chatMessages}
                chatPrompt={chatPrompt}
                chatStatus={chatStatus}
                contextPackPreview={contextPackPreview}
                contextPackPaths={contextPackPaths}
                agentTools={agentTools}
                agentToolPlan={agentToolPlan}
                agentToolRuns={agentToolRuns}
                canSaveLatestAssistantArtifact={canSaveLatestAssistantArtifact}
                isSavingSettings={isSavingSettings}
                isSavingChatArtifact={isSavingChatArtifact}
                isSendingPrompt={isSendingPrompt}
                isTestingConnection={isTestingConnection}
                isRunningAgentTool={isRunningAgentTool}
                onChatPromptChange={setChatPrompt}
                onClearChatHistory={() => void clearChatHistory()}
                onClearContextPack={() => setContextPackPaths([])}
                onDryRunAgentTool={(item) => void dryRunAgentTool(item)}
                onExecuteAgentTool={(item) => void executeAgentTool(item)}
                onRefreshAgentPlan={() => void refreshAgentTools()}
                onRemoveContextPath={removeContextPath}
                onSaveLatestAssistantArtifact={() => void saveLatestAssistantArtifact()}
                onSaveSettings={() => void saveLLMSettings()}
                onSendPrompt={() => void sendPrompt()}
                onSettingsDraftChange={updateSettingsDraft}
                onTestConnection={() => void testLLMConnection()}
                probeResult={probeResult}
                settingsDraft={settingsDraft}
                settingsStatus={settingsStatus}
                tagline={state.tagline}
                toolEvents={localToolEvents}
            />
        </div>
    );
}

function findWorkspaceNode(snapshot: WorkspaceSnapshot, relPath: string) {
    return snapshot.nodes.find((node) => node.relPath === relPath) ?? null;
}

function dirtyDraftPaths(fileDrafts: Record<string, string>, openTabs: FilePreview[]) {
    return Object.entries(fileDrafts)
        .filter(([relPath, draft]) => {
            const tab = openTabs.find((current) => current.relPath === relPath);
            return Boolean(tab && draft !== tab.content);
        })
        .map(([relPath]) => relPath);
}

function currentDatasetColumns(preview: FilePreview | null, profile: DatasetProfile | null) {
    if (preview?.table?.columns.length) {
        return preview.table.columns;
    }
    if (profile?.profiles.length) {
        return profile.profiles.map((column) => column.name);
    }
    return [];
}

function isArtifactPath(relPath: string) {
    const normalized = relPath.replaceAll('\\', '/').toLowerCase();
    return normalized.startsWith('.nexusdesk/artifacts/') || normalized.includes('/.nexusdesk/artifacts/');
}

function omitKey<T>(record: Record<string, T>, key: string) {
    const next = {...record};
    delete next[key];
    return next;
}

function normalizeNewFileRelPath(rawRelPath: string) {
    return rawRelPath.trim().replaceAll('\\', '/').replace(/^\/+/, '').replace(/\/+/g, '/');
}

function suggestedNewFilePath(preview: FilePreview | null, activeFile: string) {
    const currentPath = preview?.relPath || activeFile;
    if (!currentPath || !currentPath.includes('/')) {
        return 'notes/new-file.md';
    }

    const directory = currentPath.endsWith('/')
        ? currentPath.replace(/\/+$/, '')
        : currentPath.split('/').slice(0, -1).join('/');
    return directory ? `${directory}/new-file.md` : 'new-file.md';
}

function defaultNewFileContent(relPath: string) {
    const name = fileNameFromRelPath(relPath);
    if (/\.mdx?$/i.test(relPath)) {
        const title = name.replace(/\.[^.]+$/, '').replace(/[-_]+/g, ' ').trim() || 'New File';
        return `# ${title}\n\n`;
    }
    if (/\.jsonc?$/i.test(relPath)) {
        return "{}\n";
    }
    if (/\.ya?ml$/i.test(relPath)) {
        return "---\n";
    }
    if (/\.html?$/i.test(relPath)) {
        return "<!doctype html>\n";
    }
    if (/\.css$/i.test(relPath)) {
        return ":root {\n}\n";
    }
    if (/\.gitignore$/i.test(relPath)) {
        return "# Ignore local files\n";
    }
    return "\n";
}

function fileNameFromRelPath(relPath: string) {
    return relPath.split('/').filter(Boolean).pop() ?? relPath;
}

function fileTypeForRelPath(relPath: string) {
    if (/\.(csv|tsv|xlsx?|parquet|jsonl)$/i.test(relPath)) {
        return 'data';
    }
    if (/\.(mdx?|txt|pdf|docx?|rtf)$/i.test(relPath)) {
        return 'document';
    }
    return 'code';
}

function latestAssistantMessage(messages: ChatMessage[]) {
    return [...messages].reverse().find((message) => message.role === 'assistant' && message.content.trim()) ?? null;
}

function latestAssistantArtifactTitle(messages: ChatMessage[]) {
    const assistantIndex = findLatestAssistantIndex(messages);
    if (assistantIndex === -1) {
        return 'Assistant response';
    }

    const prompt = latestUserPromptBefore(messages, assistantIndex);
    if (!prompt) {
        return 'Assistant response';
    }

    return `Assistant response - ${prompt.replace(/\s+/g, ' ').slice(0, 64)}`;
}

function latestUserPromptForAssistant(messages: ChatMessage[]) {
    const assistantIndex = findLatestAssistantIndex(messages);
    if (assistantIndex === -1) {
        return '';
    }
    return latestUserPromptBefore(messages, assistantIndex);
}

function latestUserPromptBefore(messages: ChatMessage[], index: number) {
    return messages
        .slice(0, index)
        .reverse()
        .find((message) => message.role === 'user' && message.content.trim())?.content.trim() ?? '';
}

function findLatestAssistantIndex(messages: ChatMessage[]) {
    for (let index = messages.length - 1; index >= 0; index -= 1) {
        const message = messages[index];
        if (message.role === 'assistant' && message.content.trim()) {
            return index;
        }
    }
    return -1;
}

function sourcePathsFromContext(contextRelPath: string) {
    if (!contextRelPath) {
        return [];
    }
    if (contextRelPath.startsWith('pack: ')) {
        return contextRelPath
            .slice('pack: '.length)
            .split(',')
            .map((path) => path.trim())
            .filter(Boolean);
    }
    const dirMatch = contextRelPath.match(/^dir: (.+) \(\d+ files\)$/);
    if (dirMatch) {
        return [dirMatch[1]];
    }
    return [contextRelPath];
}

function buildAgentToolPlan(
    tools: AgentToolDescriptor[],
    preview: FilePreview | null,
    metadata: ArtifactMetadata | null,
    activeFile: string
): AgentToolPlanItem[] {
    if (tools.length === 0) {
        return [];
    }

    const plan: AgentToolPlanItem[] = [];
    const byName = new Map(tools.map((tool) => [tool.name, tool]));
    const target = preview?.relPath || activeFile;
    const add = (toolName: string, nextTarget: string, status: string) => {
        const tool = byName.get(toolName);
        if (!tool || !nextTarget || plan.some((item) => item.toolName === toolName && item.target === nextTarget)) {
            return;
        }
        plan.push({
            toolName,
            title: tool.title,
            target: nextTarget,
            risk: tool.risk,
            requiresApproval: tool.requiresApproval,
            status,
        });
    };

    if (target) {
        add('workspace.preview', target, 'ready');
    }
    if (preview?.table) {
        add('dataset.query', preview.relPath, 'ready for filter, order, and export');
        add('artifact.create', preview.relPath, 'ready to create report artifact');
    }
    if (metadata) {
        add('artifact.archive', preview?.relPath ?? target, 'requires confirmation');
    }
    if (preview?.kind === 'file' && !preview.table && !isArtifactPath(preview.relPath)) {
        add('workspace.write', preview.relPath, 'draft and diff required');
    }
    if (preview && isOperationsContextPath(preview.relPath, preview.name)) {
        add('operations.inspect', preview.relPath, 'read-only');
    }

    return plan.slice(0, 5);
}

function isOperationsContextPath(relPath: string, name: string) {
    const normalizedRelPath = relPath.toLowerCase();
    const normalizedName = name.toLowerCase();
    return normalizedName === 'dockerfile' ||
        normalizedName.includes('docker-compose') ||
        /^compose\.ya?ml$/i.test(normalizedName) ||
        normalizedRelPath.startsWith('services/') ||
        /\.(env|ps1|sh|bat|cmd|toml|ya?ml)$/i.test(normalizedName);
}

function agentToolInputs(item: AgentToolPlanItem, datasetQuery: string) {
    const inputs: Record<string, string> = {};
    if (item.toolName === 'dataset.query') {
        inputs.query = datasetQuery;
    }
    if (item.toolName === 'artifact.create') {
        inputs.kind = 'markdown-report';
        inputs.sourcePath = item.target;
    }
    return inputs;
}

function approvalRisk(risk: string): 'low' | 'medium' | 'high' {
    if (risk === 'low' || risk === 'medium' || risk === 'high') {
        return risk;
    }
    return 'medium';
}

function createRequestId() {
    if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
        return crypto.randomUUID();
    }
    return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function isWailsRuntimeAvailable() {
    return typeof window !== 'undefined' && 'runtime' in window;
}

function clamp(value: number, min: number, max: number) {
    return Math.min(Math.max(value, min), max);
}

function previewMeta(preview: FilePreview) {
    const details = [
        preview.fileType,
        preview.encoding,
        preview.size > 0 ? formatBytes(preview.size) : '',
        preview.truncated ? 'truncated' : '',
    ].filter(Boolean);

    return details.length > 0 ? details.join(' | ') : preview.name;
}

function formatBytes(size: number) {
    if (size < 1024) {
        return `${size} B`;
    }

    const units = ['KB', 'MB', 'GB'];
    let value = size / 1024;
    let unitIndex = 0;

    while (value >= 1024 && unitIndex < units.length - 1) {
        value /= 1024;
        unitIndex += 1;
    }

    return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unitIndex]}`;
}
