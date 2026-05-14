import {StatusBadge} from '../../components/ui';
import type {ArtifactMetadata, FilePreview} from '../../types';

export function ArtifactMetadataPanel({metadata, preview}: {metadata: ArtifactMetadata; preview: FilePreview | null}) {
    const isChart = metadata.kind === 'chart-svg' || /\.svg$/i.test(preview?.name ?? '');

    return (
        <div className="artifact-metadata-panel">
            <strong>{metadata.title || 'Artifact metadata'}</strong>
            <p>{metadata.source || metadata.kind}</p>
            {isChart && preview?.content && (
                <div className="artifact-chart-preview">
                    <img src={preview.content} alt={metadata.title || preview.name} />
                </div>
            )}
            <dl>
                <div>
                    <dt>Kind</dt>
                    <dd><StatusBadge tone="warning">{metadata.kind || 'artifact'}</StatusBadge></dd>
                </div>
                {metadata.contextRelPath && (
                    <div>
                        <dt>Context</dt>
                        <dd>{metadata.contextRelPath}</dd>
                    </div>
                )}
                {metadata.model && (
                    <div>
                        <dt>Model</dt>
                        <dd>{metadata.model}</dd>
                    </div>
                )}
                {metadata.createdAt && (
                    <div>
                        <dt>Created</dt>
                        <dd>{metadata.createdAt}</dd>
                    </div>
                )}
            </dl>
            {metadata.prompt && (
                <div className="artifact-config-block">
                    <small>Configuration</small>
                    <p>{metadata.prompt}</p>
                </div>
            )}
        </div>
    );
}
