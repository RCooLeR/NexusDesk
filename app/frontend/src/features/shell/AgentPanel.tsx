import {brandAssets, productBrand} from '../../brand/assets';
import type {AssistantProfile, ChatMessage, ContextPreview} from '../../types';
import {AgentChatCard} from './AgentChatCard';

type AgentPanelProps = {
    assistantProfile: AssistantProfile;
    assistantProfileDraft: AssistantProfile;
    chatMessages: ChatMessage[];
    chatPrompt: string;
    chatStatus: string;
    contextPackPreview: ContextPreview | null;
    contextPackPaths: string[];
    currentModel: string;
    hasSelectableContext: boolean;
    staleSourcePaths: string[];
    canSaveLatestAssistantArtifact: boolean;
    canRetryLatestAssistant: boolean;
    isSavingAssistantProfile: boolean;
    isSavingChatArtifact: boolean;
    isSendingPrompt: boolean;
    onChatPromptChange: (value: string) => void;
    onClearChatHistory: () => void;
    onClearContextPack: () => void;
    onCompareLatestAssistant: () => void;
    onAssistantProfileDraftChange: (profile: AssistantProfile) => void;
    onModelChange: (model: string) => void;
    onRemoveContextPath: (relPath: string) => void;
    onRetryLatestAssistant: () => void;
    onRunAgent: () => void;
    onSaveAssistantProfile: () => void;
    onSaveLatestAssistantArtifact: () => void;
    onSendPrompt: () => void;
    tagline: string;
};

export function AgentPanel({
    assistantProfile,
    assistantProfileDraft,
    chatMessages,
    chatPrompt,
    chatStatus,
    contextPackPreview,
    contextPackPaths,
    currentModel,
    hasSelectableContext,
    staleSourcePaths,
    canSaveLatestAssistantArtifact,
    canRetryLatestAssistant,
    isSavingAssistantProfile,
    isSavingChatArtifact,
    isSendingPrompt,
    onChatPromptChange,
    onClearChatHistory,
    onClearContextPack,
    onCompareLatestAssistant,
    onAssistantProfileDraftChange,
    onModelChange,
    onRemoveContextPath,
    onRetryLatestAssistant,
    onRunAgent,
    onSaveAssistantProfile,
    onSaveLatestAssistantArtifact,
    onSendPrompt,
    tagline,
}: AgentPanelProps) {
    return (
        <aside className="agent-panel">
            <header>
                <h2>{productBrand.shortName} Assistant</h2>
                <span>{tagline || productBrand.tagline}</span>
            </header>

            <AgentChatCard
                assistantProfile={assistantProfile}
                assistantProfileDraft={assistantProfileDraft}
                chatMessages={chatMessages}
                chatPrompt={chatPrompt}
                chatStatus={chatStatus}
                contextPackPreview={contextPackPreview}
                contextPackPaths={contextPackPaths}
                currentModel={currentModel}
                hasSelectableContext={hasSelectableContext}
                staleSourcePaths={staleSourcePaths}
                canSaveLatestAssistantArtifact={canSaveLatestAssistantArtifact}
                canRetryLatestAssistant={canRetryLatestAssistant}
                isSavingAssistantProfile={isSavingAssistantProfile}
                isSavingChatArtifact={isSavingChatArtifact}
                isSendingPrompt={isSendingPrompt}
                onChatPromptChange={onChatPromptChange}
                onClearChatHistory={onClearChatHistory}
                onClearContextPack={onClearContextPack}
                onCompareLatestAssistant={onCompareLatestAssistant}
                onAssistantProfileDraftChange={onAssistantProfileDraftChange}
                onModelChange={onModelChange}
                onRemoveContextPath={onRemoveContextPath}
                onRetryLatestAssistant={onRetryLatestAssistant}
                onRunAgent={onRunAgent}
                onSaveAssistantProfile={onSaveAssistantProfile}
                onSaveLatestAssistantArtifact={onSaveLatestAssistantArtifact}
                onSendPrompt={onSendPrompt}
            />
        </aside>
    );
}
