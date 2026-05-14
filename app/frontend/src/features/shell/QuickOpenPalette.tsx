import {useEffect, useMemo, useRef, useState} from 'react';
import {brandAssets} from '../../brand/assets';
import type {FileNode, FilePreview, WorkspaceSnapshot} from '../../types';

type QuickOpenPaletteProps = {
    activeFile: string;
    isOpen: boolean;
    onClose: () => void;
    onQueryChange: (value: string) => void;
    onSelectNode: (node: FileNode) => void;
    onSelectTab: (relPath: string) => void;
    openTabs: FilePreview[];
    query: string;
    workspace: WorkspaceSnapshot | null;
};

type QuickOpenEntry = {
    id: string;
    label: string;
    detail: string;
    fileType: string;
    kind: string;
    node?: FileNode;
    relPath: string;
    tab?: FilePreview;
};

const maxQuickOpenResults = 80;

export function QuickOpenPalette({
    activeFile,
    isOpen,
    onClose,
    onQueryChange,
    onSelectNode,
    onSelectTab,
    openTabs,
    query,
    workspace,
}: QuickOpenPaletteProps) {
    const inputRef = useRef<HTMLInputElement>(null);
    const [selectedIndex, setSelectedIndex] = useState(0);
    const entries = useMemo(() => buildQuickOpenEntries(workspace, openTabs, query), [openTabs, query, workspace]);
    const selectedEntry = entries[selectedIndex] ?? null;

    useEffect(() => {
        if (!isOpen) {
            return;
        }
        setSelectedIndex(0);
        window.setTimeout(() => inputRef.current?.focus(), 0);
    }, [isOpen]);

    useEffect(() => {
        setSelectedIndex(0);
    }, [query]);

    if (!isOpen) {
        return null;
    }

    function chooseEntry(entry: QuickOpenEntry | null) {
        if (!entry) {
            return;
        }

        if (entry.tab) {
            onSelectTab(entry.tab.relPath);
        } else if (entry.node) {
            onSelectNode(entry.node);
        }
        onClose();
    }

    return (
        <div className="quick-open-backdrop" onMouseDown={onClose}>
            <section
                aria-label="Quick open"
                className="quick-open"
                onMouseDown={(event) => event.stopPropagation()}
            >
                <div className="quick-open-input-row">
                    <img src={brandAssets.icons.code} alt="" />
                    <input
                        aria-label="Quick open query"
                        onChange={(event) => onQueryChange(event.target.value)}
                        onKeyDown={(event) => {
                            if (event.key === 'Escape') {
                                event.preventDefault();
                                onClose();
                            }
                            if (event.key === 'ArrowDown') {
                                event.preventDefault();
                                setSelectedIndex((current) => Math.min(current + 1, Math.max(entries.length - 1, 0)));
                            }
                            if (event.key === 'ArrowUp') {
                                event.preventDefault();
                                setSelectedIndex((current) => Math.max(current - 1, 0));
                            }
                            if (event.key === 'Enter') {
                                event.preventDefault();
                                chooseEntry(selectedEntry);
                            }
                        }}
                        placeholder={workspace ? 'Open file, folder, dataset, artifact...' : 'Open a workspace first'}
                        ref={inputRef}
                        value={query}
                    />
                </div>
                <div className="quick-open-results">
                    {entries.length === 0 ? (
                        <div className="quick-open-empty">
                            {workspace ? 'No matching workspace items.' : 'No workspace opened.'}
                        </div>
                    ) : entries.map((entry, index) => (
                        <button
                            className={[
                                'quick-open-result',
                                index === selectedIndex ? 'selected' : '',
                                entry.relPath === activeFile ? 'active' : '',
                            ].filter(Boolean).join(' ')}
                            key={entry.id}
                            onClick={() => chooseEntry(entry)}
                            onMouseEnter={() => setSelectedIndex(index)}
                        >
                            <span className={`file-glyph ${entry.kind}`}>
                                <img src={iconForEntry(entry)} alt="" />
                            </span>
                            <span>
                                <strong>{entry.label}</strong>
                                <small>{entry.detail}</small>
                            </span>
                            <em>{entry.tab ? 'tab' : entry.fileType}</em>
                        </button>
                    ))}
                </div>
            </section>
        </div>
    );
}

function buildQuickOpenEntries(workspace: WorkspaceSnapshot | null, openTabs: FilePreview[], query: string) {
    const trimmedQuery = query.trim();
    const tabEntries = openTabs.map((tab): QuickOpenEntry => ({
        id: `tab:${tab.relPath}`,
        label: tab.name,
        detail: tab.relPath,
        fileType: tab.kind === 'pdf' ? 'pdf' : tab.fileType,
        kind: tab.kind,
        relPath: tab.relPath,
        tab,
    }));

    const tabPaths = new Set(openTabs.map((tab) => tab.relPath));
    const nodeEntries = (workspace?.nodes ?? []).map((node): QuickOpenEntry => ({
        id: `node:${node.relPath}`,
        label: node.name,
        detail: node.relPath,
        fileType: node.fileType,
        kind: node.kind,
        node,
        relPath: node.relPath,
    })).filter((entry) => !tabPaths.has(entry.relPath));

    const entries = [...tabEntries, ...nodeEntries];
    return entries
        .map((entry, index) => ({entry, score: scoreQuickOpenEntry(entry, trimmedQuery), index}))
        .filter((result) => result.score > 0)
        .sort((a, b) => b.score - a.score || a.index - b.index)
        .slice(0, maxQuickOpenResults)
        .map((result) => result.entry);
}

function scoreQuickOpenEntry(entry: QuickOpenEntry, query: string) {
    if (!query) {
        return entry.tab ? 80 : entry.kind === 'directory' ? 35 : 50;
    }

    const needle = query.toLowerCase();
    const label = entry.label.toLowerCase();
    const path = entry.relPath.toLowerCase();
    const compactPath = path.replace(/[\\/_.-]+/g, '');
    const compactNeedle = needle.replace(/[\\/_.-]+/g, '');

    if (label === needle || path === needle) {
        return 220;
    }
    if (label.startsWith(needle)) {
        return 180;
    }
    if (path.startsWith(needle)) {
        return 160;
    }
    if (label.includes(needle)) {
        return 130;
    }
    if (path.includes(needle)) {
        return 105;
    }
    if (compactNeedle && compactPath.includes(compactNeedle)) {
        return 75;
    }
    return 0;
}

function iconForEntry(entry: QuickOpenEntry) {
    if (entry.fileType === 'code') {
        return brandAssets.icons.code;
    }
    if (entry.fileType === 'data') {
        return brandAssets.icons.data;
    }
    if (entry.fileType === 'document' || entry.kind === 'pdf') {
        return brandAssets.icons.documents;
    }
    return brandAssets.icons.documents;
}
