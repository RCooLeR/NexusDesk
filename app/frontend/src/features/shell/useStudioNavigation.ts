import {useEffect, useMemo, useState} from 'react';
import {studioRouteLabels, studioRouteSurfaceTab} from '../../brand/assets';
import type {StudioRouteId} from '../../brand/assets';
import type {BottomStudioTab} from './BottomStudioPanel';

const navigationStorageKey = 'nexus:studio-navigation';
const bottomTabs = new Set<BottomStudioTab>(['approvals', 'activity']);
const studioRoutes = new Set<StudioRouteId>(['code', 'assistant', 'data', 'analytics', 'documents', 'ops', 'artifacts', 'settings']);

export function useStudioNavigation(pushToolEvent: (title: string, detail: string) => void) {
    const initial = readNavigationSettings();
    const [activeBottomTab, setActiveBottomTab] = useState<BottomStudioTab>(initial.activeBottomTab);
    const [activeStudioRoute, setActiveStudioRoute] = useState<StudioRouteId>(initial.activeStudioRoute);
    const mainStudioTab = useMemo(() => mainStudioTabForRoute(activeStudioRoute), [activeStudioRoute]);

    useEffect(() => {
        writeNavigationSettings({activeBottomTab, activeStudioRoute});
    }, [activeBottomTab, activeStudioRoute]);

    function changeStudioRoute(route: StudioRouteId) {
        setActiveStudioRoute(route);
        pushToolEvent('Studio route selected', studioRouteLabels[route]);
    }

    function changeBottomStudioTab(tab: BottomStudioTab) {
        setActiveBottomTab(tab);
        pushToolEvent('Studio drawer tab selected', tab);
    }

    return {
        activeBottomTab,
        activeStudioRoute,
        changeBottomStudioTab,
        changeStudioRoute,
        mainStudioTab,
    };
}

function mainStudioTabForRoute(route: StudioRouteId): BottomStudioTab | null {
    if (route === 'code' || route === 'documents') {
        return null;
    }
    if (route === 'analytics' || route === 'ops') {
        return 'data';
    }
    return studioRouteSurfaceTab[route] ?? null;
}

function readNavigationSettings() {
    const fallback = {activeBottomTab: 'activity' as BottomStudioTab, activeStudioRoute: 'code' as StudioRouteId};
    try {
        const raw = window.localStorage.getItem(navigationStorageKey);
        if (!raw) {
            return fallback;
        }
        const parsed = JSON.parse(raw) as Partial<typeof fallback>;
        return {
            activeBottomTab: parsed.activeBottomTab && bottomTabs.has(parsed.activeBottomTab) ? parsed.activeBottomTab : fallback.activeBottomTab,
            activeStudioRoute: parsed.activeStudioRoute && studioRoutes.has(parsed.activeStudioRoute) ? parsed.activeStudioRoute : fallback.activeStudioRoute,
        };
    } catch {
        return fallback;
    }
}

function writeNavigationSettings(settings: {activeBottomTab: BottomStudioTab; activeStudioRoute: StudioRouteId}) {
    try {
        window.localStorage.setItem(navigationStorageKey, JSON.stringify(settings));
    } catch {
        // Navigation persistence is best-effort; route changes should never fail the shell.
    }
}
