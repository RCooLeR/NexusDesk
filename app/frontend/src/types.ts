export type Capability = {
    title: string;
    description: string;
    status: string;
};

export type WorkspaceItem = {
    name: string;
    kind: string;
    meta: string;
};

export type FileNode = {
    name: string;
    path: string;
    relPath: string;
    kind: string;
    fileType: string;
    depth: number;
    meta: string;
};

export type WorkspaceSnapshot = {
    root: string;
    name: string;
    nodes: FileNode[];
    truncated: boolean;
};

export type WorkspaceOpenResult = {
    selected: boolean;
    snapshot: WorkspaceSnapshot;
};

export type FilePreview = {
    relPath: string;
    name: string;
    kind: string;
    fileType: string;
    content: string;
    truncated: boolean;
    message: string;
    size: number;
};

export type RecentWorkspace = {
    name: string;
    path: string;
    lastOpened: string;
};

export type LLMSettings = {
    providerName: string;
    baseUrl: string;
    model: string;
    apiKey: string;
    updatedAt: string;
};

export type LLMProbeResult = {
    ok: boolean;
    message: string;
    endpoint: string;
    modelCount: number;
    modelSample: string[];
    capabilities: string[];
    warnings: string[];
};

export type LLMChatResult = {
    message: string;
    model: string;
    endpoint: string;
    contextRelPath: string;
};

export type ToolEvent = {
    time: string;
    title: string;
    detail: string;
};

export type StartupState = {
    productName: string;
    tagline: string;
    buildStage: string;
    capabilities: Capability[];
    workspaceItems: WorkspaceItem[];
    toolEvents: ToolEvent[];
};
