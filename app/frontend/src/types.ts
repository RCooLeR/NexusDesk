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
    scan: ScanStatus;
};

export type ScanStatus = {
    included: number;
    ignored: number;
    depthSkipped: number;
    entrySkipped: number;
    unreadable: number;
    maxDepth: number;
    maxEntries: number;
    ignoredSamples: string[];
    skippedSamples: string[];
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
    source: string;
    summary: string;
    model: string;
};

export type ArtifactMetadata = {
    kind: string;
    title: string;
    source: string;
    sourcePaths: string[];
    contextRelPath: string;
    prompt: string;
    model: string;
    createdAt: string;
};

export type AgentToolDescriptor = {
    name: string;
    title: string;
    description: string;
    surface: string;
    risk: string;
    requiresApproval: boolean;
    inputs: string[];
};

export type AgentToolPlanItem = {
    toolName: string;
    title: string;
    target: string;
    risk: string;
    requiresApproval: boolean;
    status: string;
};

export type AgentToolRunRequest = {
    toolName: string;
    target: string;
    inputs: Record<string, string>;
    approved: boolean;
    approvalId: string;
};

export type AgentToolRunRecord = {
    id: string;
    toolName: string;
    title: string;
    target: string;
    risk: string;
    requiresApproval: boolean;
    status: string;
    mode: string;
    inputs: Record<string, string>;
    outputSummary: string;
    error: string;
    approvalId: string;
    startedAt: string;
    completedAt: string;
    durationMs: number;
};

export type SQLiteMetadataStatus = {
    path: string;
    schemaPath: string;
    schemaVersion: number;
    schemaHash: string;
    tables: string[];
    message: string;
    updatedAt: string;
};

export type WorkspaceFileChange = {
    relPath: string;
    kind: string;
    message: string;
};

export type WorkspaceFreshnessStatus = {
    changed: WorkspaceFileChange[];
    staleArtifacts: string[];
    message: string;
};

export type LineageNode = {
    id: string;
    kind: string;
    label: string;
    relPath: string;
};

export type LineageEdge = {
    from: string;
    to: string;
    label: string;
};

export type ArtifactLineage = {
    nodes: LineageNode[];
    edges: LineageEdge[];
    message: string;
};

export type ContextPreviewFile = {
    relPath: string;
    required: boolean;
};

export type ContextPreview = {
    roots: string[];
    files: ContextPreviewFile[];
    fileCount: number;
    truncated: boolean;
    message: string;
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

export type DatasetSQLQueryResult = {
    relPath: string;
    sql: string;
    engine: string;
    columns: string[];
    rows: string[][];
    totalRows: number;
    matchedRows: number;
    message: string;
};

export type DatasetSQLQueryRequest = {
    relPath: string;
    sql: string;
};

export type SavedDatasetQuery = {
    relPath: string;
    query: string;
    label: string;
    updatedAt: string;
};

export type DatasetChartRequest = {
    relPath: string;
    chartType: string;
    categoryColumn: string;
    valueColumn: string;
};

export type DatasetChartPoint = {
    label: string;
    value: number;
    count: number;
};

export type DatasetChartResult = {
    relPath: string;
    chartType: string;
    categoryColumn: string;
    valueColumn: string;
    mode: string;
    points: DatasetChartPoint[];
    totalRows: number;
    usedRows: number;
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
    sourcePaths: string[];
};

export type ChatStreamEvent = {
    requestId: string;
    type: 'delta' | 'done' | 'error';
    delta: string;
    message: string;
    model: string;
    endpoint: string;
    contextRelPath: string;
    sourcePaths: string[];
};

export type ChatMessage = {
    role: string;
    content: string;
    contextRelPath: string;
    sourcePaths?: string[];
    createdAt: string;
};

export type ToolEvent = {
    time: string;
    title: string;
    detail: string;
};

export type ApprovalRecord = {
    id: string;
    action: string;
    target: string;
    risk: string;
    decision: string;
    message: string;
    createdAt: string;
};

export type ArtifactComparison = {
    leftRelPath: string;
    rightRelPath: string;
    leftTitle: string;
    rightTitle: string;
    sameKind: boolean;
    sizeDelta: number;
    addedLines: string[];
    removedLines: string[];
    message: string;
};

export type StartupState = {
    productName: string;
    tagline: string;
    buildStage: string;
    capabilities: Capability[];
    workspaceItems: WorkspaceItem[];
    toolEvents: ToolEvent[];
};
