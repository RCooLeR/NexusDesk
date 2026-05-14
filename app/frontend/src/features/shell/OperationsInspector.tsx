import {StatusBadge} from '../../components/ui';
import type {FileNode, FilePreview, WorkspaceSnapshot} from '../../types';

export function OperationsInspector({preview, workspace}: {preview: FilePreview | null; workspace: WorkspaceSnapshot | null}) {
    if (!workspace || !isOperationsContext(preview)) {
        return null;
    }

    const composeFiles = workspace.nodes.filter(isComposeNode);
    const dockerfiles = workspace.nodes.filter((node) => node.name.toLowerCase() === 'dockerfile');
    const serviceNodes = workspace.nodes.filter((node) => node.relPath.startsWith('services/'));
    const selectedStatus = selectedOperationsStatus(preview);

    return (
        <div className="operations-inspector-panel">
            <strong>Operations Inspector</strong>
            <div className="operations-status-grid">
                <div>
                    <small>Selected</small>
                    <StatusBadge tone="neutral">{selectedStatus}</StatusBadge>
                </div>
                <div>
                    <small>Compose</small>
                    <strong>{composeFiles.length}</strong>
                </div>
                <div>
                    <small>Dockerfiles</small>
                    <strong>{dockerfiles.length}</strong>
                </div>
                <div>
                    <small>Services</small>
                    <strong>{serviceNodes.length}</strong>
                </div>
            </div>
            <div className="operations-readonly-note">
                <small>Read-only inspector</small>
                <p>Service and Docker actions stay disabled until modal approvals cover external operations.</p>
            </div>
            {composeFiles.length > 0 && (
                <div className="operations-file-list">
                    <small>Compose files</small>
                    {composeFiles.slice(0, 5).map((node) => (
                        <span key={node.relPath}>{node.relPath}</span>
                    ))}
                </div>
            )}
        </div>
    );
}

function isOperationsContext(preview: FilePreview | null) {
    if (!preview) {
        return false;
    }
    const relPath = preview.relPath.toLowerCase();
    const name = preview.name.toLowerCase();
    return name === 'dockerfile' ||
        name.includes('docker-compose') ||
        relPath.startsWith('services/') ||
        /\.(env|ps1|sh|bat|cmd|toml|ya?ml)$/i.test(name);
}

function isComposeNode(node: FileNode) {
    const name = node.name.toLowerCase();
    return node.kind === 'file' && (name === 'compose.yml' || name === 'compose.yaml' || name.includes('docker-compose'));
}

function selectedOperationsStatus(preview: FilePreview | null) {
    if (!preview) {
        return 'none';
    }
    if (preview.name.toLowerCase().includes('docker-compose') || /^compose\.ya?ml$/i.test(preview.name)) {
        return 'compose';
    }
    if (preview.name.toLowerCase() === 'dockerfile') {
        return 'image';
    }
    if (preview.relPath.toLowerCase().startsWith('services/')) {
        return 'service';
    }
    return 'config';
}
