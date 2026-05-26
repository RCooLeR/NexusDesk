import {useEffect, useState} from 'react';
import type {CSSProperties, MouseEvent as ReactMouseEvent} from 'react';
import type {IconDefinition} from '@fortawesome/fontawesome-svg-core';
import {
    faBan,
    faBroom,
    faChevronRight,
    faCompress,
    faCopy,
    faExpand,
    faFileCirclePlus,
    faFloppyDisk,
    faFolderOpen,
    faFolderPlus,
    faMagnifyingGlass,
    faPaste,
    faPen,
    faRotateRight,
    faScissors,
    faTrash,
    faUpRightFromSquare,
} from '@fortawesome/free-solid-svg-icons';
import {FontAwesomeIcon} from '@fortawesome/react-fontawesome';
import {brandAssets, workspaceIconByName} from '../../brand/assets';
import {Button, IconButton} from '../../components/ui';
import type {FileNode, GitFileChange, RecentWorkspace, WorkspaceItem, WorkspaceSearchResult, WorkspaceSnapshot} from '../../types';

type WorkspaceNavigatorProps = {
    activeFile: string;
    buildStage: string;
    expandedDirectories: Set<string>;
    isSearchingWorkspace: boolean;
    isManagingRecent: boolean;
    isOpeningWorkspace: boolean;
    isRefreshingWorkspace: boolean;
    isCreatingScanReport: boolean;
    onClearRecentWorkspaces: () => void;
    onClearWorkspaceSearch: () => void;
    onCollapseAllDirectories: () => void;
    onCreateScanReport: () => void;
    onExpandAllDirectories: () => void;
    onOpenWorkspace: () => void;
    onRefreshWorkspace: () => void;
    onRemoveRecentWorkspace: (workspace: RecentWorkspace) => void;
    onReopenWorkspace: (workspace: RecentWorkspace) => void;
    onSearchWorkspace: () => void;
    onSelectFallbackItem: (name: string) => void;
    onSelectSearchResult: (result: WorkspaceSearchResult) => void;
    onTreeContextAction: (action: TreeContextAction, node: FileNode) => void;
    onSelectWorkspaceNode: (node: FileNode) => void;
    onWorkspaceSearchQueryChange: (value: string) => void;
    recentWorkspaces: RecentWorkspace[];
    workspace: WorkspaceSnapshot | null;
    workspaceItems: WorkspaceItem[];
    workspaceNodes: FileNode[];
    workspaceSearchQuery: string;
    workspaceSearchResults: WorkspaceSearchResult[];
    workspaceStatus: string;
    gitFileChanges: GitFileChange[];
    treeClipboard: TreeClipboardState | null;
};

export type TreeContextAction = 'new-file' | 'new-folder' | 'rename' | 'move' | 'delete' | 'cut' | 'copy' | 'paste' | 'copy-path' | 'reveal';
export type TreeClipboardState = {
    mode: 'copy' | 'cut';
    sourceRelPath: string;
};

