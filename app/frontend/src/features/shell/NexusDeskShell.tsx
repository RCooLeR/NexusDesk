import {useEffect, useMemo, useState} from 'react';
import type {CSSProperties, MouseEvent as ReactMouseEvent} from 'react';
import {
    AskLLM,
    AskLLMStream,
    ClearRecentWorkspaces,
    ClearChatHistory,
    CreateMarkdownReport,
    GetChatHistory,
    GetRecentWorkspaces,
    OpenWorkspace,
    ReadWorkspaceFile,
    RemoveRecentWorkspace,
    RefreshWorkspace,
    SaveLLMSettings,
    SelectWorkspace,
    TestLLMConnection,
} from '../../../wailsjs/go/main/App';
import {EventsOn} from '../../../wailsjs/runtime/runtime';
import type {
    ChatStreamEvent,
    ChatMessage,
    FileNode,
    FilePreview,
    LLMChatResult,
    LLMProbeResult,
    LLMSettings,
    MarkdownReport,
    RecentWorkspace,
    StartupState,
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
    const [isSendingPrompt, setIsSendingPrompt] = useState(false);
    const [isCreatingReport, setIsCreatingReport] = useState(false);
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

        return workspace.nodes.filter((node) => isWorkspaceNodeVisible(node, expandedDirectories)).slice(0, 80);
    }, [expandedDirectories, workspace]);

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

        if (node.kind === 'directory') {
            setIsLoadingPreview(false);
            setFilePreview(createDirectoryPreview(node));
            return;
        }

        setFilePreview(null);
        setIsLoadingPreview(true);
        try {
            const preview = await ReadWorkspaceFile(node.relPath);
            setFilePreview(preview);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            if (message.includes('undefined') || message.includes('window')) {
                setFilePreview({
                    relPath: node.relPath,
                    name: node.name,
                    kind: 'unsupported',
                    fileType: node.fileType,
                    content: '',
                    encoding: '',
                    truncated: false,
                    message: 'File previews are available in the desktop runtime.',
                    size: 0,
                });
                return;
            }
            setFilePreview({
                relPath: node.relPath,
                name: node.name,
                kind: 'unsupported',
                fileType: node.fileType,
                content: '',
                encoding: '',
                truncated: false,
                message: message || 'Could not preview this file.',
                size: 0,
            });
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
            encoding: '',
            truncated: false,
            message: 'Select a file inside this folder to preview its contents.',
            size: 0,
        };
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

        const selectedNode = selectNodeAfterWorkspaceUpdate(result.snapshot);

        onWorkspaceChange(result.snapshot);
        await refreshChatHistory();
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
            setChatStatus('Select a text/code preview before asking for an explanation.');
            return;
        }

        const prompt = [
            `Explain ${filePreview?.relPath}.`,
            'Cover the file purpose, important structures, notable dependencies, risks, and practical next steps.',
        ].join(' ');
        await sendPromptText(prompt, false);
    }

    async function sendPromptText(rawPrompt: string, clearComposer: boolean) {
        const prompt = rawPrompt.trim();
        if (!prompt) {
            setChatStatus('Write a prompt before sending.');
            return;
        }

        const contextRelPath = selectedTextContextRelPath();
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
            const result: LLMChatResult = unsubscribe
                ? await AskLLMStream(prompt, contextRelPath, requestId)
                : await AskLLM(prompt, contextRelPath);
            if (workspace) {
                await refreshChatHistory();
            } else {
                replaceChatMessage(assistantMessage.createdAt, result.message, result.contextRelPath);
            }
            setChatStatus(result.contextRelPath ? `Answered with ${result.contextRelPath}.` : `Answered by ${result.model}.`);
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
        if (filePreview?.kind === 'file' && filePreview.content) {
            return filePreview.relPath;
        }
        return '';
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
                setExpandedDirectories((current) => reconcileExpandedDirectories(current, result.snapshot, findWorkspaceNode(result.snapshot, report.relPath)));
                await selectWorkspaceFile(result.snapshot, report.relPath);
                setWorkspaceStatus(`${report.name} created in .nexusdesk/artifacts.`);
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
                isManagingRecent={isManagingRecent}
                isOpeningWorkspace={isOpeningWorkspace}
                isRefreshingWorkspace={isRefreshingWorkspace}
                onClearRecentWorkspaces={() => void clearRecentWorkspaces()}
                onOpenWorkspace={() => void openWorkspace()}
                onRefreshWorkspace={() => void refreshWorkspace()}
                onRemoveRecentWorkspace={(recentWorkspace) => void removeRecentWorkspace(recentWorkspace)}
                onReopenWorkspace={(recentWorkspace) => void reopenWorkspace(recentWorkspace)}
                onSelectFallbackItem={selectFallbackItem}
                onSelectWorkspaceNode={(node) => void selectWorkspaceNode(node)}
                recentWorkspaces={recentWorkspaces}
                workspace={workspace}
                workspaceItems={state.workspaceItems}
                workspaceNodes={workspaceNodes}
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
                capabilities={state.capabilities}
                filePreview={filePreview}
                isCreatingReport={isCreatingReport}
                isLoadingPreview={isLoadingPreview}
                isSendingPrompt={isSendingPrompt}
                onCreateReport={() => void createMarkdownReport()}
                onExplainContext={() => void explainSelectedContext()}
                onRefreshPreview={() => void refreshSelectedPreview()}
                selectedMeta={selectedMeta}
                workspace={workspace}
            />

            <AgentPanel
                chatMessages={chatMessages}
                chatPrompt={chatPrompt}
                chatStatus={chatStatus}
                isSavingSettings={isSavingSettings}
                isSendingPrompt={isSendingPrompt}
                isTestingConnection={isTestingConnection}
                onChatPromptChange={setChatPrompt}
                onClearChatHistory={() => void clearChatHistory()}
                onSaveSettings={() => void saveLLMSettings()}
                onSendPrompt={() => void sendPrompt()}
                onSettingsDraftChange={updateSettingsDraft}
                onTestConnection={() => void testLLMConnection()}
                probeResult={probeResult}
                settingsDraft={settingsDraft}
                settingsStatus={settingsStatus}
                tagline={state.tagline}
                toolEvents={state.toolEvents}
            />
        </div>
    );
}

function findWorkspaceNode(snapshot: WorkspaceSnapshot, relPath: string) {
    return snapshot.nodes.find((node) => node.relPath === relPath) ?? null;
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
