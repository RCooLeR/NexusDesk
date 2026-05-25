import {useState} from 'react';
import type {CSSProperties, MouseEvent as ReactMouseEvent} from 'react';
import type {IconDefinition} from '@fortawesome/fontawesome-svg-core';
import {faChevronRight, faRotateRight} from '@fortawesome/free-solid-svg-icons';
import {FontAwesomeIcon} from '@fortawesome/react-fontawesome';
import {brandAssets, productBrand, workspaceIconByName} from '../../brand/assets';
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
    changedFilePaths: string[];
};

export type TreeContextAction = 'new-file' | 'new-folder' | 'rename' | 'move' | 'delete' | 'copy-path' | 'reveal';

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
    changedFilePaths,
}: WorkspaceNavigatorProps) {
    const changedFiles = new Set(changedFilePaths);
    const [contextMenu, setContextMenu] = useState<{node: FileNode; x: number; y: number} | null>(null);
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
            <header className="navigator-header">
                <div className="product-lockup" aria-label={productBrand.name}>
                    <img src={brandAssets.logoHorizontalDark} alt="" />
                </div>
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
                            <small>{scanStatusSummary(workspace, workspaceStatus)}</small>
                            <ScanStatusDetails workspace={workspace} />
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
                                <Button onClick={onCreateScanReport} disabled={isCreatingScanReport} variant="subtle">
                                    {isCreatingScanReport ? 'Saving scan...' : 'Save scan'}
                                </Button>
                                {workspaceSearchResults.length > 0 && <Button onClick={onClearWorkspaceSearch} variant="subtle">Clear results</Button>}
                            </div>
                        </div>
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
                                    changed={changedFiles.has(node.relPath)}
                                    expandedDirectories={expandedDirectories}
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
                                <button onClick={() => runContextAction('new-file')} role="menuitem">New file</button>
                                <button onClick={() => runContextAction('new-folder')} role="menuitem">New folder</button>
                                <button onClick={() => runContextAction('rename')} role="menuitem">Rename</button>
                                <button onClick={() => runContextAction('move')} role="menuitem">Move</button>
                                <button disabled={contextMenu.node.kind !== 'file'} onClick={() => runContextAction('delete')} role="menuitem">Delete</button>
                                <button onClick={() => runContextAction('copy-path')} role="menuitem">Copy path</button>
                                <button onClick={() => runContextAction('reveal')} role="menuitem">Reveal</button>
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
    changed,
    expandedDirectories,
    icon,
    node,
    onContextMenu,
    onSelect,
}: {
    activeFile: string;
    changed: boolean;
    expandedDirectories: Set<string>;
    icon: IconDefinition;
    node: FileNode;
    onContextMenu: (event: ReactMouseEvent, node: FileNode) => void;
    onSelect: (node: FileNode) => void;
}) {
    const isDirectory = node.kind === 'directory';
    const isExpanded = isDirectory && expandedDirectories.has(node.relPath);
    const depth = Math.min(Math.max(node.depth, 0), 16);
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
                <small>{changed ? `${node.meta} / changed` : node.meta}</small>
            </span>
            <span className={changed ? 'tree-node-badge changed' : 'tree-node-badge'}>
                {changed ? 'M' : treeNodeBadge(node)}
            </span>
        </button>
    );
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

function scanStatusSummary(workspace: WorkspaceSnapshot, fallback: string) {
    if (!workspace.scan) {
        return workspace.truncated ? `${workspace.nodes.length} indexed items; scan capped for responsiveness.` : fallback;
    }
    const included = numericScanValue(workspace.scan.included);
    const skipped = numericScanValue(workspace.scan.ignored)
        + numericScanValue(workspace.scan.depthSkipped)
        + numericScanValue(workspace.scan.entrySkipped)
        + numericScanValue(workspace.scan.unreadable);
    return `${included} indexed, ${skipped} skipped. Depth ${numericScanValue(workspace.scan.maxDepth)}, cap ${numericScanValue(workspace.scan.maxEntries)}.`;
}

function ScanStatusDetails({workspace}: {workspace: WorkspaceSnapshot}) {
    const scan = workspace.scan;
    if (!scan) {
        return null;
    }
    const samples = [...safeStringArray(scan.ignoredSamples), ...safeStringArray(scan.skippedSamples)];

    return (
        <details className="scan-status-details">
            <summary>Scan status</summary>
            <dl>
                <div><dt>Included</dt><dd>{numericScanValue(scan.included)}</dd></div>
                <div><dt>Ignored</dt><dd>{numericScanValue(scan.ignored)}</dd></div>
                <div><dt>Depth skipped</dt><dd>{numericScanValue(scan.depthSkipped)}</dd></div>
                <div><dt>Entry cap</dt><dd>{numericScanValue(scan.entrySkipped)}</dd></div>
                <div><dt>Unreadable</dt><dd>{numericScanValue(scan.unreadable)}</dd></div>
            </dl>
            {samples.length > 0 && (
                <ul>
                    {samples.slice(0, 6).map((sample) => (
                        <li key={sample}>{sample}</li>
                    ))}
                </ul>
            )}
        </details>
    );
}

function safeStringArray(value: unknown) {
    return Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string') : [];
}

function numericScanValue(value: unknown) {
    return typeof value === 'number' && Number.isFinite(value) ? value : 0;
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
