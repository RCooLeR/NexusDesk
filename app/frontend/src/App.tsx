import {useEffect, useState} from 'react';
import {GetRecentWorkspaces, GetStartupState} from '../wailsjs/go/main/App';
import {fallbackState} from './data/startupState';
import {NexusDeskShell} from './features/shell/NexusDeskShell';
import type {RecentWorkspace, StartupState, WorkspaceSnapshot} from './types';
import './App.css';

function App() {
    const [state, setState] = useState<StartupState>(fallbackState);
    const [workspace, setWorkspace] = useState<WorkspaceSnapshot | null>(null);
    const [recentWorkspaces, setRecentWorkspaces] = useState<RecentWorkspace[]>([]);

    useEffect(() => {
        Promise.resolve()
            .then(() => GetStartupState())
            .then(setState)
            .catch(() => setState(fallbackState));

        Promise.resolve()
            .then(() => GetRecentWorkspaces())
            .then(setRecentWorkspaces)
            .catch(() => setRecentWorkspaces([]));
    }, []);

    return (
        <NexusDeskShell
            state={state}
            workspace={workspace}
            recentWorkspaces={recentWorkspaces}
            onWorkspaceChange={setWorkspace}
            onRecentWorkspacesChange={setRecentWorkspaces}
        />
    );
}

export default App;