const fileIconByType: Record<string, IconDefinition> = {
    code: brandAssets.icons.code,
    data: brandAssets.icons.data,
    document: brandAssets.icons.documents,
    image: brandAssets.icons.documents,
    folder: brandAssets.icons.workspace,
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
    isCreatingScanReport,
    onClearRecentWorkspaces,
    onClearWorkspaceSearch,
    onCollapseAllDirectories,
    onCreateScanReport,
    onExpandAllDirectories,
    onOpenWorkspace,
    onRefreshWorkspace,
    onRemoveRecentWorkspace,
    onReopenWorkspace,
    onSearchWorkspace,
    onSelectFallbackItem,
    onSelectSearchResult,
    onTreeContextAction,
    onSelectWorkspaceNode,
    onWorkspaceSearchQueryChange,
    recentWorkspaces,
    workspace,
    workspaceItems,
    workspaceNodes,
    workspaceSearchQuery,
    workspaceSearchResults,
    workspaceStatus,
    gitFileChanges,
    treeClipboard,
}: WorkspaceNavigatorProps) {
    const gitStatusByPath = buildGitStatusMap(gitFileChanges);
    const [contextMenu, setContextMenu] = useState<{node: FileNode; x: number; y: number} | null>(null);
    const [showIgnoredSamples, setShowIgnoredSamples] = useState(false);

    useEffect(() => {
        if (!contextMenu) {
            return;
        }
        function closeContextMenuOnEscape(event: KeyboardEvent) {
            if (event.key === 'Escape') {
                setContextMenu(null);
            }
        }
        window.addEventListener('keydown', closeContextMenuOnEscape);
        return () => window.removeEventListener('keydown', closeContextMenuOnEscape);
    }, [contextMenu]);

    function openContextMenu(event: ReactMouseEvent, node: FileNode) {
        event.preventDefault();
        setContextMenu({node, x: event.clientX, y: event.clientY});
    }
    function runContextAction(action: TreeContextAction) {
        if (!contextMenu) {
            return;
        }
        onTreeContextAction(action, contextMenu.node);
        setContextMenu(null);
    }
    return (
        <section className="navigator" onClick={() => setContextMenu(null)}>
            <div className="action-row">
                <Button className="primary-action" onClick={onOpenWorkspace} disabled={isOpeningWorkspace} variant="primary">
                    <FontAwesomeIcon icon={faFolderOpen} />
                    {isOpeningWorkspace ? 'Opening...' : 'Open Folder'}
                </Button>
                <IconButton
                    className="icon-action"
                    label="Refresh workspace"
                    onClick={onRefreshWorkspace}
                    disabled={isRefreshingWorkspace}
                >
                    <FontAwesomeIcon icon={faRotateRight} />
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
                            <small title={workspaceStatus}>{workspace.root}</small>
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
                                    <FontAwesomeIcon icon={faMagnifyingGlass} />
                                    {isSearchingWorkspace ? 'Searching...' : 'Search'}
                                </Button>
                            </div>
                            <div className="tree-tool-row">
                                <Button onClick={onExpandAllDirectories} variant="subtle">
                                    <FontAwesomeIcon icon={faExpand} />
                                    Expand all
                                </Button>
                                <Button onClick={onCollapseAllDirectories} variant="subtle">
                                    <FontAwesomeIcon icon={faCompress} />
                                    Collapse all
                                </Button>
                                {workspace.scan.ignoredSamples.length > 0 && (
                                    <Button onClick={() => setShowIgnoredSamples((current) => !current)} variant="subtle">
                                        <FontAwesomeIcon icon={faBan} />
                                        {showIgnoredSamples ? 'Hide ignored' : `Ignored ${workspace.scan.ignored}`}
                                    </Button>
                                )}
                                <Button onClick={onCreateScanReport} disabled={isCreatingScanReport} variant="subtle">
                                    <FontAwesomeIcon icon={faFloppyDisk} />
                                    {isCreatingScanReport ? 'Saving scan...' : 'Save scan'}
                                </Button>
                                {workspaceSearchResults.length > 0 && (
                                    <Button onClick={onClearWorkspaceSearch} variant="subtle">
                                        <FontAwesomeIcon icon={faBroom} />
                                        Clear results
                                    </Button>
                                )}
                            </div>
                        </div>
                        {showIgnoredSamples && workspace.scan.ignoredSamples.length > 0 && (
                            <div className="ignored-path-panel">
                                <strong>Ignored paths</strong>
                                {workspace.scan.ignoredSamples.map((sample) => <small key={sample}>{sample}</small>)}
                            </div>
                        )}
                        {workspaceSearchResults.length > 0 && (
                            <div className="search-results">
                                <div className="section-label">{workspaceSearchResults.length} matches</div>
                                {groupSearchResults(workspaceSearchResults).map((group) => (
                                    <div className="search-result-group" key={group.key}>
                                        <small>{group.label}</small>
                                        {group.results.map((result, index) => (
                                            <button className="search-result" key={`${group.key}-${result.relPath}-${result.matchType}-${index}`} onClick={() => onSelectSearchResult(result)}>
                                                <strong>{result.relPath}</strong>
                                                <small>{result.matchType}{result.line > 0 ? `, line ${result.line}` : ''}</small>
                                                <span>{result.snippet}</span>
                                            </button>
                                        ))}
                                    </div>
                                ))}
                            </div>
                        )}
                        <div className="project-tree" role="tree" aria-label="Project tree">
                            {workspaceNodes.map((node) => (
                                <TreeNodeButton
                                    activeFile={activeFile}
                                    expandedDirectories={expandedDirectories}
                                    gitBadge={gitBadgeForNode(node, gitStatusByPath)}
                                    icon={fileIconByType[node.fileType] ?? brandAssets.icons.documents}
                                    key={node.relPath}
                                    node={node}
                                    onContextMenu={openContextMenu}
                                    onSelect={onSelectWorkspaceNode}
                                />
                            ))}
                        </div>
                        {contextMenu && (
                            <div
                                className="tree-context-menu"
                                onClick={(event) => event.stopPropagation()}
                                role="menu"
                                style={{left: contextMenu.x, top: contextMenu.y}}
                            >
                                <button onClick={() => runContextAction('new-file')} role="menuitem"><FontAwesomeIcon icon={faFileCirclePlus} /> New file</button>
                                <button onClick={() => runContextAction('new-folder')} role="menuitem"><FontAwesomeIcon icon={faFolderPlus} /> New folder</button>
                                <button onClick={() => runContextAction('rename')} role="menuitem"><FontAwesomeIcon icon={faPen} /> Rename</button>
                                <button onClick={() => runContextAction('move')} role="menuitem"><FontAwesomeIcon icon={faUpRightFromSquare} /> Move</button>
                                <button disabled={contextMenu.node.kind !== 'file'} onClick={() => runContextAction('delete')} role="menuitem"><FontAwesomeIcon icon={faTrash} /> Delete</button>
                                <button disabled={contextMenu.node.kind !== 'file'} onClick={() => runContextAction('cut')} role="menuitem"><FontAwesomeIcon icon={faScissors} /> Cut</button>
                                <button disabled={contextMenu.node.kind !== 'file'} onClick={() => runContextAction('copy')} role="menuitem"><FontAwesomeIcon icon={faCopy} /> Copy</button>
                                <button disabled={!treeClipboard} onClick={() => runContextAction('paste')} role="menuitem">
                                    <FontAwesomeIcon icon={faPaste} />
                                    Paste{treeClipboard ? ` ${treeClipboard.mode === 'cut' ? 'move' : 'copy'}` : ''}
                                </button>
                                <button onClick={() => runContextAction('copy-path')} role="menuitem"><FontAwesomeIcon icon={faCopy} /> Copy path</button>
                                <button onClick={() => runContextAction('reveal')} role="menuitem"><FontAwesomeIcon icon={faUpRightFromSquare} /> Reveal</button>
                            </div>
                        )}
                    </>
                ) : (
                    <>
                        <div className="workspace-summary">
                            <strong>Scaffold preview</strong>
                            <small>{workspaceStatus}</small>
                        </div>
                        <div className="project-tree scaffold-tree" role="tree" aria-label="Scaffold project tree">
                            {workspaceItems.map((item) => (
                                <button
                                    aria-selected={activeFile.startsWith(item.name)}
                                    className={activeFile.startsWith(item.name) ? 'tree-item depth-zero selected' : 'tree-item depth-zero'}
                                    key={item.name}
                                    onClick={() => onSelectFallbackItem(item.name)}
                                    role="treeitem"
                                    style={{'--tree-depth': 0} as CSSProperties}
                                >
                                    <span className="tree-indent-guide" />
                                    <span className="tree-disclosure" />
                                    <span className={`file-glyph ${item.kind}`}>
                                        <FontAwesomeIcon icon={workspaceIconByName[item.name] ?? brandAssets.icons.documents} />
                                    </span>
                                    <span className="tree-node-main">
                                        <strong>{item.name}</strong>
                                        <small>{item.meta}</small>
                                    </span>
                                    <span className="tree-node-badge">{item.kind}</span>
                                </button>
                            ))}
                        </div>
                    </>
                )}
            </div>
        </section>
    );
}

