import {brandAssets, workspaceIconByName} from '../../brand/assets';
import {Button, IconButton} from '../../components/ui';
import type {FileNode, RecentWorkspace, WorkspaceItem, WorkspaceSearchResult, WorkspaceSnapshot} from '../../types';

type WorkspaceNavigatorProps = {
    activeFile: string;
    buildStage: string;
    expandedDirectories: Set<string>;
    isSearchingWorkspace: boolean;
    isManagingRecent: boolean;
    isOpeningWorkspace: boolean;
    isRefreshingWorkspace: boolean;
    onClearRecentWorkspaces: () => void;
    onClearWorkspaceSearch: () => void;
    onCollapseAllDirectories: () => void;
    onExpandAllDirectories: () => void;
    onOpenWorkspace: () => void;
    onRefreshWorkspace: () => void;
    onRemoveRecentWorkspace: (workspace: RecentWorkspace) => void;
    onReopenWorkspace: (workspace: RecentWorkspace) => void;
    onSearchWorkspace: () => void;
    onSelectFallbackItem: (name: string) => void;
    onSelectSearchResult: (result: WorkspaceSearchResult) => void;
    onSelectWorkspaceNode: (node: FileNode) => void;
    onWorkspaceSearchQueryChange: (value: string) => void;
    recentWorkspaces: RecentWorkspace[];
    workspace: WorkspaceSnapshot | null;
    workspaceItems: WorkspaceItem[];
    workspaceNodes: FileNode[];
    workspaceSearchQuery: string;
    workspaceSearchResults: WorkspaceSearchResult[];
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
    isSearchingWorkspace,
    isManagingRecent,
    isOpeningWorkspace,
    isRefreshingWorkspace,
    onClearRecentWorkspaces,
    onClearWorkspaceSearch,
    onCollapseAllDirectories,
    onExpandAllDirectories,
    onOpenWorkspace,
    onRefreshWorkspace,
    onRemoveRecentWorkspace,
    onReopenWorkspace,
    onSearchWorkspace,
    onSelectFallbackItem,
    onSelectSearchResult,
    onSelectWorkspaceNode,
    onWorkspaceSearchQueryChange,
    recentWorkspaces,
    workspace,
    workspaceItems,
    workspaceNodes,
    workspaceSearchQuery,
    workspaceSearchResults,
    workspaceStatus,
}: WorkspaceNavigatorProps) {
    return (
        <section className="navigator">
            <header className="navigator-header">
                <div className="product-lockup" aria-label="NexusDesk">
                    <img src={brandAssets.symbolDark} alt="" />
                    <div>
                        <h1><span>Nexus</span><strong>Desk</strong></h1>
                        <small>AI IDE, Data &amp; Analytics Studio</small>
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
                            <small>{workspace.truncated ? `${workspace.nodes.length} indexed items; scan capped for responsiveness.` : workspaceStatus}</small>
                        </div>
                        <div className="tree-tools">
                            <div className="workspace-search">
                                <input
                                    aria-label="Search workspace"
                                    onChange={(event) => onWorkspaceSearchQueryChange(event.target.value)}
                                    onKeyDown={(event) => {
                                        if (event.key === 'Enter') {
                                            onSearchWorkspace();
                                        }
                                    }}
                                    placeholder="Search files..."
                                    value={workspaceSearchQuery}
                                />
                                <Button disabled={isSearchingWorkspace || workspaceSearchQuery.trim() === ''} onClick={onSearchWorkspace} variant="subtle">
                                    {isSearchingWorkspace ? 'Searching...' : 'Search'}
                                </Button>
                            </div>
                            <div className="tree-tool-row">
                                <Button onClick={onExpandAllDirectories} variant="subtle">Expand all</Button>
                                <Button onClick={onCollapseAllDirectories} variant="subtle">Collapse all</Button>
                                {workspaceSearchResults.length > 0 && <Button onClick={onClearWorkspaceSearch} variant="subtle">Clear results</Button>}
                            </div>
                        </div>
                        {workspaceSearchResults.length > 0 && (
                            <div className="search-results">
                                <div className="section-label">{workspaceSearchResults.length} matches</div>
                                {workspaceSearchResults.map((result, index) => (
                                    <button className="search-result" key={`${result.relPath}-${result.matchType}-${index}`} onClick={() => onSelectSearchResult(result)}>
                                        <strong>{result.relPath}</strong>
                                        <small>{result.matchType}{result.line > 0 ? `, line ${result.line}` : ''}</small>
                                        <span>{result.snippet}</span>
                                    </button>
                                ))}
                            </div>
                        )}
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
