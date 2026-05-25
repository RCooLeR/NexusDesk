import {useEffect, useState} from 'react';
import {GetLLMSettings, GetRecentWorkspaces, GetStartupState} from '../wailsjs/go/main/App';
import {fallbackState} from './data/startupState';
import {NexusDeskShell} from './features/shell/NexusDeskShell';
import type {FileNode, LLMSettings, RecentWorkspace, ScanStatus, StartupState, WorkspaceSnapshot} from './types';
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
            onWorkspaceChange={(snapshot) => setWorkspace(sanitizeWorkspaceSnapshot(snapshot))}
            onRecentWorkspacesChange={setRecentWorkspaces}
            onLLMSettingsChange={setLLMSettings}
        />
    );
}

function sanitizeWorkspaceSnapshot(snapshot: WorkspaceSnapshot): WorkspaceSnapshot {
    const nodes = Array.isArray(snapshot.nodes)
        ? snapshot.nodes
            .map((node) => sanitizeWorkspaceNode(node))
            .filter((node): node is FileNode => node !== null)
        : [];
    return {
        ...snapshot,
        nodes,
        scan: sanitizeScanStatus(snapshot.scan),
    };
}

function sanitizeScanStatus(scan: ScanStatus | null | undefined): ScanStatus {
    return {
        included: numericScanValue(scan?.included),
        ignored: numericScanValue(scan?.ignored),
        depthSkipped: numericScanValue(scan?.depthSkipped),
        entrySkipped: numericScanValue(scan?.entrySkipped),
        unreadable: numericScanValue(scan?.unreadable),
        maxDepth: numericScanValue(scan?.maxDepth),
        maxEntries: numericScanValue(scan?.maxEntries),
        ignoredSamples: safeStringArray(scan?.ignoredSamples),
        skippedSamples: safeStringArray(scan?.skippedSamples),
    };
}

function numericScanValue(value: unknown) {
    return typeof value === 'number' && Number.isFinite(value) ? value : 0;
}

function safeStringArray(value: unknown) {
    return Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string') : [];
}

function sanitizeWorkspaceNode(node: unknown): FileNode | null {
    if (!node || typeof node !== 'object') {
        return null;
    }

    const candidate = node as Partial<FileNode>;
    const relPath = typeof candidate.relPath === 'string'
        ? candidate.relPath.replace(/\\/g, '/').trim().replace(/^\/+/, '')
        : '';

    if (!relPath) {
        return null;
    }

    const depth = typeof candidate.depth === 'number' && candidate.depth >= 0
        ? candidate.depth
        : relPath.split('/').filter(Boolean).length;

    return {
        name: typeof candidate.name === 'string' && candidate.name.trim() !== '' ? candidate.name : relPath.split('/').at(-1) ?? relPath,
        path: typeof candidate.path === 'string' && candidate.path.trim() !== '' ? candidate.path : relPath,
        relPath,
        kind: candidate.kind === 'directory' ? 'directory' : 'file',
        fileType: typeof candidate.fileType === 'string' && candidate.fileType.trim() !== '' ? candidate.fileType : 'file',
        depth,
        meta: typeof candidate.meta === 'string' ? candidate.meta : 'File',
    };
}

export default App;
