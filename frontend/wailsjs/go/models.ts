export namespace agentcore {
	
	export class Events {
	
	
	    static createFrom(source: any = {}) {
	        return new Events(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class ToolDef {
	    name: string;
	    description: string;
	    input_schema: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new ToolDef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.input_schema = source["input_schema"];
	    }
	}
	export class Skill {
	    ID: string;
	    SystemPrompt: string;
	    Tools: ToolDef[];
	
	    static createFrom(source: any = {}) {
	        return new Skill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.SystemPrompt = source["SystemPrompt"];
	        this.Tools = this.convertValues(source["Tools"], ToolDef);
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
	export class LoopOptions {
	    Skill?: Skill;
	    // Go type: Events
	    Events: any;
	    MaxTurns: number;
	    MaxHistory: number;
	
	    static createFrom(source: any = {}) {
	        return new LoopOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Skill = this.convertValues(source["Skill"], Skill);
	        this.Events = this.convertValues(source["Events"], null);
	        this.MaxTurns = source["MaxTurns"];
	        this.MaxHistory = source["MaxHistory"];
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
	export class ToolResult {
	    id: string;
	    name: string;
	    output: string;
	    is_error?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ToolResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.output = source["output"];
	        this.is_error = source["is_error"];
	    }
	}
	export class ToolCall {
	    id: string;
	    name: string;
	    input: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new ToolCall(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.input = source["input"];
	    }
	}
	export class Message {
	    role: string;
	    text?: string;
	    tool_calls?: ToolCall[];
	    tool_result?: ToolResult;
	
	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.text = source["text"];
	        this.tool_calls = this.convertValues(source["tool_calls"], ToolCall);
	        this.tool_result = this.convertValues(source["tool_result"], ToolResult);
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
	export class RunResult {
	    text: string;
	    cancelled: boolean;
	    turn_limit: boolean;
	    truncated?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RunResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.cancelled = source["cancelled"];
	        this.turn_limit = source["turn_limit"];
	        this.truncated = source["truncated"];
	    }
	}
	
	
	

}

export namespace aiprovider {
	
	export class ChatMessage {
	    role: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new ChatMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	    }
	}
	export class ChatRequest {
	    messages: ChatMessage[];
	    tools?: any[];
	    max_tokens?: number;
	    temperature?: number;
	
	    static createFrom(source: any = {}) {
	        return new ChatRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.messages = this.convertValues(source["messages"], ChatMessage);
	        this.tools = source["tools"];
	        this.max_tokens = source["max_tokens"];
	        this.temperature = source["temperature"];
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
	export class UsageInfo {
	    prompt_tokens: number;
	    completion_tokens: number;
	    total_tokens: number;
	
	    static createFrom(source: any = {}) {
	        return new UsageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.prompt_tokens = source["prompt_tokens"];
	        this.completion_tokens = source["completion_tokens"];
	        this.total_tokens = source["total_tokens"];
	    }
	}
	export class ChatToolCall {
	    id: string;
	    name: string;
	    input_json: string;
	
	    static createFrom(source: any = {}) {
	        return new ChatToolCall(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.input_json = source["input_json"];
	    }
	}
	export class ChatResponse {
	    text: string;
	    tool_calls?: ChatToolCall[];
	    model?: string;
	    usage?: UsageInfo;
	
	    static createFrom(source: any = {}) {
	        return new ChatResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.tool_calls = this.convertValues(source["tool_calls"], ChatToolCall);
	        this.model = source["model"];
	        this.usage = this.convertValues(source["usage"], UsageInfo);
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
	
	export class Settings {
	    provider: string;
	    api_key: string;
	    model: string;
	    base_url?: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.api_key = source["api_key"];
	        this.model = source["model"];
	        this.base_url = source["base_url"];
	    }
	}

}

export namespace docs {
	
	export class ParaInfo {
	    index: number;
	    text: string;
	    style?: string;
	    alignment?: string;
	    is_bold?: boolean;
	    is_italic?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ParaInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.text = source["text"];
	        this.style = source["style"];
	        this.alignment = source["alignment"];
	        this.is_bold = source["is_bold"];
	        this.is_italic = source["is_italic"];
	    }
	}
	export class DocState {
	    file_path: string;
	    is_dirty: boolean;
	    title: string;
	    word_count: number;
	    char_count: number;
	    page_count: number;
	    paragraphs?: ParaInfo[];
	
	    static createFrom(source: any = {}) {
	        return new DocState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_path = source["file_path"];
	        this.is_dirty = source["is_dirty"];
	        this.title = source["title"];
	        this.word_count = source["word_count"];
	        this.char_count = source["char_count"];
	        this.page_count = source["page_count"];
	        this.paragraphs = this.convertValues(source["paragraphs"], ParaInfo);
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
	export class FindReplaceResult {
	    count: number;
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new FindReplaceResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.count = source["count"];
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	export class OpenFileResult {
	    success: boolean;
	    file_path: string;
	    title: string;
	    error?: string;
	    word_count: number;
	    char_count: number;
	    page_count: number;
	    paragraphs?: ParaInfo[];
	
	    static createFrom(source: any = {}) {
	        return new OpenFileResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.file_path = source["file_path"];
	        this.title = source["title"];
	        this.error = source["error"];
	        this.word_count = source["word_count"];
	        this.char_count = source["char_count"];
	        this.page_count = source["page_count"];
	        this.paragraphs = this.convertValues(source["paragraphs"], ParaInfo);
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
	
	export class SaveResult {
	    success: boolean;
	    file_path: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.file_path = source["file_path"];
	        this.error = source["error"];
	    }
	}

}

export namespace pdf {
	
	export class OpResult {
	    success: boolean;
	    file_path?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new OpResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.file_path = source["file_path"];
	        this.error = source["error"];
	    }
	}
	export class PDFMeta {
	    title?: string;
	    author?: string;
	    subject?: string;
	    creator?: string;
	    producer?: string;
	
	    static createFrom(source: any = {}) {
	        return new PDFMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.author = source["author"];
	        this.subject = source["subject"];
	        this.creator = source["creator"];
	        this.producer = source["producer"];
	    }
	}
	export class PageInfo {
	    index: number;
	    width: number;
	    height: number;
	
	    static createFrom(source: any = {}) {
	        return new PageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.width = source["width"];
	        this.height = source["height"];
	    }
	}
	export class PDFState {
	    file_path: string;
	    title: string;
	    is_dirty: boolean;
	    page_count: number;
	    pages: PageInfo[];
	    meta: PDFMeta;
	
	    static createFrom(source: any = {}) {
	        return new PDFState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_path = source["file_path"];
	        this.title = source["title"];
	        this.is_dirty = source["is_dirty"];
	        this.page_count = source["page_count"];
	        this.pages = this.convertValues(source["pages"], PageInfo);
	        this.meta = this.convertValues(source["meta"], PDFMeta);
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

export namespace projectstore {
	
	export class ChatMessage {
	    role: string;
	    text: string;
	    // Go type: time
	    timestamp: any;
	
	    static createFrom(source: any = {}) {
	        return new ChatMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.text = source["text"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
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
	export class ChatMeta {
	    id: string;
	    project_id: string;
	    messages: ChatMessage[];
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new ChatMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.project_id = source["project_id"];
	        this.messages = this.convertValues(source["messages"], ChatMessage);
	        this.created_at = this.convertValues(source["created_at"], null);
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
	export class ProjectInfo {
	    id: string;
	    name: string;
	    file_path: string;
	    kind: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new ProjectInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.file_path = source["file_path"];
	        this.kind = source["kind"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
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

export namespace sheets {
	
	export class MergeCell {
	    start_cell: string;
	    end_cell: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new MergeCell(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start_cell = source["start_cell"];
	        this.end_cell = source["end_cell"];
	        this.value = source["value"];
	    }
	}
	export class SaveResult {
	    success: boolean;
	    file_path?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.file_path = source["file_path"];
	        this.error = source["error"];
	    }
	}
	export class WorksheetInfo {
	    id: number;
	    name: string;
	    index: number;
	    row_count: number;
	    column_count: number;
	    hidden: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WorksheetInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.index = source["index"];
	        this.row_count = source["row_count"];
	        this.column_count = source["column_count"];
	        this.hidden = source["hidden"];
	    }
	}
	export class WorkbookState {
	    file_path: string;
	    title: string;
	    is_dirty: boolean;
	    sheets: WorksheetInfo[];
	    active_sheet: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkbookState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_path = source["file_path"];
	        this.title = source["title"];
	        this.is_dirty = source["is_dirty"];
	        this.sheets = this.convertValues(source["sheets"], WorksheetInfo);
	        this.active_sheet = source["active_sheet"];
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

export namespace shell {
	
	export class AppSettings {
	    language: string;
	    theme: string;
	    onboard_done: boolean;
	    default_save_dir?: string;
	    update_channel: string;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.language = source["language"];
	        this.theme = source["theme"];
	        this.onboard_done = source["onboard_done"];
	        this.default_save_dir = source["default_save_dir"];
	        this.update_channel = source["update_channel"];
	    }
	}
	export class RecentFile {
	    path: string;
	    name: string;
	    kind: string;
	    // Go type: time
	    opened_at: any;
	    is_starred: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RecentFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.opened_at = this.convertValues(source["opened_at"], null);
	        this.is_starred = source["is_starred"];
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
	export class TabSummary {
	    id: string;
	    kind: string;
	    title: string;
	    file_path?: string;
	    is_dirty: boolean;
	    is_active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TabSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.title = source["title"];
	        this.file_path = source["file_path"];
	        this.is_dirty = source["is_dirty"];
	        this.is_active = source["is_active"];
	    }
	}

}

export namespace slides {
	
	export class SlideElement {
	    id: string;
	    kind: string;
	    x: number;
	    y: number;
	    w: number;
	    h: number;
	    text?: string;
	    bold?: boolean;
	    italic?: boolean;
	    align?: string;
	    ph_type?: string;
	    image_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new SlideElement(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.x = source["x"];
	        this.y = source["y"];
	        this.w = source["w"];
	        this.h = source["h"];
	        this.text = source["text"];
	        this.bold = source["bold"];
	        this.italic = source["italic"];
	        this.align = source["align"];
	        this.ph_type = source["ph_type"];
	        this.image_id = source["image_id"];
	    }
	}
	export class SlideInfo {
	    index: number;
	    elements: SlideElement[];
	
	    static createFrom(source: any = {}) {
	        return new SlideInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.elements = this.convertValues(source["elements"], SlideElement);
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
	export class DeckState {
	    file_path: string;
	    is_dirty: boolean;
	    title: string;
	    slide_count: number;
	    slides: SlideInfo[];
	    width: number;
	    height: number;
	
	    static createFrom(source: any = {}) {
	        return new DeckState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_path = source["file_path"];
	        this.is_dirty = source["is_dirty"];
	        this.title = source["title"];
	        this.slide_count = source["slide_count"];
	        this.slides = this.convertValues(source["slides"], SlideInfo);
	        this.width = source["width"];
	        this.height = source["height"];
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
	export class OpenResult {
	    success: boolean;
	    file_path?: string;
	    title: string;
	    error?: string;
	    slide_count: number;
	    slides?: SlideInfo[];
	    width: number;
	    height: number;
	
	    static createFrom(source: any = {}) {
	        return new OpenResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.file_path = source["file_path"];
	        this.title = source["title"];
	        this.error = source["error"];
	        this.slide_count = source["slide_count"];
	        this.slides = this.convertValues(source["slides"], SlideInfo);
	        this.width = source["width"];
	        this.height = source["height"];
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
	export class SaveResult {
	    success: boolean;
	    file_path?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.file_path = source["file_path"];
	        this.error = source["error"];
	    }
	}
	

}

