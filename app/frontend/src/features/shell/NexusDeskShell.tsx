import {useEffect, useMemo, useState} from 'react';
import {
    AskLLM,
    ClearRecentWorkspaces,
    ClearChatHistory,
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
import {brandAssets, capabilityIconByTitle, railItems, workspaceIconByName} from '../../brand/assets';
import {Button, EmptyState, IconButton, InlineAlert, LoadingState, StatusBadge} from '../../components/ui';
import type {
    ChatMessage,
    FileNode,
    FilePreview,
    LLMChatResult,
    LLMProbeResult,
    LLMSettings,
    RecentWorkspace,
    StartupState,
    WorkspaceOpenResult,
    WorkspaceSnapshot,
} from '../../types';
import {AgentChatCard} from './AgentChatCard';
import {LLMSettingsCard} from './LLMSettingsCard';

type NexusDeskShellProps = {
    state: StartupState;
    workspace: WorkspaceSnapshot | null;
    recentWorkspaces: RecentWorkspace[];
    llmSettings: LLMSettings;
    onWorkspaceChange: (workspace: WorkspaceSnapshot) => void;
    onRecentWorkspacesChange: (workspaces: RecentWorkspace[]) => void;
    onLLMSettingsChange: (settings: LLMSettings) => void;
};

const fileIconByType: Record<string, string> = {
    code: brandAssets.icons.code,
    data: brandAssets.icons.data,
    document: brandAssets.icons.documents,
    image: brandAssets.icons.documents,
    folder: brandAssets.icons.documents,
    file: brandAssets.icons.documents,
};

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

    useEffect(() => {
        setSettingsDraft(llmSettings);
        setSettingsStatus(llmSettings.updatedAt ? 'LLM settings loaded from local config.' : 'LLM provider not connected yet.');
    }, [llmSettings]);

    const selectedMeta = useMemo(() => {
        if (workspace) {
            return workspace.nodes.find((node) => node.relPath === activeFile)?.meta ?? workspace.root;
        }

        return state.workspaceItems.find((item) => activeFile.startsWith(item.name))?.meta ?? 'Selected planning source';
    }, [activeFile, state.workspaceItems, workspace]);

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
        const prompt = chatPrompt.trim();
        if (!prompt) {
            setChatStatus('Write a prompt before sending.');
            return;
        }

        const contextRelPath = filePreview?.content ? filePreview.relPath : '';
        setIsSendingPrompt(true);
        setChatStatus(contextRelPath ? `Sending with ${contextRelPath} as context...` : 'Sending without selected file context...');
        setChatMessages((current) => [...current, {content: prompt, contextRelPath, createdAt: new Date().toISOString(), role: 'user'}]);
        setChatPrompt('');

        try {
            const result: LLMChatResult = await AskLLM(prompt, contextRelPath);
            if (workspace) {
                await refreshChatHistory();
            } else {
                setChatMessages((current) => [
                    ...current,
                    {
                        content: result.message,
                        contextRelPath: result.contextRelPath,
                        createdAt: new Date().toISOString(),
                        role: 'assistant',
                    },
                ]);
            }
            setChatStatus(result.contextRelPath ? `Answered with ${result.contextRelPath}.` : `Answered by ${result.model}.`);
        } catch (error) {
            const message = error instanceof Error ? error.message : '';
            if (message.includes('undefined') || message.includes('window')) {
                setChatStatus('Chat is available in the desktop runtime.');
                return;
            }
            setChatMessages((current) => [
                ...current,
                {
                    content: message || 'The provider did not return a usable chat response.',
                    contextRelPath: '',
                    createdAt: new Date().toISOString(),
                    role: 'assistant',
                },
            ]);
            setChatStatus(message || 'Chat request failed.');
        } finally {
            setIsSendingPrompt(false);
        }
    }

    return (
        <div className="app-shell">
            <aside className="workspace-rail">
                <div className="brand-mark" aria-label="NexusDesk">
                    <img src={brandAssets.symbolSilver} alt="" />
                </div>
                {railItems.map((item) => (
                    <button
                        key={item.label}
                        className={item.active ? 'rail-button active' : 'rail-button'}
                        title={item.label}
                        aria-label={item.label}
                    >
                        <img src={item.icon} alt="" />
                    </button>
                ))}
            </aside>

            <section className="navigator">
                <header className="navigator-header">
                    <div className="product-lockup" aria-label="NexusDesk">
                        <img src={brandAssets.symbolDark} alt="" />
                        <div>
                            <h1><span>Nexus</span><strong>Desk</strong></h1>
                            <small>AI Workbench for Code, Data &amp; Ops</small>
                        </div>
                    </div>
                    <p className="eyebrow">Workspace</p>
                    <span>{state.buildStage}</span>
                </header>

                <div className="action-row">
                    <Button className="primary-action" onClick={openWorkspace} disabled={isOpeningWorkspace} variant="primary">
                        {isOpeningWorkspace ? 'Opening...' : 'Open Folder'}
                    </Button>
                    <IconButton
                        className="icon-action"
                        label="Refresh workspace"
                        onClick={refreshWorkspace}
                        disabled={isRefreshingWorkspace}
                    >
                        R
                    </IconButton>
                </div>

                <div className="tree-list">
                    {!workspace && recentWorkspaces.length > 0 && (
                        <div className="recent-list">
                            <div className="recent-list-header">
                                <div className="section-label">Recent</div>
                                <Button onClick={clearRecentWorkspaces} disabled={isManagingRecent} variant="subtle">Clear</Button>
                            </div>
                            {recentWorkspaces.slice(0, 4).map((recentWorkspace) => (
                                <div className="recent-row" key={recentWorkspace.path}>
                                    <button
                                        className="recent-item"
                                        onClick={() => reopenWorkspace(recentWorkspace)}
                                        disabled={isOpeningWorkspace}
                                    >
                                        <strong>{recentWorkspace.name}</strong>
                                        <small>{recentWorkspace.path}</small>
                                    </button>
                                    <Button
                                        className="recent-remove"
                                        onClick={() => void removeRecentWorkspace(recentWorkspace)}
                                        disabled={isManagingRecent}
                                        variant="subtle"
                                    >
                                        Remove
                                    </Button>
                                </div>
                            ))}
                        </div>
                    )}

                    {workspace ? (
                        <>
                            <div className="workspace-summary">
                                <strong>{workspace.name}</strong>
                                <small>{workspace.truncated ? 'Showing first indexed items' : workspaceStatus}</small>
                            </div>
                            {workspaceNodes.map((node) => (
                                <button
                                    key={node.relPath}
                                    className={activeFile === node.relPath ? 'tree-item selected' : 'tree-item'}
                                    onClick={() => void selectWorkspaceNode(node)}
                                    style={{paddingLeft: `${8 + Math.min(node.depth, 4) * 10}px`}}
                                >
                                    <span className="tree-disclosure">
                                        {node.kind === 'directory' ? (expandedDirectories.has(node.relPath) ? '-' : '+') : ''}
                                    </span>
                                    <span className={`file-glyph ${node.kind}`}>
                                        <img src={fileIconByType[node.fileType] ?? brandAssets.icons.documents} alt="" />
                                    </span>
                                    <span>
                                        <strong>{node.name}</strong>
                                        <small>{node.meta}</small>
                                    </span>
                                </button>
                            ))}
                        </>
                    ) : (
                        <>
                            <div className="workspace-summary">
                                <strong>Scaffold preview</strong>
                                <small>{workspaceStatus}</small>
                            </div>
                            {state.workspaceItems.map((item) => (
                                <button
                                    key={item.name}
                                    className={activeFile.startsWith(item.name) ? 'tree-item selected' : 'tree-item'}
                                    onClick={() => selectFallbackItem(item.name)}
                                >
                                    <span className={`file-glyph ${item.kind}`}>
                                        <img src={workspaceIconByName[item.name] ?? brandAssets.icons.documents} alt="" />
                                    </span>
                                    <span>
                                        <strong>{item.name}</strong>
                                        <small>{item.meta}</small>
                                    </span>
                                </button>
                            ))}
                        </>
                    )}
                </div>
            </section>

            <main className="workbench">
                <header className="topbar">
                    <div>
                        <p className="eyebrow">Active Context</p>
                        <h2>{activeFile}</h2>
                    </div>
                    <div className="topbar-actions">
                        <Button>Preview</Button>
                        <Button>Explain</Button>
                        <Button>Report</Button>
                    </div>
                </header>

                <section className="canvas-grid">
                    <article className="editor-pane">
                        <div className="pane-title">
                            <span>Source Preview</span>
                            <small>{selectedMeta}</small>
                        </div>
                        {workspace ? (
                            <div className="file-preview" aria-label="Workspace file preview">
                                {isLoadingPreview ? (
                                    <LoadingState
                                        detail="Reading the selected file inside the approved workspace root."
                                        iconSrc={brandAssets.icons.documents}
                                        title="Loading preview"
                                    />
                                ) : filePreview?.content ? (
                                    <>
                                        {filePreview.message && <InlineAlert>{filePreview.message}</InlineAlert>}
                                        <pre>{filePreview.content}</pre>
                                    </>
                                ) : (
                                    <EmptyState
                                        detail={filePreview?.message ?? 'Select a file from the workspace tree to preview it here.'}
                                        iconSrc={brandAssets.icons.documents}
                                        title={filePreview?.kind === 'unsupported' ? 'Preview unavailable' : 'No file selected'}
                                        tone={filePreview?.kind === 'unsupported' ? 'warning' : 'neutral'}
                                    />
                                )}
                            </div>
                        ) : (
                            <div className="code-preview" aria-label="NexusDesk workflow preview">
                                <p><span>01</span>Open a workspace root.</p>
                                <p><span>02</span>Index files, datasets, docs, and metadata.</p>
                                <p><span>03</span>Ask the agent with selected source context.</p>
                                <p><span>04</span>Approve writes, Docker actions, and database mutations.</p>
                                <p><span>05</span>Save reports, charts, diffs, and generated configs as artifacts.</p>
                            </div>
                        )}
                    </article>

                    <article className="status-pane">
                        <div className="pane-title">
                            <span>MVP Capabilities</span>
                            <small>Phase 1 focus</small>
                        </div>
                        <div className="capability-list">
                            {state.capabilities.map((capability) => (
                                <div className="capability-card" key={capability.title}>
                                    <img src={capabilityIconByTitle[capability.title] ?? brandAssets.icons.ai} alt="" />
                                    <strong>{capability.title}</strong>
                                    <p>{capability.description}</p>
                                    <StatusBadge tone="warning">{capability.status}</StatusBadge>
                                </div>
                            ))}
                        </div>
                    </article>
                </section>
            </main>

            <aside className="agent-panel">
                <header>
                    <img className="agent-symbol" src={brandAssets.symbolDark} alt="" />
                    <p className="eyebrow">Agent</p>
                    <h2>Grounded Assistant</h2>
                    <span>{state.tagline}</span>
                </header>

                <AgentChatCard
                    chatMessages={chatMessages}
                    chatPrompt={chatPrompt}
                    chatStatus={chatStatus}
                    isSendingPrompt={isSendingPrompt}
                    onChatPromptChange={setChatPrompt}
                    onClearChatHistory={() => void clearChatHistory()}
                    onSendPrompt={() => void sendPrompt()}
                />

                <LLMSettingsCard
                    isSavingSettings={isSavingSettings}
                    isTestingConnection={isTestingConnection}
                    onSaveSettings={() => void saveLLMSettings()}
                    onSettingsDraftChange={updateSettingsDraft}
                    onTestConnection={() => void testLLMConnection()}
                    probeResult={probeResult}
                    settingsDraft={settingsDraft}
                    settingsStatus={settingsStatus}
                />

                <section className="timeline">
                    <div className="pane-title">
                        <span>Tool Timeline</span>
                        <small>Visible by design</small>
                    </div>
                    {state.toolEvents.map((event) => (
                        <div className="timeline-item" key={`${event.time}-${event.title}`}>
                            <time>{event.time}</time>
                            <strong>{event.title}</strong>
                            <p>{event.detail}</p>
                        </div>
                    ))}
                </section>
            </aside>
        </div>
    );
}
