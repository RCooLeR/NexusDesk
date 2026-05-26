import {useEffect, useMemo, useState} from 'react';
import {FontAwesomeIcon} from '@fortawesome/react-fontawesome';
import {faArrowRight, faDiagramProject, faKey, faTable} from '@fortawesome/free-solid-svg-icons';
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
    const relationshipMapObjects = useMemo(
        () => objects.filter((object) => object.columns.length > 0 || relationshipsForObject(metadata.relationships, object.name).length > 0),
        [metadata.relationships, objects],
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
                                {metadata.relationships.length > 0 && (
                                    <ConnectorRelationshipMap
                                        objects={relationshipMapObjects}
                                        onSelectObject={setSelectedObjectName}
                                        relationships={metadata.relationships}
                                        selectedObjectName={selectedObject.name}
                                    />
                                )}
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

function ConnectorRelationshipMap({
    objects,
    onSelectObject,
    relationships,
    selectedObjectName,
}: {
    objects: ConnectorTable[];
    onSelectObject: (objectName: string) => void;
    relationships: ConnectorRelationship[];
    selectedObjectName: string;
}) {
    const activeObjectNames = relatedObjectNames(relationships, selectedObjectName);
    const visibleRelationships = relationshipPriority(relationships, selectedObjectName).slice(0, 12);
    return (
        <div className="connector-erd-map" aria-label="Connector relationship map">
            <div className="connector-erd-heading">
                <span><FontAwesomeIcon icon={faDiagramProject} /> Relationship map</span>
                <small>{relationships.length} links / {objects.length} objects</small>
            </div>
            <div className="connector-erd-nodes">
                {objects.slice(0, 18).map((object) => {
                    const active = object.name === selectedObjectName;
                    const related = activeObjectNames.has(object.name);
                    const keyColumns = object.columns.filter((column) => column.primaryKey).slice(0, 3);
                    return (
                        <button
                            className={['connector-erd-node', active ? 'active' : '', related ? 'related' : ''].filter(Boolean).join(' ')}
                            key={`${object.type}-${object.name}`}
                            onClick={() => onSelectObject(object.name)}
                            type="button"
                        >
                            <span><FontAwesomeIcon icon={faTable} /> {object.name}</span>
                            <small>{object.type} / {object.rowCount} rows</small>
                            {keyColumns.length > 0 && (
                                <small className="connector-erd-keys">
                                    <FontAwesomeIcon icon={faKey} /> {keyColumns.map((column) => column.name).join(', ')}
                                </small>
                            )}
                        </button>
                    );
                })}
            </div>
            <div className="connector-erd-links">
                {visibleRelationships.map((relationship) => {
                    const active = relationship.fromTable === selectedObjectName || relationship.toTable === selectedObjectName;
                    return (
                        <button
                            className={active ? 'connector-erd-link active' : 'connector-erd-link'}
                            key={relationshipKey(relationship)}
                            onClick={() => onSelectObject(relationship.fromTable === selectedObjectName ? relationship.toTable : relationship.fromTable)}
                            type="button"
                        >
                            <span>{relationship.fromTable}<small>{relationship.fromColumn}</small></span>
                            <strong><FontAwesomeIcon icon={faArrowRight} /> {relationship.kind === 'foreign-key' ? 'FK' : 'hint'}</strong>
                            <span>{relationship.toTable}<small>{relationship.toColumn || 'id'}</small></span>
                        </button>
                    );
                })}
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

function relatedObjectNames(relationships: ConnectorRelationship[], objectName: string) {
    const names = new Set<string>([objectName]);
    relationships.forEach((relationship) => {
        if (relationship.fromTable === objectName) {
            names.add(relationship.toTable);
        }
        if (relationship.toTable === objectName) {
            names.add(relationship.fromTable);
        }
    });
    return names;
}

function relationshipPriority(relationships: ConnectorRelationship[], objectName: string) {
    return [...relationships].sort((left, right) => Number(isRelationshipActive(right, objectName)) - Number(isRelationshipActive(left, objectName)));
}

function isRelationshipActive(relationship: ConnectorRelationship, objectName: string) {
    return relationship.fromTable === objectName || relationship.toTable === objectName;
}

function relationshipKey(relationship: ConnectorRelationship) {
    return `${relationship.kind}-${relationship.fromTable}-${relationship.fromColumn}-${relationship.toTable}-${relationship.toColumn}`;
}

function quoteConnectorIdentifierForUI(value: string) {
    return value
        .split('.')
        .map((part) => `"${part.replaceAll('"', '""')}"`)
        .join('.');
}
