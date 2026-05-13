import {useEffect, useState} from 'react';
import {GetLLMSettings, GetRecentWorkspaces, GetStartupState} from '../wailsjs/go/main/App';
import {fallbackState} from './data/startupState';
import {NexusDeskShell} from './features/shell/NexusDeskShell';
import type {LLMSettings, RecentWorkspace, StartupState, WorkspaceSnapshot} from './types';
import './App.css';

const fallbackLLMSettings: LLMSettings = {
    providerName: 'Local OpenAI-compatible',
    baseUrl: 'http://localhost:11434/v1',
    model: 'qwen3:8b',
    apiKey: '',
    updatedAt: '',
};

function App() {
    const [state, setState] = useState<StartupState>(fallbackState);
    const [workspace, setWorkspace] = useState<WorkspaceSnapshot | null>(null);
    const [recentWorkspaces, setRecentWorkspaces] = useState<RecentWorkspace[]>([]);
    const [llmSettings, setLLMSettings] = useState<LLMSettings>(fallbackLLMSettings);

    useEffect(() => {
        Promise.resolve()
            .then(() => GetStartupState())
            .then(setState)
            .catch(() => setState(fallbackState));

        Promise.resolve()
            .then(() => GetRecentWorkspaces())
            .then(setRecentWorkspaces)
            .catch(() => setRecentWorkspaces([]));

        Promise.resolve()
            .then(() => GetLLMSettings())
            .then(setLLMSettings)
            .catch(() => setLLMSettings(fallbackLLMSettings));
    }, []);

    return (
        <NexusDeskShell
            state={state}
            workspace={workspace}
            recentWorkspaces={recentWorkspaces}
            llmSettings={llmSettings}
            onWorkspaceChange={setWorkspace}
            onRecentWorkspacesChange={setRecentWorkspaces}
            onLLMSettingsChange={setLLMSettings}
        />
    );
}

export default App;
