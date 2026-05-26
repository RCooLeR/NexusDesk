import {useEffect, useMemo, useRef, useState} from 'react';
import {FontAwesomeIcon} from '@fortawesome/react-fontawesome';
import {brandAssets} from '../../brand/assets';

export type CommandAction = {
    detail: string;
    disabled?: boolean;
    group: string;
    id: string;
    run: () => void;
    shortcut?: string;
    title: string;
};

type CommandPaletteProps = {
    commands: CommandAction[];
    isOpen: boolean;
    onClose: () => void;
    onQueryChange: (value: string) => void;
    query: string;
};

const maxCommandResults = 60;

export function CommandPalette({commands, isOpen, onClose, onQueryChange, query}: CommandPaletteProps) {
    const inputRef = useRef<HTMLInputElement>(null);
    const [selectedIndex, setSelectedIndex] = useState(0);
    const entries = useMemo(() => filterCommands(commands, query), [commands, query]);
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

    function chooseCommand(command: CommandAction | null) {
        if (!command || command.disabled) {
            return;
        }
        command.run();
        onClose();
    }

    return (
        <div className="quick-open-backdrop" onMouseDown={onClose}>
            <section
                aria-label="Command palette"
                className="quick-open command-palette"
                onMouseDown={(event) => event.stopPropagation()}
            >
                <div className="quick-open-input-row">
                    <FontAwesomeIcon icon={brandAssets.icons.ai} />
                    <input
                        aria-label="Command palette query"
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
                                chooseCommand(selectedEntry);
                            }
                        }}
                        placeholder="Run command..."
                        ref={inputRef}
                        value={query}
                    />
                </div>
                <div className="quick-open-results command-results">
                    {entries.length === 0 ? (
                        <div className="quick-open-empty">No matching commands.</div>
                    ) : entries.map((entry, index) => (
                        <button
                            className={[
                                'quick-open-result',
                                'command-result',
                                index === selectedIndex ? 'selected' : '',
                            ].filter(Boolean).join(' ')}
                            disabled={entry.disabled}
                            key={entry.id}
                            onClick={() => chooseCommand(entry)}
                            onMouseEnter={() => setSelectedIndex(index)}
                        >
                            <span className="command-copy">
                                <span className="command-title-row">
                                    <strong>{entry.title}</strong>
                                    <small className="command-group">{entry.group}</small>
                                </span>
                                <small>{entry.detail}</small>
                            </span>
                            {entry.shortcut ? <em className="command-shortcut">{entry.shortcut}</em> : null}
                        </button>
                    ))}
                </div>
            </section>
        </div>
    );
}

function filterCommands(commands: CommandAction[], query: string) {
    const trimmedQuery = query.trim();
    return commands
        .map((command, index) => ({command, index, score: scoreCommand(command, trimmedQuery)}))
        .filter((result) => result.score > 0)
        .sort((a, b) => b.score - a.score || Number(a.command.disabled) - Number(b.command.disabled) || a.index - b.index)
        .slice(0, maxCommandResults)
        .map((result) => result.command);
}

function scoreCommand(command: CommandAction, query: string) {
    if (!query) {
        return command.disabled ? 30 : 80;
    }

    const needle = query.toLowerCase();
    const title = command.title.toLowerCase();
    const detail = command.detail.toLowerCase();
    const group = command.group.toLowerCase();
    const shortcut = (command.shortcut ?? '').toLowerCase();
    const compactNeedle = needle.replace(/\s+/g, '');
    const compactTitle = title.replace(/\s+/g, '');

    if (title === needle) {
        return 240;
    }
    if (title.startsWith(needle)) {
        return 200;
    }
    if (group.includes(needle)) {
        return 160;
    }
    if (title.includes(needle)) {
        return 140;
    }
    if (detail.includes(needle) || shortcut.includes(needle)) {
        return 100;
    }
    if (compactNeedle && compactTitle.includes(compactNeedle)) {
        return 75;
    }
    return 0;
}
