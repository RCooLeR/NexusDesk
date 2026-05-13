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
