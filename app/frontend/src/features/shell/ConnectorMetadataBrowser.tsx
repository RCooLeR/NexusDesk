import {useEffect, useMemo, useState} from 'react';
import {Button} from '../../components/ui';
import type {ConnectorMetadata, ConnectorRelationship, ConnectorTable} from '../../types';

type ConnectorMetadataBrowserProps = {
    metadata: ConnectorMetadata;
    onExplainObject?: (objectName: string) => void;
    onPreviewObject?: (objectName: string) => void;
    onUseQuery?: (query: string) => void;
};

export function ConnectorMetadataBrowser({
    metadata,
    onExplainObject,
    onPreviewObject,
    onUseQuery,
}: ConnectorMetadataBrowserProps) {
    const objects = useMemo(() => [...metadata.tables, ...metadata.views], [metadata.tables, metadata.views]);
    const [selectedObjectName, setSelectedObjectName] = useState(objects[0]?.name ?? '');
    const selectedObject = objects.find((object) => object.name === selectedObjectName) ?? objects[0] ?? null;
    const selectedRelationships = useMemo(
        () => selectedObject ? relationshipsForObject(metadata.relationships, selectedObject.name) : [],
        [metadata.relationships, selectedObject],
    );

    useEffect(() => {
        setSelectedObjectName((current) => objects.some((object) => object.name === current) ? current : objects[0]?.name ?? '');
    }, [metadata.id, objects]);

    return (
        <div className="connector-metadata-panel">
            <small>{metadata.message}</small>
            <div className="metadata-dataset-views">
                <strong>{metadata.engine}{metadata.readOnly ? ' / read-only' : ''}</strong>
                {objects.length > 0 ? (
                    <div className="connector-schema-browser">
                        <select aria-label={`${metadata.kind} schema object`} onChange={(event) => setSelectedObjectName(event.target.value)} value={selectedObject?.name ?? ''}>
                            {objects.map((object) => (
                                <option key={`${object.type}-${object.name}`} value={object.name}>
                                    {object.type}: {object.name} / {object.rowCount} rows
                                </option>
                            ))}
                        </select>
                        {selectedObject && (
                            <>
                                <div className="metadata-action-row">
                                    {onPreviewObject && <Button onClick={() => onPreviewObject(selectedObject.name)} variant="subtle">Preview rows</Button>}
                                    {onUseQuery && <Button onClick={() => onUseQuery(`select * from ${quoteConnectorIdentifierForUI(selectedObject.name)}`)} variant="subtle">Use query</Button>}
                                    {onExplainObject && <Button onClick={() => onExplainObject(selectedObject.name)} variant="subtle">Explain schema</Button>}
                                </div>
                                <p>
                                    {selectedObject.type}: {selectedObject.name} / {selectedObject.rowCount} rows
                                    <small>{selectedObject.columns.map((column) => `${column.name}:${column.type || 'ANY'}${column.primaryKey ? ' pk' : ''}`).slice(0, 8).join(', ')}</small>
                                </p>
                                {selectedObject.indexes.length > 0 && (
                                    <div className="connector-index-list">
                                        {selectedObject.indexes.slice(0, 8).map((index) => (
                                            <small key={index.name}>{index.unique ? 'unique ' : ''}{index.name}: {index.columns.join(', ')}</small>
                                        ))}
                                    </div>
                                )}
                                {selectedRelationships.length > 0 && (
                                    <ConnectorRelationshipList relationships={selectedRelationships} selectedObject={selectedObject} />
                                )}
                                {selectedObject.sampleRows.length > 0 && (
                                    <div className="metadata-sample">
                                        {selectedObject.sampleRows.slice(0, 3).map((row, index) => (
                                            <p key={`${selectedObject.name}-${index}`}>{row.slice(0, 8).join(' | ')}</p>
                                        ))}
                                    </div>
                                )}
                            </>
                        )}
                    </div>
                ) : (
                    <small>No tables or views found.</small>
                )}
            </div>
        </div>
    );
}

function ConnectorRelationshipList({
    relationships,
    selectedObject,
}: {
    relationships: ConnectorRelationship[];
    selectedObject: ConnectorTable;
}) {
    return (
        <div className="connector-relationship-list">
            <strong>Relationships</strong>
            {relationships.slice(0, 8).map((relationship) => {
                const outbound = relationship.fromTable === selectedObject.name;
                const label = outbound
                    ? `${relationship.fromColumn} -> ${relationship.toTable}.${relationship.toColumn || 'id'}`
                    : `${relationship.fromTable}.${relationship.fromColumn} -> ${selectedObject.name}.${relationship.toColumn || 'id'}`;
                return (
                    <small key={`${relationship.kind}-${relationship.fromTable}-${relationship.fromColumn}-${relationship.toTable}-${relationship.toColumn}`}>
                        {relationship.kind === 'foreign-key' ? 'FK' : 'hint'} / {relationship.confidence}: {label}
                    </small>
                );
            })}
        </div>
    );
}

function relationshipsForObject(relationships: ConnectorRelationship[] | undefined, objectName: string) {
    return (relationships ?? []).filter((relationship) => relationship.fromTable === objectName || relationship.toTable === objectName);
}

function quoteConnectorIdentifierForUI(value: string) {
    return value
        .split('.')
        .map((part) => `"${part.replaceAll('"', '""')}"`)
        .join('.');
}
