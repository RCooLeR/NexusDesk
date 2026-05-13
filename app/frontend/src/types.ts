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
