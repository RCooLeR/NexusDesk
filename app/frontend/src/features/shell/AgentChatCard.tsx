import {Button, Card} from '../../components/ui';
import type {ChatMessage} from '../../types';
import {ChatMessageContent} from './ChatMessageContent';

type AgentChatCardProps = {
    chatMessages: ChatMessage[];
    chatPrompt: string;
    chatStatus: string;
    contextPackPaths: string[];
    canSaveLatestAssistantArtifact: boolean;
    isSavingChatArtifact: boolean;
    isSendingPrompt: boolean;
    onChatPromptChange: (value: string) => void;
    onClearChatHistory: () => void;
    onClearContextPack: () => void;
    onRemoveContextPath: (relPath: string) => void;
    onSaveLatestAssistantArtifact: () => void;
    onSendPrompt: () => void;
};

export function AgentChatCard({
    chatMessages,
    chatPrompt,
    chatStatus,
    contextPackPaths,
    canSaveLatestAssistantArtifact,
    isSavingChatArtifact,
    isSendingPrompt,
    onChatPromptChange,
    onClearChatHistory,
    onClearContextPack,
    onRemoveContextPath,
    onSaveLatestAssistantArtifact,
    onSendPrompt,
}: AgentChatCardProps) {
    return (
        <Card className="chat-card">
            <div className="chat-card-header">
                <div>
                    <strong>Conversation</strong>
                    <small>{chatStatus}</small>
                </div>
                <div className="chat-header-actions">
                    <Button
                        disabled={!canSaveLatestAssistantArtifact || isSavingChatArtifact}
                        onClick={onSaveLatestAssistantArtifact}
                        title="Save latest assistant answer as a Markdown artifact"
                        variant="subtle"
                    >
                        {isSavingChatArtifact ? 'Saving...' : 'Save answer'}
                    </Button>
                    {chatMessages.length > 0 && (
                        <Button onClick={onClearChatHistory} variant="subtle">Clear</Button>
                    )}
                </div>
            </div>
            <div className="chat-thread">
                {chatMessages.length === 0 ? (
                    <div className="assistant-message">
                        <strong>NexusDesk</strong>
                        <ChatMessageContent content="Ready to connect a model, read selected files, and turn source context into auditable work." />
                    </div>
                ) : (
                    chatMessages.map((message, index) => (
                        <div className={message.role === 'user' ? 'user-message' : 'assistant-message'} key={`${message.role}-${message.createdAt}-${index}`}>
                            <strong>{message.role === 'user' ? 'You' : 'NexusDesk'}</strong>
                            <ChatMessageContent content={message.content} />
                            {message.contextRelPath && <small>{message.contextRelPath}</small>}
                        </div>
                    ))
                )}
            </div>
            {contextPackPaths.length > 0 && (
                <div className="context-pack-list">
                    <div className="context-pack-heading">
                        <strong>{contextPackPaths.length} pinned</strong>
                        <Button onClick={onClearContextPack} variant="subtle">Clear pack</Button>
                    </div>
                    <div className="context-pack-items">
                        {contextPackPaths.map((relPath) => (
                            <button key={relPath} onClick={() => onRemoveContextPath(relPath)} title={`Remove ${contextLabel(relPath)}`}>
                                <span>{contextLabel(relPath)}</span>
                                <strong>x</strong>
                            </button>
                        ))}
                    </div>
                </div>
            )}
            <div className="prompt-box">
                <textarea
                    aria-label="Ask about the workspace"
                    onChange={(event) => onChatPromptChange(event.target.value)}
                    onKeyDown={(event) => {
                        if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
                            onSendPrompt();
                        }
                    }}
                    placeholder="Ask about the workspace..."
                    rows={4}
                    value={chatPrompt}
                />
                <Button disabled={isSendingPrompt} onClick={onSendPrompt} title="Send prompt" variant="primary">
                    {isSendingPrompt ? 'Sending...' : 'Send'}
                </Button>
            </div>
        </Card>
    );
}

function contextLabel(relPath: string) {
    return relPath === '.' ? 'Workspace root' : relPath;
}
