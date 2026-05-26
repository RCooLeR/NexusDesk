import {brandAssets, productBrand} from '../../brand/assets';
import type {ChatMessage, ContextPreview} from '../../types';
import {AgentChatCard} from './AgentChatCard';

type AgentPanelProps = {
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
    isSavingChatArtifact: boolean;
    isSendingPrompt: boolean;
    onChatPromptChange: (value: string) => void;
    onClearChatHistory: () => void;
    onClearContextPack: () => void;
    onCompareLatestAssistant: () => void;
    onModelChange: (model: string) => void;
    onRemoveContextPath: (relPath: string) => void;
    onRetryLatestAssistant: () => void;
    onRunAgent: () => void;
    onSaveLatestAssistantArtifact: () => void;
    onSendPrompt: () => void;
    tagline: string;
};

export function AgentPanel({
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
    isSavingChatArtifact,
    isSendingPrompt,
    onChatPromptChange,
    onClearChatHistory,
    onClearContextPack,
    onCompareLatestAssistant,
    onModelChange,
    onRemoveContextPath,
    onRetryLatestAssistant,
    onRunAgent,
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
                isSavingChatArtifact={isSavingChatArtifact}
                isSendingPrompt={isSendingPrompt}
                onChatPromptChange={onChatPromptChange}
                onClearChatHistory={onClearChatHistory}
                onClearContextPack={onClearContextPack}
                onCompareLatestAssistant={onCompareLatestAssistant}
                onModelChange={onModelChange}
                onRemoveContextPath={onRemoveContextPath}
                onRetryLatestAssistant={onRetryLatestAssistant}
                onRunAgent={onRunAgent}
                onSaveLatestAssistantArtifact={onSaveLatestAssistantArtifact}
                onSendPrompt={onSendPrompt}
            />
        </aside>
    );
}
