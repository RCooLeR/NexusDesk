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
    staleSourcePaths: string[];
    canSaveLatestAssistantArtifact: boolean;
    isSavingChatArtifact: boolean;
    isSendingPrompt: boolean;
    onChatPromptChange: (value: string) => void;
    onClearChatHistory: () => void;
    onClearContextPack: () => void;
    onModelChange: (model: string) => void;
    onRemoveContextPath: (relPath: string) => void;
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
    staleSourcePaths,
    canSaveLatestAssistantArtifact,
    isSavingChatArtifact,
    isSendingPrompt,
    onChatPromptChange,
    onClearChatHistory,
    onClearContextPack,
    onModelChange,
    onRemoveContextPath,
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
                staleSourcePaths={staleSourcePaths}
                canSaveLatestAssistantArtifact={canSaveLatestAssistantArtifact}
                isSavingChatArtifact={isSavingChatArtifact}
                isSendingPrompt={isSendingPrompt}
                onChatPromptChange={onChatPromptChange}
                onClearChatHistory={onClearChatHistory}
                onClearContextPack={onClearContextPack}
                onModelChange={onModelChange}
                onRemoveContextPath={onRemoveContextPath}
                onRunAgent={onRunAgent}
                onSaveLatestAssistantArtifact={onSaveLatestAssistantArtifact}
                onSendPrompt={onSendPrompt}
            />
        </aside>
    );
}
