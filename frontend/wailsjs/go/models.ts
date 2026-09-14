export namespace config {
	
	export class BrowserConfig {
	    extraHosts: string[];
	
	    static createFrom(source: any = {}) {
	        return new BrowserConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.extraHosts = source["extraHosts"];
	    }
	}
	export class InboxConfig {
	    followLatest: boolean;
	
	    static createFrom(source: any = {}) {
	        return new InboxConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.followLatest = source["followLatest"];
	    }
	}
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
	    inbox: InboxConfig;
	    browser: BrowserConfig;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sources = this.convertValues(source["sources"], SourceConfig);
	        this.notifications = this.convertValues(source["notifications"], NotifyConfig);
	        this.editor = this.convertValues(source["editor"], EditorConfig);
	        this.inbox = this.convertValues(source["inbox"], InboxConfig);
	        this.browser = this.convertValues(source["browser"], BrowserConfig);
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

export namespace ingest {
	
	export class Frame {
	    file: string;
	    line: number;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new Frame(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file = source["file"];
	        this.line = source["line"];
	        this.text = source["text"];
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
	    frames: ingest.Frame[];
	
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
	        this.frames = this.convertValues(source["frames"], ingest.Frame);
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
	export class AppState {
	    events: EventDTO[];
	    sources: source.Status[];
	    running: boolean;
	    total: number;
	    marked: boolean;
	    ingestAddr: string;
	    sourceCounts: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new AppState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.events = this.convertValues(source["events"], EventDTO);
	        this.sources = this.convertValues(source["sources"], source.Status);
	        this.running = source["running"];
	        this.total = source["total"];
	        this.marked = source["marked"];
	        this.ingestAddr = source["ingestAddr"];
	        this.sourceCounts = source["sourceCounts"];
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
	    recent: statefile.Project[];
	    editors: string[];
	    ingestAddr: string;
	
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
	        this.recent = this.convertValues(source["recent"], statefile.Project);
	        this.editors = source["editors"];
	        this.ingestAddr = source["ingestAddr"];
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

export namespace persist {
	
	export class Mute {
	    kind: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new Mute(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.value = source["value"];
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

export namespace statefile {
	
	export class Project {
	    dir: string;
	    configPath: string;
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dir = source["dir"];
	        this.configPath = source["configPath"];
	    }
	}

}

