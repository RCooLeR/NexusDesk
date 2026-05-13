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
    text: string;
    pages?: TextPage[];
    encoding?: string;
    table?: TablePreview;
    truncated: boolean;
    message: string;
    size: number;
};

export type TextPage = {
    page: number;
    text: string;
};

export type FileWriteRequest = {
    relPath: string;
    content: string;
};

export type FileWriteProposal = {
    relPath: string;
    name: string;
    action: string;
    diff: string;
    size: number;
    message: string;
};

export type TablePreview = {
    columns: string[];
    rows: string[][];
    profiles: ColumnProfile[];
    totalRows: number;
    truncated: boolean;
};

export type ColumnProfile = {
    name: string;
    type: string;
    missing: number;
    distinct: number;
    min?: string;
    max?: string;
};

export type MarkdownReport = {
    relPath: string;
    name: string;
    path: string;
    message: string;
    size: number;
};

export type WorkspaceArtifact = {
    relPath: string;
    name: string;
    path: string;
    kind: string;
    size: number;
    modifiedAt: string;
};

export type DatasetProfile = {
    relPath: string;
    name: string;
    kind: string;
    rows: number;
    columns: number;
    sheets: string[];
    profiles: ColumnProfile[];
    updatedAt: string;
    message: string;
};

export type DatasetQueryResult = {
    relPath: string;
    query: string;
    columns: string[];
    rows: string[][];
    totalRows: number;
    matchedRows: number;
    message: string;
};

export type WorkspaceSearchResult = {
    relPath: string;
    name: string;
    kind: string;
    fileType: string;
    matchType: string;
    line: number;
    snippet: string;
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
    runtime?: LLMRuntimeStatus;
};

export type LLMRuntimeStatus = {
    provider: string;
    endpoint: string;
    message: string;
    selectedModel: string;
    selectedModelLoaded: boolean;
    selectedModelVram: number;
    loadedModels: LLMRuntimeModel[];
};

export type LLMRuntimeModel = {
    name: string;
    model: string;
    size: number;
    sizeVram: number;
    contextLength: number;
};

export type LLMChatResult = {
    message: string;
    model: string;
    endpoint: string;
    contextRelPath: string;
};

export type ChatStreamEvent = {
    requestId: string;
    type: 'delta' | 'done' | 'error';
    delta: string;
    message: string;
    model: string;
    endpoint: string;
    contextRelPath: string;
};

export type ChatMessage = {
    role: string;
    content: string;
    contextRelPath: string;
    createdAt: string;
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
