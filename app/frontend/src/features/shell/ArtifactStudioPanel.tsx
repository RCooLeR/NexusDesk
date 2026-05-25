import {useEffect, useState} from 'react';
import {brandAssets} from '../../brand/assets';
import {Button, EmptyState, StatusBadge} from '../../components/ui';
import type {ArtifactComparison, ArtifactLineage, ArtifactMetadata, FilePreview, WorkspaceArtifact} from '../../types';
import {ArtifactMetadataPanel} from './ArtifactMetadataPanel';

type ArtifactStudioPanelProps = {
    artifacts: WorkspaceArtifact[];
    artifactComparison: ArtifactComparison | null;
    artifactLineage: ArtifactLineage | null;
    artifactMetadata: ArtifactMetadata | null;
    filePreview: FilePreview | null;
    isArchivingArtifact: boolean;
    isDeletingArtifact: boolean;
    onArchiveArtifact: () => void;
    onCompareArtifact: () => void;
    onDeleteArtifact: () => void;
    onExportLineage: () => void;
    onOpenArtifactSource: () => void;
    onOpenLineageSource: (relPath: string) => void;
    onRefreshLineage: () => void;
    onSelectArtifact: (artifact: WorkspaceArtifact) => void;
};

export function ArtifactStudioPanel({
    artifacts,
    artifactComparison,
    artifactLineage,
    artifactMetadata,
    filePreview,
    isArchivingArtifact,
    isDeletingArtifact,
    onArchiveArtifact,
    onCompareArtifact,
    onDeleteArtifact,
    onExportLineage,
    onOpenArtifactSource,
    onOpenLineageSource,
    onRefreshLineage,
    onSelectArtifact,
}: ArtifactStudioPanelProps) {
    return (
        <div className="artifact-studio-panel">
            <div className="artifact-studio-list">
                <div className="bottom-section-heading">
                    <strong>Artifacts</strong>
                    <small>{artifacts.length} generated reports, charts, diffs, and exports.</small>
                </div>
                {artifacts.length === 0 ? (
                    <EmptyState
                        detail="Create a report to add the first workspace artifact."
                        iconSrc={brandAssets.icons.documents}
                        title="No artifacts yet"
                    />
                ) : artifacts.map((artifact) => (
                    <button className="artifact-item" key={artifact.relPath} onClick={() => onSelectArtifact(artifact)}>
                        <img src={artifact.kind === 'chart-svg' || artifact.kind === 'dataset-query-csv' ? brandAssets.icons.data : brandAssets.icons.documents} alt="" />
                        <span>
                            <strong>{artifact.name}</strong>
                            <small>{artifact.summary || artifact.source || artifact.relPath}</small>
                            {artifact.model && <small>{artifact.model}</small>}
                        </span>
                        <StatusBadge tone="warning">{artifact.kind}</StatusBadge>
                    </button>
                ))}
            </div>

            <div className="artifact-studio-detail">
                {artifactMetadata ? (
                    <ArtifactMetadataPanel
                        isArchiving={isArchivingArtifact}
                        isDeleting={isDeletingArtifact}
                        metadata={artifactMetadata}
                        onArchive={onArchiveArtifact}
                        onCompare={onCompareArtifact}
                        onDelete={onDeleteArtifact}
                        onOpenSource={onOpenArtifactSource}
                        preview={filePreview}
                        relPath={filePreview?.relPath ?? ''}
                    />
                ) : (
                    <div className="artifact-metadata-panel">
                        <strong>Artifact Metadata</strong>
                        <small>Select an artifact to inspect source, lineage, and actions.</small>
                    </div>
                )}
                {artifactComparison && <ArtifactComparisonPanel comparison={artifactComparison} />}
            </div>

            <ArtifactLineagePanel
                lineage={artifactLineage}
                onExport={onExportLineage}
                onOpenSource={onOpenLineageSource}
                onRefresh={onRefreshLineage}
            />
        </div>
    );
}

function ArtifactComparisonPanel({comparison}: {comparison: ArtifactComparison}) {
    return (
        <div className="artifact-comparison-panel">
            <strong>Artifact Comparison</strong>
            <small>{comparison.leftTitle} {'->'} {comparison.rightTitle}</small>
            <dl>
                <div><dt>Kind</dt><dd>{comparison.sameKind ? 'same' : 'different'}</dd></div>
                <div><dt>Size delta</dt><dd>{comparison.sizeDelta} bytes</dd></div>
            </dl>
            <div className="artifact-diff-grid">
                <span>
                    <strong>Removed</strong>
                    {comparison.removedLines.length === 0 ? <small>No removed lines</small> : comparison.removedLines.map((line) => <small key={line}>- {line}</small>)}
                </span>
                <span>
                    <strong>Added</strong>
                    {comparison.addedLines.length === 0 ? <small>No added lines</small> : comparison.addedLines.map((line) => <small key={line}>+ {line}</small>)}
                </span>
            </div>
        </div>
    );
}

