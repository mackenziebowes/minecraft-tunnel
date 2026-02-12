export namespace webrtc {
	
	export class DataChannel {
	
	
	    static createFrom(source: any = {}) {
	        return new DataChannel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

