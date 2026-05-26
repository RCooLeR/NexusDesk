import type {ChangeEvent} from 'react';
import {Button, Card, StatusBadge} from '../../components/ui';
import type {ConnectorProfile} from '../../types';

type ConnectorProfilesCardProps = {
    draft: ConnectorProfile;
    isSaving: boolean;
    onDelete: (id: string) => void;
    onDraftChange: (field: keyof ConnectorProfile, value: string | number | boolean) => void;
    onSave: () => void;
    profiles: ConnectorProfile[];
    status: string;
};

const connectorKinds = [
    {id: 'postgres', label: 'PostgreSQL'},
    {id: 'mysql', label: 'MySQL'},
    {id: 'mariadb', label: 'MariaDB'},
    {id: 'sqlserver', label: 'SQL Server'},
    {id: 'duckdb', label: 'DuckDB'},
    {id: 'sqlite', label: 'SQLite'},
];

export function ConnectorProfilesCard({
    draft,
    isSaving,
    onDelete,
    onDraftChange,
    onSave,
    profiles,
    status,
}: ConnectorProfilesCardProps) {
    function updateText(field: keyof ConnectorProfile) {
        return (event: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => onDraftChange(field, event.target.value);
    }

    function updateNumber(field: keyof ConnectorProfile) {
        return (event: ChangeEvent<HTMLInputElement>) => onDraftChange(field, Number(event.target.value));
    }

    return (
        <Card className="settings-card connector-profiles-card">
            <div className="pane-title">
                <span>Connector Profiles</span>
                <StatusBadge tone="neutral">{profiles.length} saved</StatusBadge>
            </div>
            <div className="settings-form connector-profile-form">
                <label>
                    <span>Name</span>
                    <input value={draft.name} onChange={updateText('name')} placeholder="Marketing warehouse" />
                </label>
                <label>
                    <span>Engine</span>
                    <select value={draft.kind} onChange={updateText('kind')}>
                        {connectorKinds.map((kind) => <option key={kind.id} value={kind.id}>{kind.label}</option>)}
                    </select>
                </label>
                <label>
                    <span>Host or file</span>
                    <input value={draft.host} onChange={updateText('host')} placeholder="db.example.local or data/main.db" />
                </label>
                <label>
                    <span>Port</span>
                    <input min="0" max="65535" type="number" value={draft.port} onChange={updateNumber('port')} />
                </label>
                <label>
                    <span>Database</span>
                    <input value={draft.database} onChange={updateText('database')} placeholder="analytics" />
                </label>
                <label>
                    <span>User</span>
                    <input value={draft.username} onChange={updateText('username')} />
                </label>
                <label>
                    <span>Password/token</span>
                    <input type="password" value={draft.password} onChange={updateText('password')} placeholder={draft.credentialRef ? 'Stored credential' : ''} />
                </label>
                <label>
                    <span>SSL mode</span>
                    <input value={draft.sslMode} onChange={updateText('sslMode')} placeholder="prefer" />
                </label>
                <label>
                    <span>Result cap</span>
                    <input min="1" max="10000" type="number" value={draft.resultLimit} onChange={updateNumber('resultLimit')} />
                </label>
                <label>
                    <span>Timeout</span>
                    <input min="1" max="300" type="number" value={draft.timeoutSeconds} onChange={updateNumber('timeoutSeconds')} />
                </label>
                <small className="settings-help-text">
                    protected credential references are stored outside the public profile list and returned redacted.
                </small>
            </div>
            <div className="settings-actions">
                <small>{status}</small>
                <div className="settings-button-row">
                    <Button onClick={onSave} disabled={isSaving} variant="primary">
                        {isSaving ? 'Saving...' : draft.id ? 'Update profile' : 'Save profile'}
                    </Button>
                </div>
            </div>
            {profiles.length > 0 && (
                <div className="connector-profile-list">
                    {profiles.map((profile) => (
                        <div className="connector-profile-row" key={profile.id}>
                            <div>
                                <strong>{profile.name}</strong>
                                <small>{profile.kind} / {profile.readOnly ? 'read-only' : 'write-capable blocked'} / cap {profile.resultLimit}</small>
                                <small>{profile.host || profile.database || 'local profile'}{profile.credentialRef ? ' / credential stored' : ''}</small>
                            </div>
                            <Button onClick={() => onDelete(profile.id)} variant="subtle">Delete</Button>
                        </div>
                    ))}
                </div>
            )}
        </Card>
    );
}
