import type {LLMRuntimeStatus, LLMSettings} from '../../types';

export type LLMModelOption = {
    id: string;
    label: string;
    chatLabel: string;
    maxContextTokens: number;
};

export const recommendedModelOptions: LLMModelOption[] = [
    {id: 'qwen3:4b-instruct', label: 'Qwen3 4B Instruct - fast local', chatLabel: 'Qwen3 4B', maxContextTokens: 32768},
    {id: 'qwen3:8b', label: 'Qwen3 8B - balanced', chatLabel: 'Qwen3 8B', maxContextTokens: 40960},
    {id: 'qwen3.5:9b', label: 'Qwen3.5 9B - workspace chat', chatLabel: 'Qwen3.5 9B', maxContextTokens: 131072},
    {id: 'phi4:14b', label: 'Phi-4 14B - reasoning', chatLabel: 'Phi-4 14B', maxContextTokens: 16384},
    {id: 'phi4-reasoning:14b', label: 'Phi-4 Reasoning 14B - deep reasoning', chatLabel: 'Phi-4 Reasoning', maxContextTokens: 32768},
    {id: 'gpt-oss:20b', label: 'GPT-OSS 20B - strong general', chatLabel: 'GPT-OSS 20B', maxContextTokens: 131072},
    {id: 'mistral-small3.2:latest', label: 'Mistral Small 3.2 - long context', chatLabel: 'Mistral Small', maxContextTokens: 131072},
    {id: 'gemma4:26b', label: 'Gemma 4 26B - max local', chatLabel: 'Gemma 4 26B', maxContextTokens: 131072},
];

export const fallbackModelContextTokens = 32768;

export function modelContextWindow(model: string, runtime?: LLMRuntimeStatus | null) {
    const runtimeContext = runtimeContextWindow(model, runtime);
    if (runtimeContext > 0) {
        return runtimeContext;
    }
    return recommendedModelOptions.find((option) => modelMatches(option.id, model))?.maxContextTokens ?? fallbackModelContextTokens;
}

export function responseReserveForContext(maxContextTokens: number) {
    if (!Number.isFinite(maxContextTokens) || maxContextTokens <= 0) {
        return 4096;
    }
    return Math.min(32768, Math.max(2048, Math.floor(maxContextTokens / 8)));
}

export function settingsForSelectedModel(settings: LLMSettings, model: string, runtime?: LLMRuntimeStatus | null): LLMSettings {
    const maxContextTokens = modelContextWindow(model, runtime);
    return {
        ...settings,
        model,
        maxContextTokens,
        responseReserveTokens: responseReserveForContext(maxContextTokens),
    };
}

export function settingsWithRuntimeContext(settings: LLMSettings, runtime?: LLMRuntimeStatus | null): LLMSettings {
    const runtimeContext = runtimeContextWindow(settings.model, runtime);
    if (runtimeContext <= 0 || runtimeContext === settings.maxContextTokens) {
        return settings;
    }
    return {
        ...settings,
        maxContextTokens: runtimeContext,
        responseReserveTokens: responseReserveForContext(runtimeContext),
    };
}

function runtimeContextWindow(model: string, runtime?: LLMRuntimeStatus | null) {
    const loaded = runtime?.loadedModels.find((candidate) => modelMatches(candidate.name, model) || modelMatches(candidate.model, model));
    return loaded && loaded.contextLength > 0 ? loaded.contextLength : 0;
}

function modelMatches(left: string, right: string) {
    return normalizeModelName(left) === normalizeModelName(right);
}

function normalizeModelName(value: string) {
    return value.trim().toLowerCase().replace(':latest', '');
}
