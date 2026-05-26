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
    encoding?: string;
};

export type FileWriteProposal = {
    relPath: string;
    name: string;
    action: string;
    diff: string;
    encoding: string;
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
    staleDatasets: string[];
    message: string;
};

export type GitFileChange = {
    path: string;
    oldPath: string;
    index: string;
    worktree: string;
    summary: string;
};

export type GitStatus = {
    available: boolean;
    repoRoot: string;
    branch: string;
    head: string;
    dirty: boolean;
    changedFiles: GitFileChange[];
    stagedFiles: GitFileChange[];
    unstagedFiles: GitFileChange[];
    diff: string;
    diffTruncated: boolean;
    stagedDiff: string;
    stagedDiffTruncated: boolean;
    unstagedDiff: string;
    unstagedDiffTruncated: boolean;
    aheadBehind: string;
    message: string;
    generatedAt: string;
};

export type GitFileDiff = {
    path: string;
    stagedDiff: string;
    stagedDiffTruncated: boolean;
    unstagedDiff: string;
    unstagedDiffTruncated: boolean;
    message: string;
    generatedAt: string;
};

export type GitFileAction = 'stage' | 'unstage';

export type GitFileActionPreview = {
    path: string;
    action: GitFileAction | string;
    command: string[];
    requiresApproval: boolean;
    mutatesRepository: boolean;
    message: string;
    status: GitStatus;
    generatedAt: string;
};

export type GitHunkAction = 'stage' | 'unstage' | 'discard' | 'revert';

export type GitHunkDiffKind = 'staged' | 'unstaged';

export type GitHunkActionRequest = {
    path: string;
    action: GitHunkAction | string;
    diffKind: GitHunkDiffKind | string;
    hunkIndex: number;
};

export type GitHunkActionPreview = {
    path: string;
    action: GitHunkAction | string;
    diffKind: GitHunkDiffKind | string;
    hunkIndex: number;
    command: string[];
    patch: string;
    requiresApproval: boolean;
    mutatesRepository: boolean;
    message: string;
    status: GitStatus;
    generatedAt: string;
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
    relationshipCounts: Record<string, number>;
    message: string;
};

export type MetadataColumn = {
    name: string;
    type: string;
};

export type MetadataTable = {
    name: string;
    rowCount: number;
    columns: MetadataColumn[];
    sampleRows: string[][];
};

export type DatasetView = {
    name: string;
    relPath: string;
    engine: string;
    columns: string[];
    rows: number;
    message: string;
};

export type MetadataBrowser = {
    path: string;
    tables: MetadataTable[];
    datasetViews: DatasetView[];
    message: string;
    updatedAt: string;
};

export type MetadataSearchResult = {
    id: string;
    kind: string;
    title: string;
    target: string;
    snippet: string;
    createdAt: string;
};

export type DatasetDependency = {
    id: string;
    relPath: string;
    kind: string;
    target: string;
    query: string;
    artifact: string;
    createdAt: string;
    lastRefresh: string;
};

export type SQLRun = {
    id: string;
    relPath: string;
    sql: string;
    engine: string;
    rows: number;
    artifact: string;
    status: string;
    message: string;
    createdAt: string;
};

export type SQLiteQueryResult = {
    relPath: string;
    sql: string;
    engine: string;
    columns: string[];
    rows: string[][];
    totalRows: number;
    truncated: boolean;
    resultLimit: number;
    timeoutSeconds: number;
    message: string;
};

export type SQLiteQueryRequest = {
    relPath: string;
    sql: string;
    requestId: string;
    resultLimit: number;
    timeoutSeconds: number;
};

export type ConnectorMetadata = {
    id: string;
    relPath: string;
    name: string;
    kind: string;
    engine: string;
    readOnly: boolean;
    tables: ConnectorTable[];
    views: ConnectorTable[];
    indexes: ConnectorIndex[];
    message: string;
};

export type ConnectorTable = {
    name: string;
    type: string;
    rowCount: number;
    columns: ConnectorColumn[];
    indexes: ConnectorIndex[];
    sampleRows: string[][];
};

export type ConnectorColumn = {
    name: string;
    type: string;
    nullable: boolean;
    primaryKey: boolean;
    default: string;
};

export type ConnectorIndex = {
    name: string;
    table: string;
    unique: boolean;
    columns: string[];
};

export type ConnectorProfile = {
    id: string;
    name: string;
    kind: string;
    driver: string;
    host: string;
    port: number;
    database: string;
    username: string;
    password: string;
    credentialRef: string;
    sslMode: string;
    workspaceScope: string;
    readOnly: boolean;
    resultLimit: number;
    timeoutSeconds: number;
    updatedAt: string;
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
    workbook: WorkbookInfo;
    parquet: ParquetInfo;
    log: LogInfo;
    profiles: ColumnProfile[];
    updatedAt: string;
    message: string;
};

export type WorkbookInfo = {
    sheets: WorkbookSheetInfo[];
    namedRanges: string[];
    tableRanges: WorkbookTableInfo[];
    pivotTables: string[];
    formulaCount: number;
};

export type WorkbookSheetInfo = {
    name: string;
    path: string;
    dimension: string;
    rows: number;
    columns: number;
    formulaCount: number;
    tableCount: number;
};

export type WorkbookTableInfo = {
    name: string;
    sheet: string;
    ref: string;
};

export type ParquetInfo = {
    fileSize: number;
    footerMetadataBytes: number;
    dataBytes: number;
    magic: string;
    message: string;
};

export type LogInfo = {
    fileSize: number;
    sampledBytes: number;
    sampledLines: number;
    totalLines: number;
    truncated: boolean;
    levelCounts: Record<string, number>;
    timestampedLines: number;
    stackTraceLines: number;
    topPatterns: LogPattern[];
    message: string;
};

export type LogPattern = {
    pattern: string;
    count: number;
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
    kind: string;
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

export type WorkspaceSearchRequest = {
    query: string;
    regex: boolean;
    symbols: boolean;
};

export type WorkspaceProblem = {
    relPath: string;
    name: string;
    severity: string;
    source: string;
    message: string;
    line: number;
};

export type WorkspaceProblemSummary = {
    problems: WorkspaceProblem[];
    message: string;
    generatedAt: string;
    truncated: boolean;
};

export type WorkspaceTask = {
    id: string;
    kind: string;
    label: string;
    command: string;
    cwd: string;
    source: string;
};

export type WorkspaceTaskSummary = {
    tasks: WorkspaceTask[];
    message: string;
    generatedAt: string;
};

export type WorkspaceTaskRunRequest = {
    taskId: string;
};

export type WorkspaceTaskRunResult = {
    task: WorkspaceTask;
    status: string;
    exitCode: number;
    stdout: string;
    stderr: string;
    startedAt: string;
    completedAt: string;
    durationMs: number;
    artifactRelPath: string;
    message: string;
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
    maxContextTokens: number;
    responseReserveTokens: number;
    updatedAt: string;
};

export type PromptProfile = {
    id: string;
    name: string;
    instructions: string;
};

export type AssistantProfile = {
    memory: string;
    activeProfileId: string;
    promptProfiles: PromptProfile[];
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

export type AgentRunEvent = {
    requestId: string;
    type: string;
    iteration: number;
    message: string;
    model: string;
    toolName: string;
    toolArgs?: Record<string, string>;
    observation: string;
    error: string;
    risk: string;
    plan?: Array<{step: string; status: string}>;
    timestamp: string;
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
