import type {ChangeEvent} from 'react';
import {Button, Card} from '../../components/ui';
import type {LLMProbeResult, LLMSettings} from '../../types';

type LLMSettingsCardProps = {
    isSavingSettings: boolean;
    isTestingConnection: boolean;
    onSaveSettings: () => void;
    onSettingsDraftChange: (field: keyof LLMSettings, value: string) => void;
    onTestConnection: () => void;
    probeResult: LLMProbeResult | null;
    settingsDraft: LLMSettings;
    settingsStatus: string;
};

export function LLMSettingsCard({
    isSavingSettings,
    isTestingConnection,
    onSaveSettings,
    onSettingsDraftChange,
    onTestConnection,
    probeResult,
    settingsDraft,
    settingsStatus,
}: LLMSettingsCardProps) {
    function updateField(field: keyof LLMSettings) {
        return (event: ChangeEvent<HTMLInputElement>) => onSettingsDraftChange(field, event.target.value);
    }

    return (
        <Card className="settings-card">
            <div className="pane-title">
                <span>LLM Provider</span>
                <small>Local config</small>
            </div>
            <div className="settings-form">
                <label>
                    <span>Provider</span>
                    <input value={settingsDraft.providerName} onChange={updateField('providerName')} />
                </label>
                <label>
                    <span>Base URL</span>
                    <input value={settingsDraft.baseUrl} onChange={updateField('baseUrl')} />
                </label>
                <label>
                    <span>Model</span>
                    <input value={settingsDraft.model} onChange={updateField('model')} placeholder="Optional" />
                </label>
                <label>
                    <span>API Key</span>
                    <input
                        value={settingsDraft.apiKey}
                        onChange={updateField('apiKey')}
                        placeholder="Optional"
                        type="password"
                    />
                </label>
                <div className="settings-actions">
                    <small>{settingsStatus}</small>
                    <div className="settings-button-row">
                        <Button onClick={onTestConnection} disabled={isTestingConnection}>
                            {isTestingConnection ? 'Testing...' : 'Test'}
                        </Button>
                        <Button onClick={onSaveSettings} disabled={isSavingSettings} variant="primary">
                            {isSavingSettings ? 'Saving...' : 'Save'}
                        </Button>
                    </div>
                </div>
                {probeResult && (
                    <div className={probeResult.ok ? 'probe-result ok' : 'probe-result failed'}>
                        <strong>{probeResult.ok ? 'Connection ready' : 'Connection issue'}</strong>
                        <span>{probeResult.endpoint}</span>
                        {probeResult.capabilities.length > 0 && (
                            <div className="probe-capabilities">
                                {probeResult.capabilities.map((capability) => (
                                    <small key={capability}>{capability}</small>
                                ))}
                            </div>
                        )}
                        {probeResult.modelSample.length > 0 && <small>{probeResult.modelSample.join(', ')}</small>}
                        {probeResult.warnings.map((warning) => (
                            <small className="probe-warning" key={warning}>{warning}</small>
                        ))}
                    </div>
                )}
            </div>
        </Card>
    );
}
