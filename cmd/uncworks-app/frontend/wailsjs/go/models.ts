export namespace main {
	
	export class AppSettings {
	    githubToken: string;
	    namespace: string;
	    kubeContext: string;
	    portRangeStart: number;
	    portRangeEnd: number;
	    envOverrides: Record<string, string>;
	    litellmURL: string;
	    githubAuthed: boolean;
	    updateChannel: string;
	    autoUpdateEnabled: boolean;
	    defaultManageModel: string;
	    defaultImplementModel: string;
	    wizardComplete: boolean;
	    apiserverURL: string;
	    llmApiKey: string;
	    llmKeyConfigured: boolean;
	    showTrafficLights: boolean;
	    copilotModel: string;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.githubToken = source["githubToken"];
	        this.namespace = source["namespace"];
	        this.kubeContext = source["kubeContext"];
	        this.portRangeStart = source["portRangeStart"];
	        this.portRangeEnd = source["portRangeEnd"];
	        this.envOverrides = source["envOverrides"];
	        this.litellmURL = source["litellmURL"];
	        this.githubAuthed = source["githubAuthed"];
	        this.updateChannel = source["updateChannel"];
	        this.autoUpdateEnabled = source["autoUpdateEnabled"];
	        this.defaultManageModel = source["defaultManageModel"];
	        this.defaultImplementModel = source["defaultImplementModel"];
	        this.wizardComplete = source["wizardComplete"];
	        this.apiserverURL = source["apiserverURL"];
	        this.llmApiKey = source["llmApiKey"];
	        this.llmKeyConfigured = source["llmKeyConfigured"];
	        this.showTrafficLights = source["showTrafficLights"];
	        this.copilotModel = source["copilotModel"];
	    }
	}
	export class DeviceFlowPollResult {
	    done: boolean;
	    token?: string;
	
	    static createFrom(source: any = {}) {
	        return new DeviceFlowPollResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.done = source["done"];
	        this.token = source["token"];
	    }
	}
	export class DeviceFlowStart {
	    device_code: string;
	    user_code: string;
	    verification_uri: string;
	    expires_in: number;
	    interval: number;
	
	    static createFrom(source: any = {}) {
	        return new DeviceFlowStart(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.device_code = source["device_code"];
	        this.user_code = source["user_code"];
	        this.verification_uri = source["verification_uri"];
	        this.expires_in = source["expires_in"];
	        this.interval = source["interval"];
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
	export class LiteLLMCheckResult {
	    ok: boolean;
	    models: string[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new LiteLLMCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.models = source["models"];
	        this.error = source["error"];
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
	export class UpdateInfo {
	    localBuild: boolean;
	    upToDate: boolean;
	    currentVersion?: string;
	    latestVersion?: string;
	    releaseURL?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.localBuild = source["localBuild"];
	        this.upToDate = source["upToDate"];
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.releaseURL = source["releaseURL"];
	        this.error = source["error"];
	    }
	}

}

