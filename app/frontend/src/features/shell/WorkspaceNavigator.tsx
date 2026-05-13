import {brandAssets, workspaceIconByName} from '../../brand/assets';
import {Button, IconButton} from '../../components/ui';
import type {FileNode, RecentWorkspace, WorkspaceItem, WorkspaceSnapshot} from '../../types';

type WorkspaceNavigatorProps = {
    activeFile: string;
    buildStage: string;
    expandedDirectories: Set<string>;
    isManagingRecent: boolean;
    isOpeningWorkspace: boolean;
    isRefreshingWorkspace: boolean;
    onClearRecentWorkspaces: () => void;
    onOpenWorkspace: () => void;
    onRefreshWorkspace: () => void;
    onRemoveRecentWorkspace: (workspace: RecentWorkspace) => void;
    onReopenWorkspace: (workspace: RecentWorkspace) => void;
    onSelectFallbackItem: (name: string) => void;
    onSelectWorkspaceNode: (node: FileNode) => void;
    recentWorkspaces: RecentWorkspace[];
    workspace: WorkspaceSnapshot | null;
    workspaceItems: WorkspaceItem[];
    workspaceNodes: FileNode[];
    workspaceStatus: string;
};

const fileIconByType: Record<string, string> = {
    code: brandAssets.icons.code,
    data: brandAssets.icons.data,
    document: brandAssets.icons.documents,
    image: brandAssets.icons.documents,
    folder: brandAssets.icons.documents,
    file: brandAssets.icons.documents,
};

export function WorkspaceNavigator({
    activeFile,
    buildStage,
    expandedDirectories,
    isManagingRecent,
    isOpeningWorkspace,
    isRefreshingWorkspace,
    onClearRecentWorkspaces,
    onOpenWorkspace,
    onRefreshWorkspace,
    onRemoveRecentWorkspace,
    onReopenWorkspace,
    onSelectFallbackItem,
    onSelectWorkspaceNode,
    recentWorkspaces,
    workspace,
    workspaceItems,
    workspaceNodes,
    workspaceStatus,
}: WorkspaceNavigatorProps) {
    return (
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
                <span>{buildStage}</span>
            </header>

            <div className="action-row">
                <Button className="primary-action" onClick={onOpenWorkspace} disabled={isOpeningWorkspace} variant="primary">
                    {isOpeningWorkspace ? 'Opening...' : 'Open Folder'}
                </Button>
                <IconButton
                    className="icon-action"
                    label="Refresh workspace"
                    onClick={onRefreshWorkspace}
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
                            <Button onClick={onClearRecentWorkspaces} disabled={isManagingRecent} variant="subtle">Clear</Button>
                        </div>
                        {recentWorkspaces.slice(0, 4).map((recentWorkspace) => (
                            <div className="recent-row" key={recentWorkspace.path}>
                                <button
                                    className="recent-item"
                                    onClick={() => onReopenWorkspace(recentWorkspace)}
                                    disabled={isOpeningWorkspace}
                                >
                                    <strong>{recentWorkspace.name}</strong>
                                    <small>{recentWorkspace.path}</small>
                                </button>
                                <Button
                                    className="recent-remove"
                                    onClick={() => onRemoveRecentWorkspace(recentWorkspace)}
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
                                onClick={() => onSelectWorkspaceNode(node)}
                                style={{paddingLeft: `${8 + Math.min(node.depth, 10) * 8}px`}}
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
                        {workspaceItems.map((item) => (
                            <button
                                key={item.name}
                                className={activeFile.startsWith(item.name) ? 'tree-item selected' : 'tree-item'}
                                onClick={() => onSelectFallbackItem(item.name)}
                            >
                                <span className="tree-disclosure" />
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
    );
}
