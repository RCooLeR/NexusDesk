import {Button, Card} from '../../components/ui';
import type {ChatMessage} from '../../types';

type AgentChatCardProps = {
    chatMessages: ChatMessage[];
    chatPrompt: string;
    chatStatus: string;
    isSendingPrompt: boolean;
    onChatPromptChange: (value: string) => void;
    onClearChatHistory: () => void;
    onSendPrompt: () => void;
};

export function AgentChatCard({
    chatMessages,
    chatPrompt,
    chatStatus,
    isSendingPrompt,
    onChatPromptChange,
    onClearChatHistory,
    onSendPrompt,
}: AgentChatCardProps) {
    return (
        <Card className="chat-card">
            <div className="chat-thread">
                {chatMessages.length === 0 ? (
                    <div className="assistant-message">
                        <strong>NexusDesk</strong>
                        <p>Ready to connect a model, read selected files, and turn source context into auditable work.</p>
                    </div>
                ) : (
                    chatMessages.slice(-4).map((message, index) => (
                        <div className={message.role === 'user' ? 'user-message' : 'assistant-message'} key={`${message.role}-${message.createdAt}-${index}`}>
                            <strong>{message.role === 'user' ? 'You' : 'NexusDesk'}</strong>
                            <p>{message.content || 'Receiving response...'}</p>
                            {message.contextRelPath && <small>{message.contextRelPath}</small>}
                        </div>
                    ))
                )}
            </div>
            {chatMessages.length > 0 && (
                <div className="chat-actions">
                    <Button onClick={onClearChatHistory} variant="subtle">Clear chat</Button>
                </div>
            )}
            <div className="prompt-box">
                <input
                    aria-label="Ask about the workspace"
                    onChange={(event) => onChatPromptChange(event.target.value)}
                    onKeyDown={(event) => {
                        if (event.key === 'Enter') {
                            onSendPrompt();
                        }
                    }}
                    placeholder="Ask about the workspace..."
                    value={chatPrompt}
                />
                <Button disabled={isSendingPrompt} onClick={onSendPrompt} title="Send prompt" variant="primary">
                    {isSendingPrompt ? 'Sending...' : 'Send'}
                </Button>
            </div>
            <small className="chat-status">{chatStatus}</small>
        </Card>
    );
}
