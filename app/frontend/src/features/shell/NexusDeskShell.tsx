import {useEffect, useMemo, useState} from 'react';
import type {CSSProperties, MouseEvent as ReactMouseEvent} from 'react';
import {
    ApplyFileWrite,
    AskLLM,
    AskLLMContextPack,
    AskLLMStream,
    AskLLMStreamContextPack,
    ClearRecentWorkspaces,
    ClearChatHistory,
    CreateChatMarkdownArtifact,
    CreateMarkdownReport,
    GetChatHistory,
    ListDatasetProfiles,
    GetRecentWorkspaces,
    ListArtifacts,
    OpenWorkspace,
    PreviewFileWrite,
    ProfileDataset,
    QueryDataset,
    ReadWorkspaceFile,
    RemoveRecentWorkspace,
    RefreshWorkspace,
    SaveLLMSettings,
    SearchWorkspace,
    SelectWorkspace,
    TestLLMConnection,
} from '../../../wailsjs/go/main/App';
import {EventsOn} from '../../../wailsjs/runtime/runtime';
import type {
    ChatStreamEvent,
    ChatMessage,
    DatasetProfile,
    DatasetQueryResult,
    FileNode,
    FilePreview,
    FileWriteProposal,
    LLMChatResult,
    LLMProbeResult,
    LLMSettings,
    MarkdownReport,
    RecentWorkspace,
    StartupState,
    ToolEvent,
    WorkspaceArtifact,
    WorkspaceSearchResult,
    WorkspaceOpenResult,
    WorkspaceSnapshot,
} from '../../types';
import {AgentPanel} from './AgentPanel';
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
    const [localToolEvents, setLocalToolEvents] = useState<ToolEvent[]>(state.toolEvents);
    const [artifacts, setArtifacts] = useState<WorkspaceArtifact[]>([]);
    const [datasetProfiles, setDatasetProfiles] = useState<DatasetProfile[]>([]);
    const [activeDatasetProfile, setActiveDatasetProfile] = useState<DatasetProfile | null>(null);
    const [datasetQuery, setDatasetQuery] = useState('');
    const [datasetQueryResult, setDatasetQueryResult] = useState<DatasetQueryResult | null>(null);
    const [isQueryingDataset, setIsQueryingDataset] = useState(false);
    const [workspaceSearchQuery, setWorkspaceSearchQuery] = useState('');
    const [workspaceSearchResults, setWorkspaceSearchResults] = useState<WorkspaceSearchResult[]>([]);
    const [isSearchingWorkspace, setIsSearchingWorkspace] = useState(false);
    const [isSendingPrompt, setIsSendingPrompt] = useState(false);
    const [isCreatingReport, setIsCreatingReport] = useState(false);
    const [isSavingChatArtifact, setIsSavingChatArtifact] = useState(false);
    const [isProfilingDataset, setIsProfilingDataset] = useState(false);
    const [isEditingFile, setIsEditingFile] = useState(false);
    const [isPreviewingWrite, setIsPreviewingWrite] = useState(false);
    const [isApplyingWrite, setIsApplyingWrite] = useState(false);
    const [fileDraft, setFileDraft] = useState('');
    const [writeProposal, setWriteProposal] = useState<FileWriteProposal | null>(null);
    const [navigatorWidth, setNavigatorWidth] = useState(280);

    useEffect(() => {
        setSettingsDraft(llmSettings);
        setSettingsStatus(llmSettings.updatedAt ? 'LLM settings loaded from local config.' : 'LLM provider not connected yet.');
    }, [llmSettings]);

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

    function pushToolEvent(title: string, detail: string) {
        setLocalToolEvents((current) => [
            {time: new Date().toLocaleTimeString(), title, detail},
            ...current,
        ].slice(0, 12));
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
        clearFileWriteDraft();
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

    function startFileEdit() {
        if (!filePreview || filePreview.kind !== 'file') {
            setWorkspaceStatus('Select a text file before editing.');
            return;
        }
        setFileDraft(filePreview.content);
        setWriteProposal(null);
        setIsEditingFile(true);
    }

    function clearFileWriteDraft() {
        setIsEditingFile(false);
        setFileDraft('');
        setWriteProposal(null);
        setIsPreviewingWrite(false);
        setIsApplyingWrite(false);
    }

    async function previewFileWrite() {
        if (!workspace || !filePreview) {
            setWorkspaceStatus('Open a workspace and select a file before previewing writes.');
            return;
        }

        setIsPreviewingWrite(true);
        try {
            const proposal = await PreviewFileWrite({relPath: filePreview.relPath, content: fileDraft});
            setWriteProposal(proposal);
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

        setIsApplyingWrite(true);
        try {
            const proposal = await ApplyFileWrite({relPath: filePreview.relPath, content: fileDraft});
            const result = await RefreshWorkspace();
            if (result.selected) {
                onWorkspaceChange(result.snapshot);
                await selectWorkspaceFile(result.snapshot, proposal.relPath);
            }
            clearFileWriteDraft();
            setWorkspaceStatus(proposal.message);
            pushToolEvent('File write applied', proposal.relPath);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            setWorkspaceStatus(message || 'Could not apply file write.');
        } finally {
            setIsApplyingWrite(false);
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
        }
        await refreshChatHistory();
        await refreshArtifacts();
        await refreshDatasetProfiles();
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
        clearFileWriteDraft();
        setActiveFile(tab.relPath);
        setFilePreview(tab);
        setActiveDatasetProfile(datasetProfiles.find((profile) => profile.relPath === tab.relPath) ?? null);
    }

    function closeOpenTab(relPath: string) {
        const tabIndex = openTabs.findIndex((tab) => tab.relPath === relPath);
        if (tabIndex === -1) {
            return;
        }

        const nextTabs = openTabs.filter((tab) => tab.relPath !== relPath);
        setOpenTabs(nextTabs);
        if (activeFile !== relPath) {
            return;
        }

        clearFileWriteDraft();
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

    async function refreshDatasetProfiles() {
        try {
            const profiles = await ListDatasetProfiles();
            setDatasetProfiles(profiles);
            setActiveDatasetProfile((current) => profiles.find((profile) => profile.relPath === current?.relPath) ?? current);
        } catch {
            setDatasetProfiles([]);
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
        await sendPromptText(chatPrompt, true);
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
        await sendPromptText(prompt, false);
    }

    async function sendPromptText(rawPrompt: string, clearComposer: boolean) {
        const prompt = rawPrompt.trim();
        if (!prompt) {
            setChatStatus('Write a prompt before sending.');
            return;
        }

        const selectedContextRelPath = selectedTextContextRelPath();
        const contextPaths = contextPackPaths.length > 0 ? contextPackPaths : selectedContextRelPath ? [selectedContextRelPath] : [];
        const contextRelPath = contextPaths.length > 1 ? `pack: ${contextPaths.join(', ')}` : contextPaths[0] ?? '';
        const requestId = createRequestId();
        const userMessage: ChatMessage = {content: prompt, contextRelPath, createdAt: new Date().toISOString(), role: 'user'};
        const assistantMessage: ChatMessage = {content: '', contextRelPath, createdAt: new Date().toISOString(), role: 'assistant'};

        setIsSendingPrompt(true);
        setChatStatus(contextRelPath ? `Streaming with ${contextRelPath} as context...` : 'Streaming without selected file context...');
        setChatMessages((current) => [...current, userMessage, assistantMessage]);
        if (clearComposer) {
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
                replaceChatMessage(assistantMessage.createdAt, result.message, result.contextRelPath);
            }
            setChatStatus(result.contextRelPath ? `Answered with ${result.contextRelPath}.` : `Answered by ${result.model}.`);
            pushToolEvent('Chat completed', result.contextRelPath || result.model);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            if (message.includes('undefined') || message.includes('window')) {
                replaceChatMessage(assistantMessage.createdAt, 'Chat is available in the desktop runtime.', '');
                setChatStatus('Chat is available in the desktop runtime.');
                return;
            }
            replaceChatMessage(assistantMessage.createdAt, message || 'The provider did not return a usable chat response.', '');
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
            const report: MarkdownReport = await CreateChatMarkdownArtifact({
                title: latestAssistantArtifactTitle(chatMessages),
                content: latest.content,
                contextRelPath: latest.contextRelPath,
                source: 'NexusDesk chat',
            });
            const result = await RefreshWorkspace();
            if (result.selected) {
                onWorkspaceChange(result.snapshot);
                await refreshArtifacts();
                setExpandedDirectories((current) => reconcileExpandedDirectories(current, result.snapshot, findWorkspaceNode(result.snapshot, report.relPath)));
                await selectWorkspaceFile(result.snapshot, report.relPath);
                setWorkspaceStatus(`${report.name} saved in .nexusdesk/artifacts.`);
            }
            setChatStatus(`${report.name} saved as a Markdown artifact.`);
            pushToolEvent('Answer saved', report.relPath);
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
                    appendChatDelta(assistantCreatedAt, event.delta, event.contextRelPath || fallbackContextRelPath);
                }
                if (event.type === 'done') {
                    replaceChatMessage(assistantCreatedAt, event.message, event.contextRelPath || fallbackContextRelPath);
                }
                if (event.type === 'error') {
                    replaceChatMessage(assistantCreatedAt, event.message || 'Streaming response failed.', '');
                }
            });
        } catch {
            return null;
        }
    }

    function appendChatDelta(createdAt: string, delta: string, contextRelPath: string) {
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
            };
        }));
    }

    function replaceChatMessage(createdAt: string, content: string, contextRelPath: string) {
        setChatMessages((current) => current.map((message) => {
            if (message.createdAt !== createdAt) {
                return message;
            }
            return {
                ...message,
                content,
                contextRelPath,
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

            <WorkspaceNavigator
                activeFile={activeFile}
                buildStage={state.buildStage}
                expandedDirectories={expandedDirectories}
                isSearchingWorkspace={isSearchingWorkspace}
                isManagingRecent={isManagingRecent}
                isOpeningWorkspace={isOpeningWorkspace}
                isRefreshingWorkspace={isRefreshingWorkspace}
                onClearRecentWorkspaces={() => void clearRecentWorkspaces()}
                onClearWorkspaceSearch={() => setWorkspaceSearchResults([])}
                onCollapseAllDirectories={collapseAllDirectories}
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
                capabilities={state.capabilities}
                datasetProfiles={datasetProfiles}
                datasetQuery={datasetQuery}
                datasetQueryResult={datasetQueryResult}
                activeDatasetProfile={activeDatasetProfile}
                fileDraft={fileDraft}
                filePreview={filePreview}
                isCreatingReport={isCreatingReport}
                isProfilingDataset={isProfilingDataset}
                isQueryingDataset={isQueryingDataset}
                isEditingFile={isEditingFile}
                isApplyingWrite={isApplyingWrite}
                isLoadingPreview={isLoadingPreview}
                isPreviewingWrite={isPreviewingWrite}
                isSendingPrompt={isSendingPrompt}
                onApplyFileWrite={() => void applyFileWrite()}
                onCancelFileEdit={clearFileWriteDraft}
                onCreateReport={() => void createMarkdownReport()}
                onDatasetQueryChange={setDatasetQuery}
                onFileDraftChange={setFileDraft}
                onExplainContext={() => void explainSelectedContext()}
                onPinContext={pinSelectedContext}
                onPinProjectContext={pinProjectContext}
                onPreviewFileWrite={() => void previewFileWrite()}
                onProfileDataset={() => void profileSelectedDataset()}
                onQueryDataset={() => void querySelectedDataset()}
                onCloseTab={closeOpenTab}
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
                contextPackPaths={contextPackPaths}
                canSaveLatestAssistantArtifact={canSaveLatestAssistantArtifact}
                isSavingSettings={isSavingSettings}
                isSavingChatArtifact={isSavingChatArtifact}
                isSendingPrompt={isSendingPrompt}
                isTestingConnection={isTestingConnection}
                onChatPromptChange={setChatPrompt}
                onClearChatHistory={() => void clearChatHistory()}
                onClearContextPack={() => setContextPackPaths([])}
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

function latestAssistantMessage(messages: ChatMessage[]) {
    return [...messages].reverse().find((message) => message.role === 'assistant' && message.content.trim()) ?? null;
}

function latestAssistantArtifactTitle(messages: ChatMessage[]) {
    const assistantIndex = findLatestAssistantIndex(messages);
    if (assistantIndex === -1) {
        return 'Assistant response';
    }

    const prompt = messages
        .slice(0, assistantIndex)
        .reverse()
        .find((message) => message.role === 'user' && message.content.trim())?.content.trim();
    if (!prompt) {
        return 'Assistant response';
    }

    return `Assistant response - ${prompt.replace(/\s+/g, ' ').slice(0, 64)}`;
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
