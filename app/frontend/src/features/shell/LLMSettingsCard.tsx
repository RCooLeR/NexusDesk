import type {ChangeEvent} from 'react';
import {Button, Card} from '../../components/ui';
import type {LLMProbeResult, LLMSettings} from '../../types';

const recommendedModelOptions = [
    {id: 'qwen3:4b-instruct', label: 'Qwen3 4B Instruct - fast local'},
    {id: 'qwen3:8b', label: 'Qwen3 8B - balanced'},
    {id: 'qwen3.5:9b', label: 'Qwen3.5 9B - workspace chat'},
    {id: 'phi4:14b', label: 'Phi-4 14B - reasoning'},
    {id: 'phi4-reasoning:14b', label: 'Phi-4 Reasoning 14B - deep reasoning'},
    {id: 'gpt-oss:20b', label: 'GPT-OSS 20B - strong general'},
    {id: 'mistral-small3.2:latest', label: 'Mistral Small 3.2 - long context'},
    {id: 'gemma4:26b', label: 'Gemma 4 26B - max local'},
];

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
        return (event: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => onSettingsDraftChange(field, event.target.value);
    }

    const hasCustomModel = settingsDraft.model !== '' && !recommendedModelOptions.some((option) => option.id === settingsDraft.model);
    const runtimeModels = probeResult?.runtime?.loadedModels ?? [];

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
                    <select value={settingsDraft.model} onChange={updateField('model')}>
                        <option value="">Select model...</option>
                        {hasCustomModel && <option value={settingsDraft.model}>{settingsDraft.model} - current custom</option>}
                        {recommendedModelOptions.map((option) => (
                            <option key={option.id} value={option.id}>{option.label}</option>
                        ))}
                    </select>
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
                <div className="settings-number-grid">
                    <label>
                        <span>Context window</span>
                        <input
                            min={4096}
                            step={1024}
                            type="number"
                            value={settingsDraft.maxContextTokens}
                            onChange={updateField('maxContextTokens')}
                        />
                    </label>
                    <label>
                        <span>Response reserve</span>
                        <input
                            min={512}
                            step={512}
                            type="number"
                            value={settingsDraft.responseReserveTokens}
                            onChange={updateField('responseReserveTokens')}
                        />
                    </label>
                </div>
                <small className="settings-help-text">
                    NexusDesk uses the remaining window for selected files and context packs, and sends num_ctx to local Ollama-compatible runners.
                </small>
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
                        {probeResult.runtime && (
                            <div className="probe-runtime">
                                <strong>Ollama runtime</strong>
                                <span>{probeResult.runtime.message}</span>
                                {runtimeModels.length > 0 && (
                                    <div className="probe-runtime-models">
                                        {runtimeModels.map((model) => (
                                            <small key={model.name || model.model}>
                                                {model.name || model.model}: {formatBytes(model.sizeVram)} VRAM
                                                {model.contextLength > 0 ? `, ctx ${model.contextLength}` : ''}
                                            </small>
                                        ))}
                                    </div>
                                )}
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

function formatBytes(bytes: number) {
    if (!Number.isFinite(bytes) || bytes <= 0) {
        return '0 B';
    }

    const units = ['B', 'KiB', 'MiB', 'GiB'];
    let value = bytes;
    let unitIndex = 0;
    while (value >= 1024 && unitIndex < units.length - 1) {
        value /= 1024;
        unitIndex += 1;
    }

    return `${value.toFixed(unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
}
