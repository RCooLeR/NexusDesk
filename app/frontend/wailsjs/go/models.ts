export namespace llm {
	
	export class ProbeResult {
	    ok: boolean;
	    message: string;
	    endpoint: string;
	    modelCount: number;
	    modelSample: string[];
	
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
	    }
	}

}

export namespace main {
	
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

}

export namespace storage {
	
	export class LLMSettings {
	    providerName: string;
	    baseUrl: string;
	    model: string;
	    apiKey: string;
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
	export class WorkspaceSnapshot {
	    root: string;
	    name: string;
	    nodes: FileNode[];
	    truncated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.root = source["root"];
	        this.name = source["name"];
	        this.nodes = this.convertValues(source["nodes"], FileNode);
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

}

