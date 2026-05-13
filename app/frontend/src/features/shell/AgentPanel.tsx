import {brandAssets} from '../../brand/assets';
import type {ChatMessage, LLMProbeResult, LLMSettings, ToolEvent} from '../../types';
import {AgentChatCard} from './AgentChatCard';
import {LLMSettingsCard} from './LLMSettingsCard';
import {ToolTimeline} from './ToolTimeline';

type AgentPanelProps = {
    chatMessages: ChatMessage[];
    chatPrompt: string;
    chatStatus: string;
    contextPackPaths: string[];
    isSavingSettings: boolean;
    isSendingPrompt: boolean;
    isTestingConnection: boolean;
    onChatPromptChange: (value: string) => void;
    onClearChatHistory: () => void;
    onClearContextPack: () => void;
    onRemoveContextPath: (relPath: string) => void;
    onSaveSettings: () => void;
    onSendPrompt: () => void;
    onSettingsDraftChange: (field: keyof LLMSettings, value: string) => void;
    onTestConnection: () => void;
    probeResult: LLMProbeResult | null;
    settingsDraft: LLMSettings;
    settingsStatus: string;
    tagline: string;
    toolEvents: ToolEvent[];
};

export function AgentPanel({
    chatMessages,
    chatPrompt,
    chatStatus,
    contextPackPaths,
    isSavingSettings,
    isSendingPrompt,
    isTestingConnection,
    onChatPromptChange,
    onClearChatHistory,
    onClearContextPack,
    onRemoveContextPath,
    onSaveSettings,
    onSendPrompt,
    onSettingsDraftChange,
    onTestConnection,
    probeResult,
    settingsDraft,
    settingsStatus,
    tagline,
    toolEvents,
}: AgentPanelProps) {
    return (
        <aside className="agent-panel">
            <header>
                <img className="agent-symbol" src={brandAssets.symbolDark} alt="" />
                <p className="eyebrow">Agent</p>
                <h2>Grounded Assistant</h2>
                <span>{tagline}</span>
            </header>

            <AgentChatCard
                chatMessages={chatMessages}
                chatPrompt={chatPrompt}
                chatStatus={chatStatus}
                contextPackPaths={contextPackPaths}
                isSendingPrompt={isSendingPrompt}
                onChatPromptChange={onChatPromptChange}
                onClearChatHistory={onClearChatHistory}
                onClearContextPack={onClearContextPack}
                onRemoveContextPath={onRemoveContextPath}
                onSendPrompt={onSendPrompt}
            />

            <LLMSettingsCard
                isSavingSettings={isSavingSettings}
                isTestingConnection={isTestingConnection}
                onSaveSettings={onSaveSettings}
                onSettingsDraftChange={onSettingsDraftChange}
                onTestConnection={onTestConnection}
                probeResult={probeResult}
                settingsDraft={settingsDraft}
                settingsStatus={settingsStatus}
            />

            <ToolTimeline events={toolEvents} />
        </aside>
    );
}
