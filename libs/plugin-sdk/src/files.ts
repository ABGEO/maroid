/** The state of a file, from the start of its upload to its result. */
export type FileState = 'pending' | 'processing' | 'ready' | 'failed';

/** Why a file failed: the content is of another type, does not decode, or never came. */
export type FileFailure = 'media_type' | 'damaged' | 'bytes_absent';

/** A form of a ready file. An image holds `thumbnail` and `display`, any other file `original`. */
export type FileVariant = 'original' | 'thumbnail' | 'display';

/** A signed address that reads one variant until it expires. */
export interface ReadAddress {
	url: string;
	expires_at: string;
}

/** One file of a plugin, as the hub answers it. */
export interface FileRecord {
	id: string;
	kind: string;
	state: FileState;
	failure?: FileFailure;
	name: string;
	/** The declared type until the worker runs, then the detected type. */
	media_type: string;
	size: number;
	/** The size of a ready image, after the orientation. */
	width?: number;
	height?: number;
	uploaded_by: string;
	uploaded_at: string;
	claimed: boolean;
	/** Present when the state is `ready`. */
	addresses?: Partial<Record<FileVariant, ReadAddress>>;
}

/** What a person declares at the start of an upload. */
export interface FileStart {
	/** The key of a file kind of the plugin. */
	kind: string;
	name: string;
	media_type: string;
	size: number;
}

/** A pending file, and the address that takes its bytes with one PUT. */
export interface UploadTicket {
	file: FileRecord;
	upload: {
		url: string;
		method: 'PUT';
		/** The headers that the PUT must carry, as the hub signed them. */
		headers: Record<string, string>;
		expires_at: string;
	};
}

/** The files of the plugin in the acting workspace. Each call checks the permission of the kind. */
export interface FileClient {
	/** Start the upload of one file, and answer the address that takes its bytes. */
	start(input: FileStart): Promise<UploadTicket>;
	/** Report that the bytes reached the file store, so the hub processes the file. */
	complete(id: string): Promise<void>;
	/** Read one file, with a read address for each variant once it is ready. */
	get(id: string): Promise<FileRecord>;
}
