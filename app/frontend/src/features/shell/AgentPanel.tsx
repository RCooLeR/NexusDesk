import {brandAssets} from '../../brand/assets';
import type {AgentToolDescriptor, AgentToolPlanItem, AgentToolRunRecord, ChatMessage, ContextPreview, LLMProbeResult, LLMSettings, ToolEvent} from '../../types';
import {AgentChatCard} from './AgentChatCard';
import {AgentToolPlanCard} from './AgentToolPlanCard';
import {LLMSettingsCard} from './LLMSettingsCard';
import {ToolTimeline} from './ToolTimeline';

type AgentPanelProps = {
    chatMessages: ChatMessage[];
    chatPrompt: string;
    chatStatus: string;
    contextPackPreview: ContextPreview | null;
    contextPackPaths: string[];
    agentTools: AgentToolDescriptor[];
    agentToolPlan: AgentToolPlanItem[];
    agentToolRuns: AgentToolRunRecord[];
    canSaveLatestAssistantArtifact: boolean;
    isSavingSettings: boolean;
    isSavingChatArtifact: boolean;
    isSendingPrompt: boolean;
    isTestingConnection: boolean;
    isRunningAgentTool: boolean;
    onChatPromptChange: (value: string) => void;
    onClearChatHistory: () => void;
    onClearContextPack: () => void;
    onDryRunAgentTool: (item: AgentToolPlanItem) => void;
    onExecuteAgentTool: (item: AgentToolPlanItem) => void;
    onRefreshAgentPlan: () => void;
    onRemoveContextPath: (relPath: string) => void;
    onSaveLatestAssistantArtifact: () => void;
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
    contextPackPreview,
    contextPackPaths,
    agentTools,
    agentToolPlan,
    agentToolRuns,
    canSaveLatestAssistantArtifact,
    isSavingSettings,
    isSavingChatArtifact,
    isSendingPrompt,
    isTestingConnection,
    isRunningAgentTool,
    onChatPromptChange,
    onClearChatHistory,
    onClearContextPack,
    onDryRunAgentTool,
    onExecuteAgentTool,
    onRefreshAgentPlan,
    onRemoveContextPath,
    onSaveLatestAssistantArtifact,
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
                contextPackPreview={contextPackPreview}
                contextPackPaths={contextPackPaths}
                canSaveLatestAssistantArtifact={canSaveLatestAssistantArtifact}
                isSavingChatArtifact={isSavingChatArtifact}
                isSendingPrompt={isSendingPrompt}
                onChatPromptChange={onChatPromptChange}
                onClearChatHistory={onClearChatHistory}
                onClearContextPack={onClearContextPack}
                onRemoveContextPath={onRemoveContextPath}
                onSaveLatestAssistantArtifact={onSaveLatestAssistantArtifact}
                onSendPrompt={onSendPrompt}
            />

            <AgentToolPlanCard
                tools={agentTools}
                planItems={agentToolPlan}
                runs={agentToolRuns}
                isRunning={isRunningAgentTool}
                onDryRun={onDryRunAgentTool}
                onExecute={onExecuteAgentTool}
                onRefreshPlan={onRefreshAgentPlan}
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
