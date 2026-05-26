import {useEffect, useState} from 'react';
import type {MouseEvent as ReactMouseEvent} from 'react';

const navigatorMinWidth = 220;
const navigatorMaxWidth = 460;
const agentMinWidth = 360;
const agentDefaultWidth = 480;
const bottomMinHeight = 150;
const bottomDefaultHeight = 300;
const panelStorageKey = 'nexus:resizable-panels';

export function useResizablePanels({railWidth}: {railWidth: number}) {
    const initial = readPanelSettings();
    const [navigatorWidth, setNavigatorWidth] = useState(initial.navigatorWidth);
    const [agentWidth, setAgentWidth] = useState(initial.agentWidth);
    const [bottomPanelHeight, setBottomPanelHeight] = useState(initial.bottomPanelHeight);

    useEffect(() => {
        writePanelSettings({agentWidth, bottomPanelHeight, navigatorWidth});
    }, [agentWidth, bottomPanelHeight, navigatorWidth]);

    function startNavigatorResize(event: ReactMouseEvent<HTMLDivElement>) {
        event.preventDefault();

        function resize(moveEvent: MouseEvent) {
            setNavigatorWidth(clamp(moveEvent.clientX - railWidth, navigatorMinWidth, navigatorMaxWidth));
        }

        beginResize('col-resize', resize);
    }

    function startAgentResize(event: ReactMouseEvent<HTMLDivElement>) {
        event.preventDefault();

        function resize(moveEvent: MouseEvent) {
            const maxWidth = Math.max(agentMinWidth, Math.floor(window.innerWidth * 0.5));
            setAgentWidth(clamp(window.innerWidth - moveEvent.clientX, agentMinWidth, maxWidth));
        }

        beginResize('col-resize', resize);
    }

    function startBottomResize(event: ReactMouseEvent<HTMLDivElement>) {
        event.preventDefault();

        function resize(moveEvent: MouseEvent) {
            const maxHeight = Math.max(bottomMinHeight, Math.floor(window.innerHeight * 0.7));
            setBottomPanelHeight(clamp(window.innerHeight - moveEvent.clientY, bottomMinHeight, maxHeight));
        }

        beginResize('row-resize', resize);
    }

    return {
        agentWidth,
        bottomPanelHeight,
        navigatorWidth,
        startAgentResize,
        startBottomResize,
        startNavigatorResize,
    };
}

function beginResize(cursor: string, resize: (event: MouseEvent) => void) {
    function stopResize() {
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
        window.removeEventListener('mousemove', resize);
        window.removeEventListener('mouseup', stopResize);
    }

    document.body.style.cursor = cursor;
    document.body.style.userSelect = 'none';
    window.addEventListener('mousemove', resize);
    window.addEventListener('mouseup', stopResize);
}

function clamp(value: number, min: number, max: number) {
    return Math.min(Math.max(value, min), max);
}

function readPanelSettings() {
    const fallback = {
        agentWidth: agentDefaultWidth,
        bottomPanelHeight: bottomDefaultHeight,
        navigatorWidth: 280,
    };
    try {
        const raw = window.localStorage.getItem(panelStorageKey);
        if (!raw) {
            return fallback;
        }
        const parsed = JSON.parse(raw) as Partial<typeof fallback>;
        return {
            agentWidth: clamp(Number(parsed.agentWidth) || fallback.agentWidth, agentMinWidth, Math.max(agentMinWidth, Math.floor(window.innerWidth * 0.5))),
            bottomPanelHeight: clamp(Number(parsed.bottomPanelHeight) || fallback.bottomPanelHeight, bottomMinHeight, Math.max(bottomMinHeight, Math.floor(window.innerHeight * 0.7))),
            navigatorWidth: clamp(Number(parsed.navigatorWidth) || fallback.navigatorWidth, navigatorMinWidth, navigatorMaxWidth),
        };
    } catch {
        return fallback;
    }
}

function writePanelSettings(settings: {agentWidth: number; bottomPanelHeight: number; navigatorWidth: number}) {
    try {
        window.localStorage.setItem(panelStorageKey, JSON.stringify(settings));
    } catch {
        // Ignore storage failures; panel sizing remains functional for this session.
    }
}
