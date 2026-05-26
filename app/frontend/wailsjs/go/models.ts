export namespace agent {

	export class PlanStep {
	    step: string;
	    status: string;

	    static createFrom(source: any = {}) {
	        return new PlanStep(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.step = source["step"];
	        this.status = source["status"];
	    }
	}
	export class RunRequest {
	    requestId: string;
	    prompt: string;
	    maxIterations: number;
	    approveHighImpact: boolean;
	    allowShellCommands: boolean;

	    static createFrom(source: any = {}) {
	        return new RunRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requestId = source["requestId"];
	        this.prompt = source["prompt"];
	        this.maxIterations = source["maxIterations"];
	        this.approveHighImpact = source["approveHighImpact"];
	        this.allowShellCommands = source["allowShellCommands"];
	    }
	}
	export class ToolCall {
	    name: string;
	    arguments: Record<string, string>;
	    observation: string;
	    error: string;
	    risk: string;
	    startedAt: string;
	    completedAt: string;

	    static createFrom(source: any = {}) {
	        return new ToolCall(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.arguments = source["arguments"];
	        this.observation = source["observation"];
	        this.error = source["error"];
	        this.risk = source["risk"];
	        this.startedAt = source["startedAt"];
	        this.completedAt = source["completedAt"];
	    }
	}
	export class RunResult {
	    message: string;
	    plan: PlanStep[];
	    toolCalls: ToolCall[];
	    iterations: number;
	    truncated: boolean;
	    stopReason?: string;

	    static createFrom(source: any = {}) {
	        return new RunResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.plan = this.convertValues(source["plan"], PlanStep);
	        this.toolCalls = this.convertValues(source["toolCalls"], ToolCall);
	        this.iterations = source["iterations"];
	        this.truncated = source["truncated"];
	        this.stopReason = source["stopReason"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace agenttools {

	export class Descriptor {
	    name: string;
	    title: string;
	    description: string;
	    surface: string;
	    risk: string;
	    requiresApproval: boolean;
	    inputs: string[];

	    static createFrom(source: any = {}) {
	        return new Descriptor(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.surface = source["surface"];
	        this.risk = source["risk"];
	        this.requiresApproval = source["requiresApproval"];
	        this.inputs = source["inputs"];
	    }
	}
	export class RunRecord {
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

	    static createFrom(source: any = {}) {
	        return new RunRecord(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.toolName = source["toolName"];
	        this.title = source["title"];
	        this.target = source["target"];
	        this.risk = source["risk"];
	        this.requiresApproval = source["requiresApproval"];
	        this.status = source["status"];
	        this.mode = source["mode"];
	        this.inputs = source["inputs"];
	        this.outputSummary = source["outputSummary"];
	        this.error = source["error"];
	        this.approvalId = source["approvalId"];
	        this.startedAt = source["startedAt"];
	        this.completedAt = source["completedAt"];
	        this.durationMs = source["durationMs"];
	    }
	}
	export class RunRequest {
	    toolName: string;
	    target: string;
	    inputs: Record<string, string>;
	    approved: boolean;
	    approvalId: string;

	    static createFrom(source: any = {}) {
	        return new RunRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.toolName = source["toolName"];
	        this.target = source["target"];
	        this.inputs = source["inputs"];
	        this.approved = source["approved"];
	        this.approvalId = source["approvalId"];
	    }
	}

}

export namespace analytics {

	export class SQLQueryRequest {
	    relPath: string;
	    sql: string;

	    static createFrom(source: any = {}) {
	        return new SQLQueryRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relPath = source["relPath"];
	        this.sql = source["sql"];
	    }
	}
	export class SQLQueryResult {
	    relPath: string;
	    sql: string;
	    engine: string;
	    columns: string[];
	    rows: string[][];
	    totalRows: number;
	    matchedRows: number;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new SQLQueryResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relPath = source["relPath"];
	        this.sql = source["sql"];
	        this.engine = source["engine"];
	        this.columns = source["columns"];
	        this.rows = source["rows"];
	        this.totalRows = source["totalRows"];
	        this.matchedRows = source["matchedRows"];
	        this.message = source["message"];
	    }
	}

}

export namespace appmeta {

	export class DatasetDependency {
	    id: string;
	    relPath: string;
	    kind: string;
	    target: string;
	    query: string;
	    artifact: string;
	    createdAt: string;
	    lastRefresh: string;

	    static createFrom(source: any = {}) {
	        return new DatasetDependency(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.relPath = source["relPath"];
	        this.kind = source["kind"];
	        this.target = source["target"];
	        this.query = source["query"];
	        this.artifact = source["artifact"];
	        this.createdAt = source["createdAt"];
	        this.lastRefresh = source["lastRefresh"];
	    }
	}
	export class DatasetView {
	    name: string;
	    relPath: string;
	    engine: string;
	    columns: string[];
	    rows: number;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new DatasetView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.relPath = source["relPath"];
	        this.engine = source["engine"];
	        this.columns = source["columns"];
	        this.rows = source["rows"];
	        this.message = source["message"];
	    }
	}
	export class MetadataColumn {
	    name: string;
	    type: string;

	    static createFrom(source: any = {}) {
	        return new MetadataColumn(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	    }
	}
	export class MetadataTable {
	    name: string;
	    rowCount: number;
	    columns: MetadataColumn[];
	    sampleRows: string[][];

	    static createFrom(source: any = {}) {
	        return new MetadataTable(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.rowCount = source["rowCount"];
	        this.columns = this.convertValues(source["columns"], MetadataColumn);
	        this.sampleRows = source["sampleRows"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MetadataBrowser {
	    path: string;
	    tables: MetadataTable[];
	    datasetViews: DatasetView[];
	    message: string;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new MetadataBrowser(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.tables = this.convertValues(source["tables"], MetadataTable);
	        this.datasetViews = this.convertValues(source["datasetViews"], DatasetView);
	        this.message = source["message"];
	        this.updatedAt = source["updatedAt"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class MetadataSearchResult {
	    id: string;
	    kind: string;
	    title: string;
	    target: string;
	    snippet: string;
	    createdAt: string;

	    static createFrom(source: any = {}) {
	        return new MetadataSearchResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.title = source["title"];
	        this.target = source["target"];
	        this.snippet = source["snippet"];
	        this.createdAt = source["createdAt"];
	    }
	}

	export class SQLRun {
	    id: string;
	    relPath: string;
	    sql: string;
	    engine: string;
	    rows: number;
	    artifact: string;
	    status: string;
	    message: string;
	    createdAt: string;

	    static createFrom(source: any = {}) {
	        return new SQLRun(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.relPath = source["relPath"];
	        this.sql = source["sql"];
	        this.engine = source["engine"];
	        this.rows = source["rows"];
	        this.artifact = source["artifact"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class SQLiteStatus {
	    path: string;
	    schemaPath: string;
	    schemaVersion: number;
	    schemaHash: string;
	    tables: string[];
	    message: string;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new SQLiteStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.schemaPath = source["schemaPath"];
	        this.schemaVersion = source["schemaVersion"];
	        this.schemaHash = source["schemaHash"];
	        this.tables = source["tables"];
	        this.message = source["message"];
	        this.updatedAt = source["updatedAt"];
	    }
	}

}

export namespace approval {

	export class Record {
	    id: string;
	    action: string;
	    target: string;
	    risk: string;
	    decision: string;
	    message: string;
	    createdAt: string;

	    static createFrom(source: any = {}) {
	        return new Record(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.action = source["action"];
	        this.target = source["target"];
	        this.risk = source["risk"];
	        this.decision = source["decision"];
	        this.message = source["message"];
	        this.createdAt = source["createdAt"];
	    }
	}

}

export namespace artifact {

	export class ArtifactComparison {
	    leftRelPath: string;
	    rightRelPath: string;
	    leftTitle: string;
	    rightTitle: string;
	    sameKind: boolean;
	    sizeDelta: number;
	    addedLines: string[];
	    removedLines: string[];
	    message: string;

	    static createFrom(source: any = {}) {
	        return new ArtifactComparison(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.leftRelPath = source["leftRelPath"];
	        this.rightRelPath = source["rightRelPath"];
	        this.leftTitle = source["leftTitle"];
	        this.rightTitle = source["rightTitle"];
	        this.sameKind = source["sameKind"];
	        this.sizeDelta = source["sizeDelta"];
	        this.addedLines = source["addedLines"];
	        this.removedLines = source["removedLines"];
	        this.message = source["message"];
	    }
	}
	export class ArtifactMetadata {
	    kind: string;
	    title: string;
	    source: string;
	    sourcePaths: string[];
	    contextRelPath: string;
	    prompt: string;
	    model: string;
	    createdAt: string;

	    static createFrom(source: any = {}) {
	        return new ArtifactMetadata(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.title = source["title"];
	        this.source = source["source"];
	        this.sourcePaths = source["sourcePaths"];
	        this.contextRelPath = source["contextRelPath"];
	        this.prompt = source["prompt"];
	        this.model = source["model"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class MarkdownArtifactRequest {
	    title: string;
	    content: string;
	    kind: string;
	    contextRelPath: string;
	    prompt: string;
	    model: string;
	    source: string;
	    sourcePaths: string[];

	    static createFrom(source: any = {}) {
	        return new MarkdownArtifactRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.content = source["content"];
	        this.kind = source["kind"];
	        this.contextRelPath = source["contextRelPath"];
	        this.prompt = source["prompt"];
	        this.model = source["model"];
	        this.source = source["source"];
	        this.sourcePaths = source["sourcePaths"];
	    }
	}
	export class MarkdownReport {
	    relPath: string;
	    name: string;
	    path: string;
	    message: string;
	    size: number;

	    static createFrom(source: any = {}) {
	        return new MarkdownReport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relPath = source["relPath"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.message = source["message"];
	        this.size = source["size"];
	    }
	}
	export class WorkspaceArtifact {
	    relPath: string;
	    name: string;
	    path: string;
	    kind: string;
	    size: number;
	    modifiedAt: string;
	    source: string;
	    summary: string;
	    model: string;

	    static createFrom(source: any = {}) {
	        return new WorkspaceArtifact(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relPath = source["relPath"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.kind = source["kind"];
	        this.size = source["size"];
	        this.modifiedAt = source["modifiedAt"];
	        this.source = source["source"];
	        this.summary = source["summary"];
	        this.model = source["model"];
	    }
	}

}

export namespace dataset {

	export class LogPattern {
	    pattern: string;
	    count: number;

	    static createFrom(source: any = {}) {
	        return new LogPattern(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pattern = source["pattern"];
	        this.count = source["count"];
	    }
	}
	export class LogInfo {
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

	    static createFrom(source: any = {}) {
	        return new LogInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileSize = source["fileSize"];
	        this.sampledBytes = source["sampledBytes"];
	        this.sampledLines = source["sampledLines"];
	        this.totalLines = source["totalLines"];
	        this.truncated = source["truncated"];
	        this.levelCounts = source["levelCounts"];
	        this.timestampedLines = source["timestampedLines"];
	        this.stackTraceLines = source["stackTraceLines"];
	        this.topPatterns = this.convertValues(source["topPatterns"], LogPattern);
	        this.message = source["message"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class ParquetInfo {
	    fileSize: number;
	    footerMetadataBytes: number;
	    dataBytes: number;
	    magic: string;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new ParquetInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileSize = source["fileSize"];
	        this.footerMetadataBytes = source["footerMetadataBytes"];
	        this.dataBytes = source["dataBytes"];
	        this.magic = source["magic"];
	        this.message = source["message"];
	    }
	}
	export class WorkbookTableInfo {
	    name: string;
	    sheet: string;
	    ref: string;

	    static createFrom(source: any = {}) {
	        return new WorkbookTableInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.sheet = source["sheet"];
	        this.ref = source["ref"];
	    }
	}
	export class WorkbookSheetInfo {
	    name: string;
	    path: string;
	    dimension: string;
	    rows: number;
	    columns: number;
	    formulaCount: number;
	    tableCount: number;

	    static createFrom(source: any = {}) {
	        return new WorkbookSheetInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.dimension = source["dimension"];
	        this.rows = source["rows"];
	        this.columns = source["columns"];
	        this.formulaCount = source["formulaCount"];
	        this.tableCount = source["tableCount"];
	    }
	}
	export class WorkbookInfo {
	    sheets: WorkbookSheetInfo[];
	    namedRanges: string[];
	    tableRanges: WorkbookTableInfo[];
	    pivotTables: string[];
	    formulaCount: number;

	    static createFrom(source: any = {}) {
	        return new WorkbookInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sheets = this.convertValues(source["sheets"], WorkbookSheetInfo);
	        this.namedRanges = source["namedRanges"];
	        this.tableRanges = this.convertValues(source["tableRanges"], WorkbookTableInfo);
	        this.pivotTables = source["pivotTables"];
	        this.formulaCount = source["formulaCount"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Profile {
	    relPath: string;
	    name: string;
	    kind: string;
	    rows: number;
	    columns: number;
	    sheets: string[];
	    workbook: WorkbookInfo;
	    parquet: ParquetInfo;
	    log: LogInfo;
	    profiles: workspace.ColumnProfile[];
	    updatedAt: string;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relPath = source["relPath"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.rows = source["rows"];
	        this.columns = source["columns"];
	        this.sheets = source["sheets"];
	        this.workbook = this.convertValues(source["workbook"], WorkbookInfo);
	        this.parquet = this.convertValues(source["parquet"], ParquetInfo);
	        this.log = this.convertValues(source["log"], LogInfo);
	        this.profiles = this.convertValues(source["profiles"], workspace.ColumnProfile);
	        this.updatedAt = source["updatedAt"];
	        this.message = source["message"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SavedQuery {
	    relPath: string;
	    query: string;
	    label: string;
	    kind: string;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new SavedQuery(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relPath = source["relPath"];
	        this.query = source["query"];
	        this.label = source["label"];
	        this.kind = source["kind"];
	        this.updatedAt = source["updatedAt"];
	    }
	}



}

export namespace dbconnector {

	export class ConnectorColumn {
	    name: string;
	    type: string;
	    nullable: boolean;
	    primaryKey: boolean;
	    default: string;

	    static createFrom(source: any = {}) {
	        return new ConnectorColumn(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.nullable = source["nullable"];
	        this.primaryKey = source["primaryKey"];
	        this.default = source["default"];
	    }
	}
	export class ConnectorIndex {
	    name: string;
	    table: string;
	    unique: boolean;
	    columns: string[];

	    static createFrom(source: any = {}) {
	        return new ConnectorIndex(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.table = source["table"];
	        this.unique = source["unique"];
	        this.columns = source["columns"];
	    }
	}
	export class ConnectorRelationship {
	    kind: string;
	    fromTable: string;
	    fromColumn: string;
	    toTable: string;
	    toColumn: string;
	    confidence: string;
	    reason: string;

	    static createFrom(source: any = {}) {
	        return new ConnectorRelationship(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.fromTable = source["fromTable"];
	        this.fromColumn = source["fromColumn"];
	        this.toTable = source["toTable"];
	        this.toColumn = source["toColumn"];
	        this.confidence = source["confidence"];
	        this.reason = source["reason"];
	    }
	}
	export class ConnectorTable {
	    name: string;
	    type: string;
	    rowCount: number;
	    columns: ConnectorColumn[];
	    indexes: ConnectorIndex[];
	    sampleRows: string[][];

	    static createFrom(source: any = {}) {
	        return new ConnectorTable(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.rowCount = source["rowCount"];
	        this.columns = this.convertValues(source["columns"], ConnectorColumn);
	        this.indexes = this.convertValues(source["indexes"], ConnectorIndex);
	        this.sampleRows = source["sampleRows"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ConnectorMetadata {
	    id: string;
	    relPath: string;
	    name: string;
	    kind: string;
	    engine: string;
	    readOnly: boolean;
	    tables: ConnectorTable[];
	    views: ConnectorTable[];
	    indexes: ConnectorIndex[];
	    relationships: ConnectorRelationship[];
	    message: string;

	    static createFrom(source: any = {}) {
	        return new ConnectorMetadata(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.relPath = source["relPath"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.engine = source["engine"];
	        this.readOnly = source["readOnly"];
	        this.tables = this.convertValues(source["tables"], ConnectorTable);
	        this.views = this.convertValues(source["views"], ConnectorTable);
	        this.indexes = this.convertValues(source["indexes"], ConnectorIndex);
	        this.relationships = this.convertValues(source["relationships"], ConnectorRelationship);
	        this.message = source["message"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ConnectorProfileStatus {
	    profileId: string;
	    name: string;
	    kind: string;
	    engine: string;
	    readOnly: boolean;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new ConnectorProfileStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.engine = source["engine"];
	        this.readOnly = source["readOnly"];
	        this.message = source["message"];
	    }
	}
	export class ConnectorQueryRequest {
	    profileId: string;
	    sql: string;
	    requestId: string;
	    resultLimit: number;
	    timeoutSeconds: number;

	    static createFrom(source: any = {}) {
	        return new ConnectorQueryRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	        this.sql = source["sql"];
	        this.requestId = source["requestId"];
	        this.resultLimit = source["resultLimit"];
	        this.timeoutSeconds = source["timeoutSeconds"];
	    }
	}
	export class ConnectorQueryResult {
	    profileId: string;
	    name: string;
	    kind: string;
	    engine: string;
	    sql: string;
	    columns: string[];
	    rows: string[][];
	    totalRows: number;
	    truncated: boolean;
	    resultLimit: number;
	    timeoutSeconds: number;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new ConnectorQueryResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.engine = source["engine"];
	        this.sql = source["sql"];
	        this.columns = source["columns"];
	        this.rows = source["rows"];
	        this.totalRows = source["totalRows"];
	        this.truncated = source["truncated"];
	        this.resultLimit = source["resultLimit"];
	        this.timeoutSeconds = source["timeoutSeconds"];
	        this.message = source["message"];
	    }
	}


	export class SQLiteQueryRequest {
	    relPath: string;
	    sql: string;
	    requestId: string;
	    resultLimit: number;
	    timeoutSeconds: number;

	    static createFrom(source: any = {}) {
	        return new SQLiteQueryRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relPath = source["relPath"];
	        this.sql = source["sql"];
	        this.requestId = source["requestId"];
	        this.resultLimit = source["resultLimit"];
	        this.timeoutSeconds = source["timeoutSeconds"];
	    }
	}
	export class SQLiteQueryResult {
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

	    static createFrom(source: any = {}) {
	        return new SQLiteQueryResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relPath = source["relPath"];
	        this.sql = source["sql"];
	        this.engine = source["engine"];
	        this.columns = source["columns"];
	        this.rows = source["rows"];
	        this.totalRows = source["totalRows"];
	        this.truncated = source["truncated"];
	        this.resultLimit = source["resultLimit"];
	        this.timeoutSeconds = source["timeoutSeconds"];
	        this.message = source["message"];
	    }
	}

}

export namespace llm {

	export class ChatResult {
	    message: string;
	    model: string;
	    endpoint: string;
	    contextRelPath: string;
	    sourcePaths: string[];

	    static createFrom(source: any = {}) {
	        return new ChatResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.model = source["model"];
	        this.endpoint = source["endpoint"];
	        this.contextRelPath = source["contextRelPath"];
	        this.sourcePaths = source["sourcePaths"];
	    }
	}
	export class RuntimeModel {
	    name: string;
	    model: string;
	    size: number;
	    sizeVram: number;
	    contextLength: number;

	    static createFrom(source: any = {}) {
	        return new RuntimeModel(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.model = source["model"];
	        this.size = source["size"];
	        this.sizeVram = source["sizeVram"];
	        this.contextLength = source["contextLength"];
	    }
	}
	export class RuntimeStatus {
	    provider: string;
	    endpoint: string;
	    message: string;
	    selectedModel: string;
	    selectedModelLoaded: boolean;
	    selectedModelVram: number;
	    loadedModels: RuntimeModel[];

	    static createFrom(source: any = {}) {
	        return new RuntimeStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.endpoint = source["endpoint"];
	        this.message = source["message"];
	        this.selectedModel = source["selectedModel"];
	        this.selectedModelLoaded = source["selectedModelLoaded"];
	        this.selectedModelVram = source["selectedModelVram"];
	        this.loadedModels = this.convertValues(source["loadedModels"], RuntimeModel);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProbeResult {
	    ok: boolean;
	    message: string;
	    endpoint: string;
	    modelCount: number;
	    modelSample: string[];
	    capabilities: string[];
	    warnings: string[];
	    runtime?: RuntimeStatus;

	    static createFrom(source: any = {}) {
	        return new ProbeResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	        this.endpoint = source["endpoint"];
	        this.modelCount = source["modelCount"];
	        this.modelSample = source["modelSample"];
	        this.capabilities = source["capabilities"];
	        this.warnings = source["warnings"];
	        this.runtime = this.convertValues(source["runtime"], RuntimeStatus);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}


}

export namespace main {

	export class LineageEdge {
	    from: string;
	    to: string;
	    label: string;

	    static createFrom(source: any = {}) {
	        return new LineageEdge(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	        this.label = source["label"];
	    }
	}
	export class LineageNode {
	    id: string;
	    kind: string;
	    label: string;
	    relPath: string;

	    static createFrom(source: any = {}) {
	        return new LineageNode(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.relPath = source["relPath"];
	    }
	}
	export class ArtifactLineage {
	    nodes: LineageNode[];
	    edges: LineageEdge[];
	    relationshipCounts: Record<string, number>;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new ArtifactLineage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodes = this.convertValues(source["nodes"], LineageNode);
	        this.edges = this.convertValues(source["edges"], LineageEdge);
	        this.relationshipCounts = source["relationshipCounts"];
	        this.message = source["message"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ArtifactLineageImport {
	    lineage: ArtifactLineage;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new ArtifactLineageImport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lineage = this.convertValues(source["lineage"], ArtifactLineage);
	        this.message = source["message"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Capability {
	    title: string;
	    description: string;
	    status: string;

	    static createFrom(source: any = {}) {
	        return new Capability(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.description = source["description"];
	        this.status = source["status"];
	    }
	}
	export class GitFileChange {
	    path: string;
	    oldPath: string;
	    index: string;
	    worktree: string;
	    summary: string;

	    static createFrom(source: any = {}) {
	        return new GitFileChange(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.oldPath = source["oldPath"];
	        this.index = source["index"];
	        this.worktree = source["worktree"];
	        this.summary = source["summary"];
	    }
	}
	export class GitStatus {
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

	    static createFrom(source: any = {}) {
	        return new GitStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.repoRoot = source["repoRoot"];
	        this.branch = source["branch"];
	        this.head = source["head"];
	        this.dirty = source["dirty"];
	        this.changedFiles = this.convertValues(source["changedFiles"], GitFileChange);
	        this.stagedFiles = this.convertValues(source["stagedFiles"], GitFileChange);
	        this.unstagedFiles = this.convertValues(source["unstagedFiles"], GitFileChange);
	        this.diff = source["diff"];
	        this.diffTruncated = source["diffTruncated"];
	        this.stagedDiff = source["stagedDiff"];
	        this.stagedDiffTruncated = source["stagedDiffTruncated"];
	        this.unstagedDiff = source["unstagedDiff"];
	        this.unstagedDiffTruncated = source["unstagedDiffTruncated"];
	        this.aheadBehind = source["aheadBehind"];
	        this.message = source["message"];
	        this.generatedAt = source["generatedAt"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GitFileActionPreview {
	    path: string;
	    action: string;
	    command: string[];
	    requiresApproval: boolean;
	    mutatesRepository: boolean;
	    message: string;
	    status: GitStatus;
	    generatedAt: string;

	    static createFrom(source: any = {}) {
	        return new GitFileActionPreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.action = source["action"];
	        this.command = source["command"];
	        this.requiresApproval = source["requiresApproval"];
	        this.mutatesRepository = source["mutatesRepository"];
	        this.message = source["message"];
	        this.status = this.convertValues(source["status"], GitStatus);
	        this.generatedAt = source["generatedAt"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GitFileActionRequest {
	    path: string;
	    action: string;

	    static createFrom(source: any = {}) {
	        return new GitFileActionRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.action = source["action"];
	    }
	}

	export class GitFileDiff {
	    path: string;
	    stagedDiff: string;
	    stagedDiffTruncated: boolean;
	    unstagedDiff: string;
	    unstagedDiffTruncated: boolean;
	    message: string;
	    generatedAt: string;

	    static createFrom(source: any = {}) {
	        return new GitFileDiff(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.stagedDiff = source["stagedDiff"];
	        this.stagedDiffTruncated = source["stagedDiffTruncated"];
	        this.unstagedDiff = source["unstagedDiff"];
	        this.unstagedDiffTruncated = source["unstagedDiffTruncated"];
	        this.message = source["message"];
	        this.generatedAt = source["generatedAt"];
	    }
	}
	export class GitHunkActionPreview {
	    path: string;
	    action: string;
	    diffKind: string;
	    hunkIndex: number;
	    command: string[];
	    patch: string;
	    requiresApproval: boolean;
	    mutatesRepository: boolean;
	    message: string;
	    status: GitStatus;
	    generatedAt: string;

	    static createFrom(source: any = {}) {
	        return new GitHunkActionPreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.action = source["action"];
	        this.diffKind = source["diffKind"];
	        this.hunkIndex = source["hunkIndex"];
	        this.command = source["command"];
	        this.patch = source["patch"];
	        this.requiresApproval = source["requiresApproval"];
	        this.mutatesRepository = source["mutatesRepository"];
	        this.message = source["message"];
	        this.status = this.convertValues(source["status"], GitStatus);
	        this.generatedAt = source["generatedAt"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GitHunkActionRequest {
	    path: string;
	    action: string;
	    diffKind: string;
	    hunkIndex: number;

	    static createFrom(source: any = {}) {
	        return new GitHunkActionRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.action = source["action"];
	        this.diffKind = source["diffKind"];
	        this.hunkIndex = source["hunkIndex"];
	    }
	}



	export class StaleContextRefresh {
	    preview: workspace.ContextPreview;
	    affectedChats: number;
	    staleArtifacts: string[];
	    staleDatasets: string[];
	    message: string;

	    static createFrom(source: any = {}) {
	        return new StaleContextRefresh(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.preview = this.convertValues(source["preview"], workspace.ContextPreview);
	        this.affectedChats = source["affectedChats"];
	        this.staleArtifacts = source["staleArtifacts"];
	        this.staleDatasets = source["staleDatasets"];
	        this.message = source["message"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ToolEvent {
	    time: string;
	    title: string;
	    detail: string;

	    static createFrom(source: any = {}) {
	        return new ToolEvent(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.title = source["title"];
	        this.detail = source["detail"];
	    }
	}
	export class WorkspaceItem {
	    name: string;
	    kind: string;
	    meta: string;

	    static createFrom(source: any = {}) {
	        return new WorkspaceItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.meta = source["meta"];
	    }
	}
	export class StartupState {
	    productName: string;
	    tagline: string;
	    buildStage: string;
	    capabilities: Capability[];
	    workspaceItems: WorkspaceItem[];
	    toolEvents: ToolEvent[];

	    static createFrom(source: any = {}) {
	        return new StartupState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.productName = source["productName"];
	        this.tagline = source["tagline"];
	        this.buildStage = source["buildStage"];
	        this.capabilities = this.convertValues(source["capabilities"], Capability);
	        this.workspaceItems = this.convertValues(source["workspaceItems"], WorkspaceItem);
	        this.toolEvents = this.convertValues(source["toolEvents"], ToolEvent);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}


	export class WorkspaceOpenResult {
	    selected: boolean;
	    snapshot: workspace.WorkspaceSnapshot;

	    static createFrom(source: any = {}) {
	        return new WorkspaceOpenResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.selected = source["selected"];
	        this.snapshot = this.convertValues(source["snapshot"], workspace.WorkspaceSnapshot);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspaceSearchRequest {
	    query: string;
	    regex: boolean;
	    symbols: boolean;

	    static createFrom(source: any = {}) {
	        return new WorkspaceSearchRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.regex = source["regex"];
	        this.symbols = source["symbols"];
	    }
	}
	export class WorkspaceTask {
	    id: string;
	    kind: string;
	    label: string;
	    command: string;
	    cwd: string;
	    source: string;

	    static createFrom(source: any = {}) {
	        return new WorkspaceTask(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.command = source["command"];
	        this.cwd = source["cwd"];
	        this.source = source["source"];
	    }
	}
	export class WorkspaceTaskRunRequest {
	    taskId: string;

	    static createFrom(source: any = {}) {
	        return new WorkspaceTaskRunRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	    }
	}
	export class WorkspaceTaskRunResult {
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

	    static createFrom(source: any = {}) {
	        return new WorkspaceTaskRunResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.task = this.convertValues(source["task"], WorkspaceTask);
	        this.status = source["status"];
	        this.exitCode = source["exitCode"];
	        this.stdout = source["stdout"];
	        this.stderr = source["stderr"];
	        this.startedAt = source["startedAt"];
	        this.completedAt = source["completedAt"];
	        this.durationMs = source["durationMs"];
	        this.artifactRelPath = source["artifactRelPath"];
	        this.message = source["message"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspaceTaskSummary {
	    tasks: WorkspaceTask[];
	    message: string;
	    generatedAt: string;

	    static createFrom(source: any = {}) {
	        return new WorkspaceTaskSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tasks = this.convertValues(source["tasks"], WorkspaceTask);
	        this.message = source["message"];
	        this.generatedAt = source["generatedAt"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace storage {

	export class PromptProfile {
	    id: string;
	    name: string;
	    instructions: string;

	    static createFrom(source: any = {}) {
	        return new PromptProfile(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.instructions = source["instructions"];
	    }
	}
	export class AssistantProfile {
	    memory: string;
	    activeProfileId: string;
	    promptProfiles: PromptProfile[];
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new AssistantProfile(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.memory = source["memory"];
	        this.activeProfileId = source["activeProfileId"];
	        this.promptProfiles = this.convertValues(source["promptProfiles"], PromptProfile);
	        this.updatedAt = source["updatedAt"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ChatMessage {
	    role: string;
	    content: string;
	    contextRelPath: string;
	    sourcePaths: string[];
	    createdAt: string;

	    static createFrom(source: any = {}) {
	        return new ChatMessage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	        this.contextRelPath = source["contextRelPath"];
	        this.sourcePaths = source["sourcePaths"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class ConnectorProfile {
	    id: string;
	    name: string;
	    kind: string;
	    driver: string;
	    host: string;
	    port: number;
	    database: string;
	    username: string;
	    password?: string;
	    credentialRef: string;
	    sslMode: string;
	    workspaceScope: string;
	    readOnly: boolean;
	    resultLimit: number;
	    timeoutSeconds: number;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new ConnectorProfile(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.driver = source["driver"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.database = source["database"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.credentialRef = source["credentialRef"];
	        this.sslMode = source["sslMode"];
	        this.workspaceScope = source["workspaceScope"];
	        this.readOnly = source["readOnly"];
	        this.resultLimit = source["resultLimit"];
	        this.timeoutSeconds = source["timeoutSeconds"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class LLMSettings {
	    providerName: string;
	    baseUrl: string;
	    model: string;
	    apiKey: string;
	    maxContextTokens: number;
	    responseReserveTokens: number;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new LLMSettings(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providerName = source["providerName"];
	        this.baseUrl = source["baseUrl"];
	        this.model = source["model"];
	        this.apiKey = source["apiKey"];
	        this.maxContextTokens = source["maxContextTokens"];
	        this.responseReserveTokens = source["responseReserveTokens"];
	        this.updatedAt = source["updatedAt"];
	    }
	}

	export class RecentWorkspace {
	    name: string;
	    path: string;
	    lastOpened: string;

	    static createFrom(source: any = {}) {
	        return new RecentWorkspace(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.lastOpened = source["lastOpened"];
	    }
	}

}

export namespace workspace {

	export class ColumnProfile {
	    name: string;
	    type: string;
	    missing: number;
	    distinct: number;
	    min?: string;
	    max?: string;

	    static createFrom(source: any = {}) {
	        return new ColumnProfile(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.missing = source["missing"];
	        this.distinct = source["distinct"];
	        this.min = source["min"];
	        this.max = source["max"];
	    }
	}
	export class ContextPreviewFile {
	    relPath: string;
	    required: boolean;

	    static createFrom(source: any = {}) {
	        return new ContextPreviewFile(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relPath = source["relPath"];
	        this.required = source["required"];
	    }
	}
	export class ContextPreview {
	    roots: string[];
	    files: ContextPreviewFile[];
	    fileCount: number;
	    truncated: boolean;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new ContextPreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.roots = source["roots"];
	        this.files = this.convertValues(source["files"], ContextPreviewFile);
	        this.fileCount = source["fileCount"];
	        this.truncated = source["truncated"];
	        this.message = source["message"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class DatasetChartPoint {
	    label: string;
	    value: number;
	    count: number;

	    static createFrom(source: any = {}) {
	        return new DatasetChartPoint(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.value = source["value"];
	        this.count = source["count"];
	    }
	}
	export class DatasetChartRequest {
	    relPath: string;
	    chartType: string;
	    categoryColumn: string;
	    valueColumn: string;

	    static createFrom(source: any = {}) {
	        return new DatasetChartRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relPath = source["relPath"];
	        this.chartType = source["chartType"];
	        this.categoryColumn = source["categoryColumn"];
	        this.valueColumn = source["valueColumn"];
	    }
	}
	export class DatasetChartResult {
	    relPath: string;
	    chartType: string;
	    categoryColumn: string;
	    valueColumn: string;
	    mode: string;
	    points: DatasetChartPoint[];
	    totalRows: number;
	    usedRows: number;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new DatasetChartResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relPath = source["relPath"];
	        this.chartType = source["chartType"];
	        this.categoryColumn = source["categoryColumn"];
	        this.valueColumn = source["valueColumn"];
	        this.mode = source["mode"];
	        this.points = this.convertValues(source["points"], DatasetChartPoint);
	        this.totalRows = source["totalRows"];
	        this.usedRows = source["usedRows"];
	        this.message = source["message"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DatasetQueryResult {
	    relPath: string;
	    query: string;
	    columns: string[];
	    rows: string[][];
	    totalRows: number;
	    matchedRows: number;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new DatasetQueryResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relPath = source["relPath"];
	        this.query = source["query"];
	        this.columns = source["columns"];
	        this.rows = source["rows"];
	        this.totalRows = source["totalRows"];
	        this.matchedRows = source["matchedRows"];
	        this.message = source["message"];
	    }
	}
	export class FileChange {
	    relPath: string;
	    kind: string;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new FileChange(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relPath = source["relPath"];
	        this.kind = source["kind"];
	        this.message = source["message"];
	    }
	}
	export class FileCopyProposal {
	    sourceRelPath: string;
	    targetRelPath: string;
	    name: string;
	    action: string;
	    size: number;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new FileCopyProposal(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceRelPath = source["sourceRelPath"];
	        this.targetRelPath = source["targetRelPath"];
	        this.name = source["name"];
	        this.action = source["action"];
	        this.size = source["size"];
	        this.message = source["message"];
	    }
	}
	export class FileCopyRequest {
	    sourceRelPath: string;
	    targetRelPath: string;

	    static createFrom(source: any = {}) {
	        return new FileCopyRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceRelPath = source["sourceRelPath"];
	        this.targetRelPath = source["targetRelPath"];
	    }
	}
	export class FileDeleteProposal {
	    relPath: string;
	    name: string;
	    action: string;
	    diff: string;
	    size: number;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new FileDeleteProposal(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relPath = source["relPath"];
	        this.name = source["name"];
	        this.action = source["action"];
	        this.diff = source["diff"];
	        this.size = source["size"];
	        this.message = source["message"];
	    }
	}
	export class FileMoveProposal {
	    sourceRelPath: string;
	    targetRelPath: string;
	    name: string;
	    action: string;
	    size: number;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new FileMoveProposal(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceRelPath = source["sourceRelPath"];
	        this.targetRelPath = source["targetRelPath"];
	        this.name = source["name"];
	        this.action = source["action"];
	        this.size = source["size"];
	        this.message = source["message"];
	    }
	}
	export class FileMoveRequest {
	    sourceRelPath: string;
	    targetRelPath: string;

	    static createFrom(source: any = {}) {
	        return new FileMoveRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceRelPath = source["sourceRelPath"];
	        this.targetRelPath = source["targetRelPath"];
	    }
	}
	export class FileNode {
	    name: string;
	    path: string;
	    relPath: string;
	    kind: string;
	    fileType: string;
	    depth: number;
	    meta: string;

	    static createFrom(source: any = {}) {
	        return new FileNode(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.relPath = source["relPath"];
	        this.kind = source["kind"];
	        this.fileType = source["fileType"];
	        this.depth = source["depth"];
	        this.meta = source["meta"];
	    }
	}
	export class TablePreview {
	    columns: string[];
	    rows: string[][];
	    profiles: ColumnProfile[];
	    totalRows: number;
	    truncated: boolean;

	    static createFrom(source: any = {}) {
	        return new TablePreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.columns = source["columns"];
	        this.rows = source["rows"];
	        this.profiles = this.convertValues(source["profiles"], ColumnProfile);
	        this.totalRows = source["totalRows"];
	        this.truncated = source["truncated"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TextPage {
	    page: number;
	    text: string;

	    static createFrom(source: any = {}) {
	        return new TextPage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.page = source["page"];
	        this.text = source["text"];
	    }
	}
	export class FilePreview {
	    relPath: string;
	    name: string;
	    kind: string;
	    fileType: string;
	    content: string;
	    text: string;
	    pages?: TextPage[];
	    encoding: string;
	    table?: TablePreview;
	    truncated: boolean;
	    message: string;
	    size: number;

	    static createFrom(source: any = {}) {
	        return new FilePreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relPath = source["relPath"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.fileType = source["fileType"];
	        this.content = source["content"];
	        this.text = source["text"];
	        this.pages = this.convertValues(source["pages"], TextPage);
	        this.encoding = source["encoding"];
	        this.table = this.convertValues(source["table"], TablePreview);
	        this.truncated = source["truncated"];
	        this.message = source["message"];
	        this.size = source["size"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FileWriteProposal {
	    relPath: string;
	    name: string;
	    action: string;
	    diff: string;
	    encoding: string;
	    size: number;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new FileWriteProposal(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relPath = source["relPath"];
	        this.name = source["name"];
	        this.action = source["action"];
	        this.diff = source["diff"];
	        this.encoding = source["encoding"];
	        this.size = source["size"];
	        this.message = source["message"];
	    }
	}
	export class FileWriteRequest {
	    relPath: string;
	    content: string;
	    encoding: string;

	    static createFrom(source: any = {}) {
	        return new FileWriteRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relPath = source["relPath"];
	        this.content = source["content"];
	        this.encoding = source["encoding"];
	    }
	}
	export class FreshnessStatus {
	    changed: FileChange[];
	    staleArtifacts: string[];
	    staleDatasets: string[];
	    message: string;

	    static createFrom(source: any = {}) {
	        return new FreshnessStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.changed = this.convertValues(source["changed"], FileChange);
	        this.staleArtifacts = source["staleArtifacts"];
	        this.staleDatasets = source["staleDatasets"];
	        this.message = source["message"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WorkspaceProblem {
	    relPath: string;
	    name: string;
	    severity: string;
	    source: string;
	    message: string;
	    line: number;

	    static createFrom(source: any = {}) {
	        return new WorkspaceProblem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relPath = source["relPath"];
	        this.name = source["name"];
	        this.severity = source["severity"];
	        this.source = source["source"];
	        this.message = source["message"];
	        this.line = source["line"];
	    }
	}
	export class ProblemSummary {
	    problems: WorkspaceProblem[];
	    message: string;
	    generatedAt: string;
	    truncated: boolean;

	    static createFrom(source: any = {}) {
	        return new ProblemSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.problems = this.convertValues(source["problems"], WorkspaceProblem);
	        this.message = source["message"];
	        this.generatedAt = source["generatedAt"];
	        this.truncated = source["truncated"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ScanStatus {
	    included: number;
	    ignored: number;
	    depthSkipped: number;
	    entrySkipped: number;
	    unreadable: number;
	    maxDepth: number;
	    maxEntries: number;
	    ignoredSamples: string[];
	    skippedSamples: string[];

	    static createFrom(source: any = {}) {
	        return new ScanStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.included = source["included"];
	        this.ignored = source["ignored"];
	        this.depthSkipped = source["depthSkipped"];
	        this.entrySkipped = source["entrySkipped"];
	        this.unreadable = source["unreadable"];
	        this.maxDepth = source["maxDepth"];
	        this.maxEntries = source["maxEntries"];
	        this.ignoredSamples = source["ignoredSamples"];
	        this.skippedSamples = source["skippedSamples"];
	    }
	}
	export class SearchResult {
	    relPath: string;
	    name: string;
	    kind: string;
	    fileType: string;
	    matchType: string;
	    line: number;
	    snippet: string;

	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.relPath = source["relPath"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.fileType = source["fileType"];
	        this.matchType = source["matchType"];
	        this.line = source["line"];
	        this.snippet = source["snippet"];
	    }
	}



	export class WorkspaceSnapshot {
	    root: string;
	    name: string;
	    nodes: FileNode[];
	    truncated: boolean;
	    scan: ScanStatus;

	    static createFrom(source: any = {}) {
	        return new WorkspaceSnapshot(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.root = source["root"];
	        this.name = source["name"];
	        this.nodes = this.convertValues(source["nodes"], FileNode);
	        this.truncated = source["truncated"];
	        this.scan = this.convertValues(source["scan"], ScanStatus);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}
