export namespace main {
	
	export class AppSettings {
	    llmKey: string;
	    githubToken: string;
	    namespace: string;
	    kubeContext: string;
	    portRangeStart: number;
	    portRangeEnd: number;
	    envOverrides: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.llmKey = source["llmKey"];
	        this.githubToken = source["githubToken"];
	        this.namespace = source["namespace"];
	        this.kubeContext = source["kubeContext"];
	        this.portRangeStart = source["portRangeStart"];
	        this.portRangeEnd = source["portRangeEnd"];
	        this.envOverrides = source["envOverrides"];
	    }
	}
	export class EnvVarInfo {
	    key: string;
	    system: string;
	    override: string;
	    desc: string;
	
	    static createFrom(source: any = {}) {
	        return new EnvVarInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.system = source["system"];
	        this.override = source["override"];
	        this.desc = source["desc"];
	    }
	}
	export class HealthComponent {
	    name: string;
	    label: string;
	    status: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new HealthComponent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.label = source["label"];
	        this.status = source["status"];
	        this.message = source["message"];
	    }
	}
	export class HealthReport {
	    overall: string;
	    components: HealthComponent[];
	
	    static createFrom(source: any = {}) {
	        return new HealthReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.overall = source["overall"];
	        this.components = this.convertValues(source["components"], HealthComponent);
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
	export class ServiceInfo {
	    name: string;
	    displayName: string;
	    clusterPort: number;
	    localPort: number;
	    ready: boolean;
	    forwarding: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ServiceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.clusterPort = source["clusterPort"];
	        this.localPort = source["localPort"];
	        this.ready = source["ready"];
	        this.forwarding = source["forwarding"];
	    }
	}

}

