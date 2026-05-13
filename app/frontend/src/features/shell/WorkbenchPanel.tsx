import {brandAssets, capabilityIconByTitle} from '../../brand/assets';
import {Button, EmptyState, InlineAlert, LoadingState, StatusBadge} from '../../components/ui';
import type {Capability, FilePreview, FileWriteProposal, TablePreview, WorkspaceArtifact, WorkspaceSnapshot} from '../../types';
import {HighlightedCode} from './HighlightedCode';

type WorkbenchPanelProps = {
    activeFile: string;
    artifacts: WorkspaceArtifact[];
    capabilities: Capability[];
    fileDraft: string;
    filePreview: FilePreview | null;
    isApplyingWrite: boolean;
    isEditingFile: boolean;
    isSendingPrompt: boolean;
    isCreatingReport: boolean;
    isLoadingPreview: boolean;
    isPreviewingWrite: boolean;
    onApplyFileWrite: () => void;
    onCancelFileEdit: () => void;
    onExplainContext: () => void;
    onCreateReport: () => void;
    onFileDraftChange: (content: string) => void;
    onPreviewFileWrite: () => void;
    onSelectArtifact: (artifact: WorkspaceArtifact) => void;
    onRefreshPreview: () => void;
    onStartFileEdit: () => void;
    selectedMeta: string;
    writeProposal: FileWriteProposal | null;
    workspace: WorkspaceSnapshot | null;
};

export function WorkbenchPanel({
    activeFile,
    artifacts,
    capabilities,
    fileDraft,
    filePreview,
    isApplyingWrite,
    isEditingFile,
    isSendingPrompt,
    isCreatingReport,
    isLoadingPreview,
    isPreviewingWrite,
    onApplyFileWrite,
    onCancelFileEdit,
    onExplainContext,
    onCreateReport,
    onFileDraftChange,
    onPreviewFileWrite,
    onSelectArtifact,
    onRefreshPreview,
    onStartFileEdit,
    selectedMeta,
    writeProposal,
    workspace,
}: WorkbenchPanelProps) {
    const canExplainContext = Boolean(workspace && filePreview?.kind === 'file' && filePreview.content);
    const canEditContext = Boolean(workspace && filePreview?.kind === 'file' && filePreview.content && !filePreview.table);

    return (
        <main className="workbench">
            <header className="topbar">
                <div>
                    <p className="eyebrow">Active Context</p>
                    <h2>{activeFile}</h2>
                </div>
                <div className="topbar-actions">
                    <Button disabled={!workspace || isLoadingPreview} onClick={onRefreshPreview}>
                        {isLoadingPreview ? 'Loading...' : 'Preview'}
                    </Button>
                    <Button disabled={!canExplainContext || isSendingPrompt} onClick={onExplainContext}>
                        {isSendingPrompt ? 'Sending...' : 'Explain'}
                    </Button>
                    <Button disabled={!canEditContext || isLoadingPreview} onClick={onStartFileEdit}>
                        Edit
                    </Button>
                    <Button disabled={!workspace || isCreatingReport} onClick={onCreateReport}>
                        {isCreatingReport ? 'Creating...' : 'Report'}
                    </Button>
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
                            ) : filePreview?.kind === 'image' && filePreview.content ? (
                                <>
                                    {filePreview.message && <InlineAlert>{filePreview.message}</InlineAlert>}
                                    <div className="image-preview">
                                        <img src={filePreview.content} alt={filePreview.name} />
                                    </div>
                                </>
                            ) : filePreview?.kind === 'pdf' && filePreview.content ? (
                                <>
                                    {filePreview.message && <InlineAlert>{filePreview.message}</InlineAlert>}
                                    <div className="document-preview">
                                        <iframe src={filePreview.content} title={filePreview.name} />
                                    </div>
                                </>
                            ) : filePreview?.table ? (
                                <>
                                    {filePreview.message && <InlineAlert>{filePreview.message}</InlineAlert>}
                                    <CsvTablePreview table={filePreview.table} />
                                </>
                            ) : isEditingFile ? (
                                <FileWriteEditor
                                    draft={fileDraft}
                                    isApplying={isApplyingWrite}
                                    isPreviewing={isPreviewingWrite}
                                    onApply={onApplyFileWrite}
                                    onCancel={onCancelFileEdit}
                                    onChange={onFileDraftChange}
                                    onPreview={onPreviewFileWrite}
                                    proposal={writeProposal}
                                />
                            ) : filePreview?.content ? (
                                <>
                                    {filePreview.message && <InlineAlert>{filePreview.message}</InlineAlert>}
                                    <HighlightedCode content={filePreview.content} fileName={filePreview.name} />
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
                        <span>{workspace ? 'Artifacts' : 'MVP Capabilities'}</span>
                        <small>{workspace ? `${artifacts.length} generated` : 'Phase 1 focus'}</small>
                    </div>
                    {workspace ? (
                        <div className="artifact-list">
                            {artifacts.length === 0 ? (
                                <EmptyState
                                    detail="Create a report to add the first workspace artifact."
                                    iconSrc={brandAssets.icons.documents}
                                    title="No artifacts yet"
                                />
                            ) : artifacts.map((artifact) => (
                                <button className="artifact-item" key={artifact.relPath} onClick={() => onSelectArtifact(artifact)}>
                                    <img src={brandAssets.icons.documents} alt="" />
                                    <span>
                                        <strong>{artifact.name}</strong>
                                        <small>{artifact.relPath}</small>
                                    </span>
                                    <StatusBadge tone="warning">{artifact.kind}</StatusBadge>
                                </button>
                            ))}
                        </div>
                    ) : (
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
                    )}
                </article>
            </section>
        </main>
    );
}

