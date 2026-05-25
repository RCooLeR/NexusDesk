import symbolApp from '../assets/brand/logos/nexus-app-icon.png';
import logoHorizontalDark from '../assets/brand/logos/nexus-horizontal-dark.png';
import logoHorizontalWhite from '../assets/brand/logos/nexus-horizontal-white.png';
import logoVerticalDark from '../assets/brand/logos/nexus-vertical-dark.png';
import logoVerticalWhite from '../assets/brand/logos/nexus-vertical-white.png';
import type {IconDefinition} from '@fortawesome/fontawesome-svg-core';
import {
    faChartLine,
    faCode,
    faDatabase,
    faFileLines,
    faFolderTree,
    faGear,
    faRobot,
    faServer,
} from '@fortawesome/free-solid-svg-icons';

export const productBrand = {
    name: 'Nexus Augentic Studio',
    shortName: 'Nexus',
    tagline: 'Agentic work. Augmented by context.',
};

export const brandAssets = {
    symbolSilver: symbolApp,
    symbolDark: symbolApp,
    appIcon: symbolApp,
    logoHorizontalDark,
    logoHorizontalWhite,
    logoVerticalDark,
    logoVerticalWhite,
    icons: {
        ai: faRobot,
        analytics: faChartLine,
        code: faCode,
        data: faDatabase,
        documents: faFileLines,
        ops: faServer,
        settings: faGear,
        workspace: faFolderTree,
    },
};

export type StudioRouteId =
    | 'code'
    | 'data'
    | 'analytics'
    | 'documents'
    | 'assistant'
    | 'ops'
    | 'artifacts'
    | 'settings';

export const railItems: Array<{
    id: StudioRouteId;
    label: string;
    icon: IconDefinition;
}> = [
    {id: 'code', label: 'Workbench', icon: brandAssets.icons.code},
    {id: 'data', label: 'Data & Analytics', icon: brandAssets.icons.data},
    {id: 'artifacts', label: 'Artifacts', icon: brandAssets.icons.documents},
    {id: 'settings', label: 'Settings', icon: brandAssets.icons.settings},
];

export const studioRouteLabels: Record<StudioRouteId, string> = {
    code: 'Workbench',
    data: 'Data & Analytics',
    analytics: 'Analytics Capabilities',
    documents: 'Document Capabilities',
    assistant: 'AI Assistant',
    ops: 'Operations Capabilities',
    artifacts: 'Artifacts',
    settings: 'Settings',
};

export const studioRouteDescriptions: Record<StudioRouteId, string> = {
    code: 'Project tree, editor tabs, search, git, and safe edits.',
    data: 'Datasets, analytics imports, SQL, metadata, connectors, and data-derived artifacts.',
    analytics: 'Marketing analytics, API connectors, dashboards, and report workflows.',
    documents: 'PDF, DOCX, Markdown, document sets, summaries, and generated briefs.',
    assistant: 'Context packs, model control, agent runs, tool plans, and citations.',
    ops: 'Docker, Compose, services, logs, environments, and safe operations.',
    artifacts: 'Generated reports, charts, exports, lineage, comparison, and archive.',
    settings: 'Providers, model context, credentials, policies, diagnostics, and UI preferences.',
};

export const studioRouteSurfaceTab: Partial<Record<StudioRouteId, 'code' | 'settings' | 'data' | 'tools' | 'artifacts' | 'approvals' | 'activity'>> = {
    code: 'code',
    data: 'data',
    analytics: 'data',
    assistant: 'tools',
    ops: 'data',
    artifacts: 'artifacts',
    settings: 'settings',
};

export const pendingStudioRoutes: StudioRouteId[] = [
    'analytics',
    'documents',
    'ops',
];

export const studioRoutePrimarySurface: Record<StudioRouteId, string> = {
    code: 'Workbench editor, file tabs, previews, and safe edits',
    data: 'Unified data, analytics, SQL, and connector surface',
    analytics: 'Data & Analytics connector surface',
    documents: 'Workbench document preview and artifact surface',
    assistant: 'Assistant workspace and tool plan surface',
    ops: 'Operations inspector and tool surface',
    artifacts: 'Primary Artifact Studio surface',
    settings: 'Primary Settings surface',
};

export const studioRouteCommandHint: Record<StudioRouteId, string> = {
    code: 'Next: add staged diffs, hunk navigation, and AI commit support.',
    data: 'Next: promote datasets, connectors, notebooks, and dump imports into a full studio.',
    analytics: 'Next: add connector runs, dashboards, and marketing report workflows.',
    documents: 'Next: add document library, extraction jobs, comparison, and deck/report generation.',
    assistant: 'Next: promote long-running agent sessions and cross-studio context control.',
    ops: 'Next: add Docker inventory, logs, health, and approval-governed operations.',
    artifacts: 'Next: promote Artifact Studio out of the bottom drawer.',
    settings: 'Next: promote provider, credential, policy, and diagnostic settings into a route.',
};

export const implementedStudioRoutes: StudioRouteId[] = [
    'code',
    'data',
    'artifacts',
    'settings',
];

export const workspaceIconByName: Record<string, IconDefinition> = {
    app: brandAssets.icons.code,
    docs: brandAssets.icons.documents,
    services: brandAssets.icons.ops,
    workspace: brandAssets.icons.workspace,
};

export const capabilityIconByTitle: Record<string, IconDefinition> = {
    'Project IDE': brandAssets.icons.code,
    'Data & analytics studio': brandAssets.icons.data,
    'Artifact workflow': brandAssets.icons.documents,
};
