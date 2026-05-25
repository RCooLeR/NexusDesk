import {useState} from 'react';
import {productBrand} from '../../brand/assets';
import {Button, Card} from '../../components/ui';
import type {ChatMessage, ContextPreview} from '../../types';
import {ChatMessageContent} from './ChatMessageContent';
import {recommendedModelOptions} from './llmModelCatalog';

type AgentChatCardProps = {
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
};

export function AgentChatCard({
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
}: AgentChatCardProps) {
    const [submitMode, setSubmitMode] = useState<'ask' | 'agent'>('ask');
    const chatModelOptions = recommendedModelOptions.map((option) => ({id: option.id, label: option.chatLabel}));
    const modelOptions = currentModel && !chatModelOptions.some((option) => option.id === currentModel)
        ? [{id: currentModel, label: currentModel}, ...chatModelOptions]
        : chatModelOptions;
    const submit = () => {
        if (submitMode === 'agent') {
            onRunAgent();
            return;
        }
        onSendPrompt();
    };

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
                        <strong>{productBrand.shortName}</strong>
                        <ChatMessageContent content="Ready to connect a model, read selected files, and turn source context into auditable work." />
                    </div>
                ) : (
                    chatMessages.map((message, index) => (
                        <div className={message.role === 'user' ? 'user-message' : 'assistant-message'} key={`${message.role}-${message.createdAt}-${index}`}>
                            <strong>{message.role === 'user' ? 'You' : productBrand.shortName}</strong>
                            <ChatMessageContent content={message.content} />
                            {message.contextRelPath && <small>{message.contextRelPath}</small>}
                            {messageHasStaleSources(message, staleSourcePaths) && (
                                <small className="stale-source-warning">Context changed since this answer was created.</small>
                            )}
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
                    {contextPackPreview && (
                        <div className="context-pack-preview">
                            <small>
                                {contextPackPreview.message}
                            </small>
                            {contextPackPreview.files.length > 0 && (
                                <ul>
                                    {contextPackPreview.files.slice(0, 8).map((file) => (
                                        <li className={staleSourcePaths.includes(file.relPath) ? 'stale' : ''} key={file.relPath}>
                                            {file.relPath}{staleSourcePaths.includes(file.relPath) ? ' / changed' : ''}
                                        </li>
                                    ))}
                                    {contextPackPreview.files.length > 8 && (
                                        <li>{contextPackPreview.files.length - 8} more files</li>
                                    )}
                                </ul>
                            )}
                        </div>
                    )}
                </div>
            )}
            <div className="prompt-box">
                <div className="composer-shell">
                    <textarea
                        aria-label="Ask about the workspace"
                        onChange={(event) => onChatPromptChange(event.target.value)}
                        onKeyDown={(event) => {
                            if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
                                submit();
                            }
                        }}
                        placeholder={`Message ${productBrand.shortName}`}
                        rows={5}
                        value={chatPrompt}
                    />
                    <div className="composer-controls">
                        <select aria-label="Chat model" value={currentModel} onChange={(event) => onModelChange(event.target.value)}>
                            {modelOptions.map((option) => (
                                <option key={option.id} value={option.id}>{option.label}</option>
                            ))}
                        </select>
                        <select aria-label="Submit mode" value={submitMode} onChange={(event) => setSubmitMode(event.target.value === 'agent' ? 'agent' : 'ask')}>
                            <option value="ask">Ask</option>
                            <option value="agent">Agent</option>
                        </select>
                        <button className="composer-submit" disabled={isSendingPrompt || !chatPrompt.trim()} onClick={submit} title={submitMode === 'agent' ? 'Run agent' : 'Send prompt'} type="button">
                            {isSendingPrompt ? '...' : String.fromCharCode(8593)}
                        </button>
                    </div>
                </div>
            </div>
        </Card>
    );
}

function contextLabel(relPath: string) {
    return relPath === '.' ? 'Workspace root' : relPath;
}

function messageHasStaleSources(message: ChatMessage, staleSourcePaths: string[]) {
    if (staleSourcePaths.length === 0) {
        return false;
    }
    return (message.sourcePaths ?? []).some((sourcePath) => staleSourcePaths.includes(sourcePath)) ||
        Boolean(message.contextRelPath && staleSourcePaths.includes(message.contextRelPath));
}