function TreeNodeButton({
    activeFile,
    expandedDirectories,
    gitBadge,
    icon,
    node,
    onContextMenu,
    onSelect,
}: {
    activeFile: string;
    expandedDirectories: Set<string>;
    gitBadge: string;
    icon: IconDefinition;
    node: FileNode;
    onContextMenu: (event: ReactMouseEvent, node: FileNode) => void;
    onSelect: (node: FileNode) => void;
}) {
    const isDirectory = node.kind === 'directory';
    const isExpanded = isDirectory && expandedDirectories.has(node.relPath);
    const depth = Math.min(Math.max(node.depth, 0), 16);
    const changed = gitBadge !== '';
    return (
        <button
            aria-expanded={isDirectory ? isExpanded : undefined}
            aria-level={depth + 1}
            aria-selected={activeFile === node.relPath}
            className={[
                'tree-item',
                depth === 0 ? 'depth-zero' : '',
                isDirectory ? 'directory-node' : 'file-node',
                activeFile === node.relPath ? 'selected' : '',
                changed ? 'changed' : '',
            ].filter(Boolean).join(' ')}
            data-file-type={node.fileType}
            data-kind={node.kind}
            onContextMenu={(event) => onContextMenu(event, node)}
            onClick={() => onSelect(node)}
            role="treeitem"
            style={{'--tree-depth': depth} as CSSProperties}
            title={node.relPath}
        >
            <span className="tree-indent-guide" />
            <span className={isExpanded ? 'tree-disclosure expanded' : 'tree-disclosure'}>
                {isDirectory ? <FontAwesomeIcon icon={faChevronRight} /> : null}
            </span>
            <span className={`file-glyph ${node.kind}`}>
                <FontAwesomeIcon icon={icon} />
            </span>
            <span className="tree-node-main">
                <strong>{node.name}</strong>
                <small>{changed ? `${node.meta} / git ${gitBadge}` : node.meta}</small>
            </span>
            <span className={changed ? 'tree-node-badge changed' : 'tree-node-badge'}>
                {changed ? gitBadge : treeNodeBadge(node)}
            </span>
        </button>
    );
}

