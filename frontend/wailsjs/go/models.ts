export namespace config {
	
	export class EditorConfig {
	    command: string;
	
	    static createFrom(source: any = {}) {
	        return new EditorConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.command = source["command"];
	    }
	}
	export class NotifyConfig {
	    enabled: boolean;
	    sound: boolean;
	
	    static createFrom(source: any = {}) {
	        return new NotifyConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.sound = source["sound"];
	    }
	}
	export class SourceConfig {
	    name: string;
	    type: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new SourceConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.path = source["path"];
	    }
	}
	export class Config {
	    sources: SourceConfig[];
	    notifications: NotifyConfig;
	    editor: EditorConfig;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sources = this.convertValues(source["sources"], SourceConfig);
	        this.notifications = this.convertValues(source["notifications"], NotifyConfig);
	        this.editor = this.convertValues(source["editor"], EditorConfig);
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

export namespace detect {
	
	export class Candidate {
	    name: string;
	    path: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new Candidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.type = source["type"];
	    }
	}

}

export namespace main {
	
	export class EventDTO {
	    source: string;
	    time: string;
	    type: string;
	    message: string;
	    file: string;
	    line: number;
	    severity: string;
	    stack: string;
	    raw: string;
	    hash: string;
	    count: number;
	    firstSeen: string;
	    lastSeen: string;
	    title: string;
	    location: string;
	
	    static createFrom(source: any = {}) {
	        return new EventDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.time = source["time"];
	        this.type = source["type"];
	        this.message = source["message"];
	        this.file = source["file"];
	        this.line = source["line"];
	        this.severity = source["severity"];
	        this.stack = source["stack"];
	        this.raw = source["raw"];
	        this.hash = source["hash"];
	        this.count = source["count"];
	        this.firstSeen = source["firstSeen"];
	        this.lastSeen = source["lastSeen"];
	        this.title = source["title"];
	        this.location = source["location"];
	    }
	}
	export class AppState {
	    events: EventDTO[];
	    sources: source.Status[];
	    running: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AppState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.events = this.convertValues(source["events"], EventDTO);
	        this.sources = this.convertValues(source["sources"], source.Status);
	        this.running = source["running"];
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
	export class Bootstrap {
	    screen: string;
	    projectDir: string;
	    configPath: string;
	    config?: config.Config;
	    fromStart: boolean;
	    running: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Bootstrap(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.screen = source["screen"];
	        this.projectDir = source["projectDir"];
	        this.configPath = source["configPath"];
	        this.config = this.convertValues(source["config"], config.Config);
	        this.fromStart = source["fromStart"];
	        this.running = source["running"];
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

export namespace source {
	
	export class Status {
	    name: string;
	    type: string;
	    path: string;
	    state: string;
	    message: string;
	    // Go type: time
	    updated: any;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.path = source["path"];
	        this.state = source["state"];
	        this.message = source["message"];
	        this.updated = this.convertValues(source["updated"], null);
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

