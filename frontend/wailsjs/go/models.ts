export namespace config {
	
	export class AppConfig {
	    socksPort: number;
	    httpPort: number;
	    allowLan: boolean;
	    proxyMode: string;
	    routingMode: string;
	    directDns: string;
	    remoteDns: string;
	    enableDoh: boolean;
	    themeMode: string;
	    themeColor: string;
	    closeToTray: boolean;
	    autoStart: boolean;
	    latencyTestUrl: string;
	    latencyTimeout: number;
	    latencyMaxConc: number;
	    activeNodeId: string;
	    activeGroupId: string;
	    coreType: string;
	    customDirectRules: string[];
	    customProxyRules: string[];
	    customBlockRules: string[];
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.socksPort = source["socksPort"];
	        this.httpPort = source["httpPort"];
	        this.allowLan = source["allowLan"];
	        this.proxyMode = source["proxyMode"];
	        this.routingMode = source["routingMode"];
	        this.directDns = source["directDns"];
	        this.remoteDns = source["remoteDns"];
	        this.enableDoh = source["enableDoh"];
	        this.themeMode = source["themeMode"];
	        this.themeColor = source["themeColor"];
	        this.closeToTray = source["closeToTray"];
	        this.autoStart = source["autoStart"];
	        this.latencyTestUrl = source["latencyTestUrl"];
	        this.latencyTimeout = source["latencyTimeout"];
	        this.latencyMaxConc = source["latencyMaxConc"];
	        this.activeNodeId = source["activeNodeId"];
	        this.activeGroupId = source["activeGroupId"];
	        this.coreType = source["coreType"];
	        this.customDirectRules = source["customDirectRules"];
	        this.customProxyRules = source["customProxyRules"];
	        this.customBlockRules = source["customBlockRules"];
	    }
	}
	export class DependencyItem {
	    id: string;
	    name: string;
	    required: boolean;
	    installed: boolean;
	    version: string;
	    path: string;
	    description: string;
	    installCmd?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new DependencyItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.required = source["required"];
	        this.installed = source["installed"];
	        this.version = source["version"];
	        this.path = source["path"];
	        this.description = source["description"];
	        this.installCmd = source["installCmd"];
	        this.error = source["error"];
	    }
	}
	export class DependencyCheckResult {
	    ready: boolean;
	    platform: string;
	    binDir: string;
	    items: DependencyItem[];
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new DependencyCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ready = source["ready"];
	        this.platform = source["platform"];
	        this.binDir = source["binDir"];
	        this.items = this.convertValues(source["items"], DependencyItem);
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
	
	export class NodeConfig {
	    id: string;
	    subId: string;
	    groupName: string;
	    name: string;
	    protocol: string;
	    server: string;
	    port: number;
	    uuid?: string;
	    password?: string;
	    alterId?: number;
	    cipher?: string;
	    network?: string;
	    path?: string;
	    host?: string;
	    serviceName?: string;
	    tls: boolean;
	    sni?: string;
	    alpn?: string;
	    fingerprint?: string;
	    security?: string;
	    publicKey?: string;
	    shortId?: string;
	    spiderX?: string;
	    mldsa65Verify?: string;
	    flow?: string;
	    obfsType?: string;
	    obfsPassword?: string;
	    upMbps?: number;
	    downMbps?: number;
	    tcpDelay: number;
	    realDelay: number;
	    rawLink?: string;
	
	    static createFrom(source: any = {}) {
	        return new NodeConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.subId = source["subId"];
	        this.groupName = source["groupName"];
	        this.name = source["name"];
	        this.protocol = source["protocol"];
	        this.server = source["server"];
	        this.port = source["port"];
	        this.uuid = source["uuid"];
	        this.password = source["password"];
	        this.alterId = source["alterId"];
	        this.cipher = source["cipher"];
	        this.network = source["network"];
	        this.path = source["path"];
	        this.host = source["host"];
	        this.serviceName = source["serviceName"];
	        this.tls = source["tls"];
	        this.sni = source["sni"];
	        this.alpn = source["alpn"];
	        this.fingerprint = source["fingerprint"];
	        this.security = source["security"];
	        this.publicKey = source["publicKey"];
	        this.shortId = source["shortId"];
	        this.spiderX = source["spiderX"];
	        this.mldsa65Verify = source["mldsa65Verify"];
	        this.flow = source["flow"];
	        this.obfsType = source["obfsType"];
	        this.obfsPassword = source["obfsPassword"];
	        this.upMbps = source["upMbps"];
	        this.downMbps = source["downMbps"];
	        this.tcpDelay = source["tcpDelay"];
	        this.realDelay = source["realDelay"];
	        this.rawLink = source["rawLink"];
	    }
	}
	export class Subscription {
	    id: string;
	    name: string;
	    url: string;
	    autoUpdate: boolean;
	    updateInterval: number;
	    lastUpdated: number;
	    nodeCount: number;
	    lastError?: string;
	
	    static createFrom(source: any = {}) {
	        return new Subscription(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.url = source["url"];
	        this.autoUpdate = source["autoUpdate"];
	        this.updateInterval = source["updateInterval"];
	        this.lastUpdated = source["lastUpdated"];
	        this.nodeCount = source["nodeCount"];
	        this.lastError = source["lastError"];
	    }
	}

}

