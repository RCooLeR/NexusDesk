import {useEffect, useMemo, useState} from 'react';
import type {ChangeEvent} from 'react';
import {
    GetRecentWorkspaces,
    OpenWorkspace,
    RefreshWorkspace,
    SaveLLMSettings,
    SelectWorkspace,
    TestLLMConnection,
} from '../../../wailsjs/go/main/App';
import {brandAssets, capabilityIconByTitle, railItems, workspaceIconByName} from '../../brand/assets';
import type {
    FileNode,
    LLMProbeResult,
    LLMSettings,
    RecentWorkspace,
    StartupState,
    WorkspaceOpenResult,
    WorkspaceSnapshot,
} from '../../types';

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
    const [settingsDraft, setSettingsDraft] = useState<LLMSettings>(llmSettings);
    const [settingsStatus, setSettingsStatus] = useState('LLM provider not connected yet.');
    const [isSavingSettings, setIsSavingSettings] = useState(false);
    const [isTestingConnection, setIsTestingConnection] = useState(false);
    const [probeResult, setProbeResult] = useState<LLMProbeResult | null>(null);

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

    const workspaceNodes = useMemo(() => workspace?.nodes.slice(0, 80) ?? [], [workspace]);

    function selectFallbackItem(name: string) {
        setActiveFile(`${name}/`);
    }

    function selectWorkspaceNode(node: FileNode) {
        setActiveFile(node.relPath);
    }

    async function openWorkspace() {
        setIsOpeningWorkspace(true);
        setWorkspaceStatus('Waiting for folder selection...');

        try {
            const result = await SelectWorkspace();
            if (!applyWorkspaceResult(result, 'indexed')) {
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
            if (applyWorkspaceResult(result, 'indexed')) {
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
            if (!applyWorkspaceResult(result, 'refreshed')) {
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

    function applyWorkspaceResult(result: WorkspaceOpenResult, verb: 'indexed' | 'refreshed') {
        if (!result.selected) {
            return false;
        }

        onWorkspaceChange(result.snapshot);
        if (!result.snapshot.nodes.some((node) => node.relPath === activeFile)) {
            setActiveFile(result.snapshot.nodes[0]?.relPath ?? result.snapshot.name);
        }
        setWorkspaceStatus(`${result.snapshot.nodes.length} items ${verb} from ${result.snapshot.name}.`);
        return true;
    }

    async function refreshRecentWorkspaces() {
        try {
            onRecentWorkspacesChange(await GetRecentWorkspaces());
        } catch {
            onRecentWorkspacesChange([]);
        }
    }

    function updateSettingsDraft(field: keyof LLMSettings) {
        return (event: ChangeEvent<HTMLInputElement>) => {
            setSettingsDraft((current) => ({
                ...current,
                [field]: event.target.value,
            }));
        };
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
                    <button className="primary-action" onClick={openWorkspace} disabled={isOpeningWorkspace}>
                        {isOpeningWorkspace ? 'Opening...' : 'Open Folder'}
                    </button>
                    <button
                        className="icon-action"
                        title="Refresh workspace"
                        aria-label="Refresh workspace"
                        onClick={refreshWorkspace}
                        disabled={isRefreshingWorkspace}
                    >
                        R
                    </button>
                </div>

                <div className="tree-list">
                    {!workspace && recentWorkspaces.length > 0 && (
                        <div className="recent-list">
                            <div className="section-label">Recent</div>
                            {recentWorkspaces.slice(0, 4).map((recentWorkspace) => (
                                <button
                                    key={recentWorkspace.path}
                                    className="recent-item"
                                    onClick={() => reopenWorkspace(recentWorkspace)}
                                    disabled={isOpeningWorkspace}
                                >
                                    <strong>{recentWorkspace.name}</strong>
                                    <small>{recentWorkspace.path}</small>
                                </button>
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
                                    onClick={() => selectWorkspaceNode(node)}
                                    style={{paddingLeft: `${8 + Math.min(node.depth, 4) * 10}px`}}
                                >
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
                        <button>Preview</button>
                        <button>Explain</button>
                        <button>Report</button>
                    </div>
                </header>

                <section className="canvas-grid">
                    <article className="editor-pane">
                        <div className="pane-title">
                            <span>Source Preview</span>
                            <small>{selectedMeta}</small>
                        </div>
                        <div className="code-preview" aria-label="NexusDesk workflow preview">
                            <p><span>01</span>Open a workspace root.</p>
                            <p><span>02</span>Index files, datasets, docs, and metadata.</p>
                            <p><span>03</span>Ask the agent with selected source context.</p>
                            <p><span>04</span>Approve writes, Docker actions, and database mutations.</p>
                            <p><span>05</span>Save reports, charts, diffs, and generated configs as artifacts.</p>
                        </div>
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
                                    <span>{capability.status}</span>
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

                <div className="chat-card">
                    <div className="assistant-message">
                        <strong>NexusDesk</strong>
                        <p>Ready to connect a model, read selected files, and turn source context into auditable work.</p>
                    </div>
                    <div className="prompt-box">
                        <span>Ask about the workspace...</span>
                        <button title="Send prompt">Send</button>
                    </div>
                </div>

                <section className="settings-card">
                    <div className="pane-title">
                        <span>LLM Provider</span>
                        <small>Local config</small>
                    </div>
                    <div className="settings-form">
                        <label>
                            <span>Provider</span>
                            <input value={settingsDraft.providerName} onChange={updateSettingsDraft('providerName')} />
                        </label>
                        <label>
                            <span>Base URL</span>
                            <input value={settingsDraft.baseUrl} onChange={updateSettingsDraft('baseUrl')} />
                        </label>
                        <label>
                            <span>Model</span>
                            <input value={settingsDraft.model} onChange={updateSettingsDraft('model')} placeholder="Optional" />
                        </label>
                        <label>
                            <span>API Key</span>
                            <input
                                value={settingsDraft.apiKey}
                                onChange={updateSettingsDraft('apiKey')}
                                placeholder="Optional"
                                type="password"
                            />
                        </label>
                        <div className="settings-actions">
                            <small>{settingsStatus}</small>
                            <div className="settings-button-row">
                                <button onClick={testLLMConnection} disabled={isTestingConnection}>
                                    {isTestingConnection ? 'Testing...' : 'Test'}
                                </button>
                                <button onClick={saveLLMSettings} disabled={isSavingSettings}>
                                    {isSavingSettings ? 'Saving...' : 'Save'}
                                </button>
                            </div>
                        </div>
                        {probeResult && (
                            <div className={probeResult.ok ? 'probe-result ok' : 'probe-result failed'}>
                                <strong>{probeResult.ok ? 'Connection ready' : 'Connection issue'}</strong>
                                <span>{probeResult.endpoint}</span>
                                {probeResult.modelSample.length > 0 && <small>{probeResult.modelSample.join(', ')}</small>}
                            </div>
                        )}
                    </div>
                </section>

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