function FileWriteEditor({
    draft,
    isApplying,
    isPreviewing,
    onApply,
    onCancel,
    onChange,
    onPreview,
    proposal,
}: {
    draft: string;
    isApplying: boolean;
    isPreviewing: boolean;
    onApply: () => void;
    onCancel: () => void;
    onChange: (content: string) => void;
    onPreview: () => void;
    proposal: FileWriteProposal | null;
}) {
    return (
        <div className="file-write-editor">
            <div className="write-toolbar">
                <Button disabled={isPreviewing || isApplying} onClick={onPreview}>
                    {isPreviewing ? 'Previewing...' : 'Preview diff'}
                </Button>
                <Button disabled={!proposal || isApplying} onClick={onApply}>
                    {isApplying ? 'Applying...' : 'Apply'}
                </Button>
                <Button disabled={isApplying} onClick={onCancel} variant="subtle">
                    Cancel
                </Button>
            </div>
            <textarea
                aria-label="File write draft"
                onChange={(event) => onChange(event.target.value)}
                spellCheck={false}
                value={draft}
            />
            {proposal && (
                <div className="write-diff">
                    <InlineAlert>{proposal.message}</InlineAlert>
                    <HighlightedCode content={proposal.diff} fileName={`${proposal.name}.diff`} />
                </div>
            )}
        </div>
    );
}

function CsvTablePreview({table}: {table: TablePreview}) {
    return (
        <div className="csv-preview" aria-label="CSV table preview">
            {table.profiles.length > 0 && (
                <div className="csv-profile-strip" aria-label="CSV column profile">
                    {table.profiles.map((profile, index) => (
                        <div className="csv-profile" key={`${profile.name}-${index}`}>
                            <strong>{profile.name || `Column ${index + 1}`}</strong>
                            <span>{profile.type}</span>
                            <small>
                                {profile.distinct} distinct
                                {profile.missing > 0 ? `, ${profile.missing} missing` : ''}
                                {profile.min && profile.max ? `, ${profile.min}-${profile.max}` : ''}
                            </small>
                        </div>
                    ))}
                </div>
            )}
            <table>
                <thead>
                    <tr>
                        {table.columns.map((column, index) => (
                            <th key={`${column}-${index}`}>{column || `Column ${index + 1}`}</th>
                        ))}
                    </tr>
                </thead>
                <tbody>
                    {table.rows.map((row, rowIndex) => (
                        <tr key={rowIndex}>
                            {table.columns.map((_, columnIndex) => (
                                <td key={columnIndex}>{row[columnIndex] ?? ''}</td>
                            ))}
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}
