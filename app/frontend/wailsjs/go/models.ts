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
	

}