function buildGitStatusMap(changes: GitFileChange[]) {
    const byPath = new Map<string, string>();
    for (const change of changes) {
        byPath.set(change.path, gitBadgeForChange(change));
        if (change.oldPath) {
            byPath.set(change.oldPath, 'R');
        }
    }
    return byPath;
}

function gitBadgeForNode(node: FileNode, byPath: Map<string, string>) {
    const direct = byPath.get(node.relPath);
    if (direct) {
        return direct;
    }
    if (node.kind !== 'directory') {
        return '';
    }
    const prefix = node.relPath ? `${node.relPath}/` : '';
    for (const [path, badge] of byPath) {
        if (path.startsWith(prefix)) {
            return badge === '?' ? '?' : '*';
        }
    }
    return '';
}

function gitBadgeForChange(change: GitFileChange) {
    const combined = `${change.index}${change.worktree}`.trim();
    if (combined.includes('?')) {
        return '?';
    }
    if (combined.includes('R')) {
        return 'R';
    }
    if (combined.includes('A')) {
        return 'A';
    }
    if (combined.includes('D')) {
        return 'D';
    }
    if (combined.includes('M')) {
        return 'M';
    }
    return '*';
}

function treeNodeBadge(node: FileNode) {
    if (node.kind === 'directory') {
        return 'dir';
    }
    if (node.fileType) {
        return node.fileType;
    }
    return 'file';
}

function groupSearchResults(results: WorkspaceSearchResult[]) {
    const groups = [
        {key: 'files', label: 'Files', results: [] as WorkspaceSearchResult[]},
        {key: 'artifacts', label: 'Artifacts', results: [] as WorkspaceSearchResult[]},
        {key: 'chat', label: 'Chat History', results: [] as WorkspaceSearchResult[]},
    ];
    for (const result of results) {
        if (result.kind === 'chat') {
            groups[2].results.push(result);
        } else if (result.matchType === 'artifact' || result.relPath.toLowerCase().startsWith('.nexusdesk/artifacts/')) {
            groups[1].results.push(result);
        } else {
            groups[0].results.push(result);
        }
    }
    return groups.filter((group) => group.results.length > 0);
}
