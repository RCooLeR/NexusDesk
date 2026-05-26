import {useState} from 'react';
import {productBrand} from '../../brand/assets';
import {Button, Card} from '../../components/ui';
import type {AssistantProfile, ChatMessage, ContextPreview} from '../../types';
import {ChatMessageContent} from './ChatMessageContent';
import {recommendedModelOptions} from './llmModelCatalog';

type AgentChatCardProps = {
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
    onAssistantProfileDraftChange: (profile: AssistantProfile) => void;
    onClearChatHistory: () => void;
    onClearContextPack: () => void;
    onCompareLatestAssistant: () => void;
    onModelChange: (model: string) => void;
    onRemoveContextPath: (relPath: string) => void;
    onRetryLatestAssistant: () => void;
    onRunAgent: (approveFileWrites: boolean) => void;
    onSaveAssistantProfile: () => void;
    onSaveLatestAssistantArtifact: () => void;
    onSendPrompt: () => void;
};

export function AgentChatCard({
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
    onAssistantProfileDraftChange,
    onClearChatHistory,
    onClearContextPack,
    onCompareLatestAssistant,
    onModelChange,
    onRemoveContextPath,
    onRetryLatestAssistant,
    onRunAgent,
    onSaveAssistantProfile,
    onSaveLatestAssistantArtifact,
    onSendPrompt,
}: AgentChatCardProps) {
    const [submitMode, setSubmitMode] = useState<'ask' | 'agent'>('ask');
    const [approveFileWrites, setApproveFileWrites] = useState(false);
    const activeProfile = assistantProfileDraft.promptProfiles.find((profile) => profile.id === assistantProfileDraft.activeProfileId);
    const savedProfile = assistantProfile.promptProfiles.find((profile) => profile.id === assistantProfile.activeProfileId);
    const hasProfileChanges = assistantProfileDraft.memory !== assistantProfile.memory || assistantProfileDraft.activeProfileId !== assistantProfile.activeProfileId;
    const chatModelOptions = recommendedModelOptions.map((option) => ({id: option.id, label: option.chatLabel}));
    const modelOptions = currentModel && !chatModelOptions.some((option) => option.id === currentModel)
        ? [{id: currentModel, label: currentModel}, ...chatModelOptions]
        : chatModelOptions;
    const submit = () => {
        if (submitMode === 'agent') {
            onRunAgent(approveFileWrites);
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
                        disabled={!canRetryLatestAssistant || isSendingPrompt}
                        onClick={onRetryLatestAssistant}
                        title="Retry the previous prompt with the same source context"
                        variant="subtle"
                    >
                        Retry
                    </Button>
                    <Button
                        disabled={!canRetryLatestAssistant || isSendingPrompt}
                        onClick={onCompareLatestAssistant}
                        title="Compare the previous answer with a fresh model response"
                        variant="subtle"
                    >
                        Compare
                    </Button>
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
                            {messageHasWeakEvidence(message) && (
                                <small className="weak-evidence-warning">No explicit source context is attached to this answer.</small>
                            )}
                        </div>
                    ))
                )}
            </div>
            {contextPackPaths.length === 0 && !hasSelectableContext && (
                <div className="missing-context-warning">
                    <strong>Missing context</strong>
                    <span>Select or pin a file, folder, diff, dataset, or artifact before asking for grounded analysis.</span>
                </div>
            )}
            <div className="assistant-profile-panel">
                <div>
                    <strong>Prompt profile</strong>
                    <small>{profileStatusLabel(activeProfile?.name, savedProfile?.name, hasProfileChanges)}</small>
                </div>
                <select
                    aria-label="Assistant prompt profile"
                    onChange={(event) => onAssistantProfileDraftChange({...assistantProfileDraft, activeProfileId: event.target.value})}
                    value={assistantProfileDraft.activeProfileId}
                >
                    {assistantProfileDraft.promptProfiles.map((profile) => (
                        <option key={profile.id} value={profile.id}>{profile.name}</option>
                    ))}
                </select>
                <textarea
                    aria-label="Assistant memory"
                    onChange={(event) => onAssistantProfileDraftChange({...assistantProfileDraft, memory: event.target.value})}
                    placeholder="Memory: preferred style, project conventions, recurring facts..."
                    rows={3}
                    value={assistantProfileDraft.memory}
                />
                <Button disabled={isSavingAssistantProfile} onClick={onSaveAssistantProfile} variant="subtle">
                    {isSavingAssistantProfile ? 'Saving...' : 'Save memory'}
                </Button>
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
                        {submitMode === 'agent' && (
                            <label className="composer-write-toggle" title="Ask for explicit approval, then let the agent create or update workspace files through safe write tools. Shell commands stay disabled.">
                                <input
                                    checked={approveFileWrites}
                                    onChange={(event) => setApproveFileWrites(event.target.checked)}
                                    type="checkbox"
                                />
                                Writes
                            </label>
                        )}
                        <button className="composer-submit" disabled={isSendingPrompt || !chatPrompt.trim()} onClick={submit} title={submitMode === 'agent' ? 'Run agent' : 'Send prompt'} type="button">
                            {isSendingPrompt ? '...' : String.fromCharCode(8593)}
                        </button>
                    </div>
                </div>
            </div>
        </Card>
    );
}

function profileStatusLabel(activeName = 'Assistant guidance', savedName = activeName, hasChanges = false) {
    return hasChanges ? `Unsaved: ${activeName} (saved ${savedName})` : `Active: ${savedName}`;
}

function messageHasWeakEvidence(message: ChatMessage) {
    if (message.role !== 'assistant' || message.contextRelPath === 'agent') {
        return false;
    }
    return !message.contextRelPath && (!message.sourcePaths || message.sourcePaths.length === 0);
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
