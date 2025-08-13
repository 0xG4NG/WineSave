export namespace main {
	
	export class BackupConfig {
	    backup_dir: string;
	    max_backups: number;
	    compression_enabled: boolean;
	    scan_interval: number;
	    exclude_patterns: string[];
	    auto_backup: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BackupConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backup_dir = source["backup_dir"];
	        this.max_backups = source["max_backups"];
	        this.compression_enabled = source["compression_enabled"];
	        this.scan_interval = source["scan_interval"];
	        this.exclude_patterns = source["exclude_patterns"];
	        this.auto_backup = source["auto_backup"];
	    }
	}
	export class BackupInfo {
	    path: string;
	    size: number;
	    // Go type: time
	    created: any;
	    compressed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BackupInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.size = source["size"];
	        this.created = this.convertValues(source["created"], null);
	        this.compressed = source["compressed"];
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
	export class GameInfo {
	    id: string;
	    name: string;
	    save_paths: string[];
	    patterns: string[];
	    platform: string;
	    // Go type: time
	    last_backup: any;
	    total_size: number;
	    file_count: number;
	    custom_paths: string[];
	    metadata: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new GameInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.save_paths = source["save_paths"];
	        this.patterns = source["patterns"];
	        this.platform = source["platform"];
	        this.last_backup = this.convertValues(source["last_backup"], null);
	        this.total_size = source["total_size"];
	        this.file_count = source["file_count"];
	        this.custom_paths = source["custom_paths"];
	        this.metadata = source["metadata"];
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
	export class GameSearchResult {
	    name: string;
	    page_id: string;
	    steam_app_id: string;
	    release_date: string;
	    cover_url: string;
	    save_paths: string[];
	
	    static createFrom(source: any = {}) {
	        return new GameSearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.page_id = source["page_id"];
	        this.steam_app_id = source["steam_app_id"];
	        this.release_date = source["release_date"];
	        this.cover_url = source["cover_url"];
	        this.save_paths = source["save_paths"];
	    }
	}
	export class ScanResult {
	    total_games: number;
	    new_games: GameInfo[];
	    updated: GameInfo[];
	    errors: string[];
	    scan_time: number;
	
	    static createFrom(source: any = {}) {
	        return new ScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_games = source["total_games"];
	        this.new_games = this.convertValues(source["new_games"], GameInfo);
	        this.updated = this.convertValues(source["updated"], GameInfo);
	        this.errors = source["errors"];
	        this.scan_time = source["scan_time"];
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
	export class UserGameSelection {
	    name: string;
	    selected_game?: GameSearchResult;
	    custom_path: string;
	    backup_path: string;
	
	    static createFrom(source: any = {}) {
	        return new UserGameSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.selected_game = this.convertValues(source["selected_game"], GameSearchResult);
	        this.custom_path = source["custom_path"];
	        this.backup_path = source["backup_path"];
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

