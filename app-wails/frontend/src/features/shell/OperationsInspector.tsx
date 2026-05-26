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
    const composeServices = parseComposeServices(preview?.content ?? '');

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
            {composeServices.length > 0 && (
                <div className="compose-service-list">
                    <small>Services in selected compose file</small>
                    {composeServices.map((service) => (
                        <div className="compose-service-row" key={service.name}>
                            <strong>{service.name}</strong>
                            <span>{service.image || 'build context'}</span>
                            <small>{service.ports.length > 0 ? `ports ${service.ports.join(', ')}` : 'no ports'}</small>
                            {service.volumes.length > 0 && <small>volumes {service.volumes.join(', ')}</small>}
                            {service.dependsOn.length > 0 && <small>depends on {service.dependsOn.join(', ')}</small>}
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}

type ComposeService = {
    name: string;
    image: string;
    ports: string[];
    volumes: string[];
    dependsOn: string[];
};

export function parseComposeServices(content: string): ComposeService[] {
    const lines = content.replace(/\r\n/g, '\n').split('\n');
    const services: ComposeService[] = [];
    let inServices = false;
    let current: ComposeService | null = null;
    let currentList: 'ports' | 'volumes' | 'dependsOn' | null = null;

    for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed || trimmed.startsWith('#')) {
            continue;
        }
        const indent = line.search(/\S/);
        if (indent === 0 && trimmed === 'services:') {
            inServices = true;
            current = null;
            currentList = null;
            continue;
        }
        if (!inServices) {
            continue;
        }
        if (indent === 0 && !trimmed.startsWith('services:')) {
            break;
        }
        const serviceMatch = line.match(/^ {2}([A-Za-z0-9._-]+):\s*$/);
        if (serviceMatch) {
            current = {name: serviceMatch[1], image: '', ports: [], volumes: [], dependsOn: []};
            services.push(current);
            currentList = null;
            continue;
        }
        if (!current || indent < 4) {
            continue;
        }

        const keyMatch = trimmed.match(/^([A-Za-z_]+):\s*(.*)$/);
        if (keyMatch) {
            const key = keyMatch[1];
            const value = stripComposeValue(keyMatch[2]);
            currentList = null;
            if (key === 'image') {
                current.image = value;
            }
            if (key === 'ports') {
                currentList = 'ports';
                appendInlineComposeList(current.ports, value);
            }
            if (key === 'volumes') {
                currentList = 'volumes';
                appendInlineComposeList(current.volumes, value);
            }
            if (key === 'depends_on') {
                currentList = 'dependsOn';
                appendInlineComposeList(current.dependsOn, value);
            }
            continue;
        }

        if (currentList && trimmed.startsWith('- ')) {
            current[currentList].push(stripComposeValue(trimmed.slice(2)));
        }
    }

    return services;
}

function appendInlineComposeList(target: string[], value: string) {
    if (!value || !value.startsWith('[') || !value.endsWith(']')) {
        return;
    }
    for (const item of value.slice(1, -1).split(',')) {
        const cleaned = stripComposeValue(item);
        if (cleaned) {
            target.push(cleaned);
        }
    }
}

function stripComposeValue(value: string) {
    return value.trim().replace(/^['"]|['"]$/g, '');
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
