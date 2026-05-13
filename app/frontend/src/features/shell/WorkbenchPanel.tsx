import {brandAssets, capabilityIconByTitle} from '../../brand/assets';
import {Button, EmptyState, InlineAlert, LoadingState, StatusBadge} from '../../components/ui';
import type {Capability, FilePreview, WorkspaceSnapshot} from '../../types';

type WorkbenchPanelProps = {
    activeFile: string;
    capabilities: Capability[];
    filePreview: FilePreview | null;
    isLoadingPreview: boolean;
    selectedMeta: string;
    workspace: WorkspaceSnapshot | null;
};

export function WorkbenchPanel({
    activeFile,
    capabilities,
    filePreview,
    isLoadingPreview,
    selectedMeta,
    workspace,
}: WorkbenchPanelProps) {
    return (
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
                        {capabilities.map((capability) => (
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
    );
}
