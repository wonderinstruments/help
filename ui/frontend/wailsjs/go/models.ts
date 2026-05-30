export namespace main {
	
	export class DocInfo {
	    Path: string;
	    Title: string;
	    Tags: string[];
	
	    static createFrom(source: any = {}) {
	        return new DocInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Path = source["Path"];
	        this.Title = source["Title"];
	        this.Tags = source["Tags"];
	    }
	}
	export class UISearchResult {
	    DocPath: string;
	    DocTitle: string;
	    Heading: string;
	    Content: string;
	    Tags: string;
	    Score: number;
	
	    static createFrom(source: any = {}) {
	        return new UISearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.DocPath = source["DocPath"];
	        this.DocTitle = source["DocTitle"];
	        this.Heading = source["Heading"];
	        this.Content = source["Content"];
	        this.Tags = source["Tags"];
	        this.Score = source["Score"];
	    }
	}

}