function ArtifactLineagePanel({
    lineage,
    onExport,
    onOpenSource,
    onRefresh,
}: {
    lineage: ArtifactLineage | null;
    onExport: () => void;
    onOpenSource: (relPath: string) => void;
    onRefresh: () => void;
}) {
    const [filter, setFilter] = useState('all');
    const [selectedNodeId, setSelectedNodeId] = useState('');
    const visibleEdges = lineage?.edges.filter((edge) => {
        if (filter === 'all') {
            return true;
        }
        const from = lineage.nodes.find((node) => node.id === edge.from);
        const to = lineage.nodes.find((node) => node.id === edge.to);
        return from?.kind === filter || to?.kind === filter;
    }) ?? [];
    const visibleNodeIds = new Set(visibleEdges.flatMap((edge) => [edge.from, edge.to]));
    const visibleNodes = lineage?.nodes.filter((node) => filter === 'all' || node.kind === filter || visibleNodeIds.has(node.id)) ?? [];
    const selectedNode = lineage?.nodes.find((node) => node.id === selectedNodeId) ?? visibleNodes[0] ?? null;
    const relationshipText = lineage?.relationshipCounts
        ? Object.entries(lineage.relationshipCounts).map(([label, count]) => `${label}: ${count}`).join(', ')
        : '';

    useEffect(() => {
        if (!selectedNodeId && visibleNodes[0]) {
            setSelectedNodeId(visibleNodes[0].id);
        }
    }, [selectedNodeId, visibleNodes]);

    return (
        <div className="metadata-store-panel artifact-lineage-panel">
            <div className="panel-toolbar">
                <strong>Artifact Lineage</strong>
                <span>
                    <Button onClick={onRefresh} variant="subtle">Refresh</Button>
                    <Button disabled={!lineage} onClick={onExport} variant="subtle">Export JSON</Button>
                </span>
            </div>
            <small>{lineage?.message ?? 'Build graph from chats, tools, source files, and artifacts.'}</small>
            {relationshipText && <small>{relationshipText}</small>}
            {lineage && (
                <div className="lineage-filter-row" aria-label="Lineage filter">
                    {['all', 'source', 'chat', 'tool', 'artifact'].map((kind) => (
                        <button className={filter === kind ? 'selected' : ''} key={kind} onClick={() => setFilter(kind)}>
                            {kind}
                        </button>
                    ))}
                </div>
            )}
            {lineage && (
                <div className="lineage-graph-layout">
                    <div className="lineage-node-cloud" aria-label="Lineage graph">
                        {visibleNodes.slice(0, 18).map((node, index) => (
                            <button
                                className={`lineage-node ${node.kind} ${selectedNode?.id === node.id ? 'selected' : ''}`}
                                key={node.id}
                                onClick={() => setSelectedNodeId(node.id)}
                                style={{
                                    gridColumn: `${(index % 3) + 1}`,
                                    gridRow: `${Math.floor(index / 3) + 1}`,
                                }}
                            >
                                <span>{node.kind}</span>
                                <strong>{node.label}</strong>
                            </button>
                        ))}
                    </div>
                    <div className="lineage-detail">
                        {selectedNode ? (
                            <>
                                <strong>{selectedNode.label}</strong>
                                <small>{selectedNode.kind}</small>
                                {selectedNode.relPath && <p>{selectedNode.relPath}</p>}
                                <small>
                                    {visibleEdges.filter((edge) => edge.from === selectedNode.id || edge.to === selectedNode.id).length} relationships
                                </small>
                                {selectedNode.relPath && (
                                    <Button onClick={() => onOpenSource(selectedNode.relPath)} variant="subtle">Open source</Button>
                                )}
                            </>
                        ) : (
                            <small>No lineage node selected.</small>
                        )}
                    </div>
                    <div className="lineage-list">
                        {visibleEdges.slice(0, 8).map((edge, index) => {
                            const from = lineage.nodes.find((node) => node.id === edge.from);
                            const to = lineage.nodes.find((node) => node.id === edge.to);
                            return (
                                <p key={`${edge.from}-${edge.to}-${index}`}>
                                    {from?.label ?? edge.from} {'->'} {to?.label ?? edge.to} <small>{edge.label}</small>
                                </p>
                            );
                        })}
                    </div>
                </div>
            )}
        </div>
    );
}
