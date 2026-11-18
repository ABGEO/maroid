import AwsS3 from '@uppy/aws-s3';
import Uppy from '@uppy/core';

import { failureMessage, type FileClient, type FileRecord } from '@maroid/plugin-sdk';

const POLL_INTERVAL_MS = 2_000;
const POLL_LIMIT_MS = 60_000;

/** Where one file stands, from the PUT to the result of the worker. */
export type UploadPhase = 'uploading' | 'processing' | 'ready' | 'failed' | 'refused';

export interface UploadEntry {
	/** The identifier that Uppy gave the file in the browser. */
	id: string;
	name: string;
	/** The share of the bytes that reached the file store, from 0 to 100. */
	progress: number;
	phase: UploadPhase;
	/** The file as the hub last answered it. */
	file?: FileRecord;
	/** Why the browser or the hub refused the file, or why the result is not known yet. */
	message?: string;
}

export interface UploaderOptions {
	client: FileClient;
	/** The key of the file kind of the plugin. */
	kind: string;
	/** The media types of the file kind. */
	accept: string[];
	/** The size limit of the file kind, in bytes. */
	maxSize: number;
	maxFiles?: number;
	/** Called once the bytes landed and the hub took the completion. */
	onuploaded?: (file: FileRecord) => void;
	/** Called once the file is ready or failed. */
	onsettled?: (file: FileRecord) => void;
}

type SignedPut = { url: string; headers: Record<string, string> };

/**
 * Uploader runs Uppy over the upload addresses of the hub. Uppy checks the type and
 * the size in the browser and sends each PUT. The hub signs each one, then processes
 * the file, which the uploader reads until it settles.
 */
export class Uploader {
	readonly uppy: Uppy;
	entries = $state<UploadEntry[]>([]);

	readonly #options: UploaderOptions;
	readonly #timers = new Set<ReturnType<typeof setTimeout>>();
	#closed = false;

	constructor(options: UploaderOptions) {
		this.#options = options;
		this.uppy = new Uppy({ autoProceed: true });
		this.restrict();

		this.uppy.use(AwsS3, {
			shouldUseMultipart: false,
			generateObjectKey: (file) => file.id,
			signRequest: (request) => this.#sign(request)
		});

		this.uppy.on('file-added', (file) => {
			this.entries.push({ id: file.id, name: file.name ?? '', progress: 0, phase: 'uploading' });
		});

		this.uppy.on('upload-progress', (file, progress) => {
			if (file && progress.bytesTotal) {
				this.#update(file.id, {
					progress: Math.round((progress.bytesUploaded / progress.bytesTotal) * 100)
				});
			}
		});

		this.uppy.on('upload-success', (file) => {
			if (file) {
				void this.#complete(file.id);
			}
		});

		this.uppy.on('upload-error', (file, error) => {
			if (file) {
				this.#update(file.id, {
					phase: 'refused',
					message: failureMessage(error, 'The upload did not finish. Try again.')
				});
			}
		});

		this.uppy.on('restriction-failed', (file, error) => {
			this.entries.push({
				id: file?.id ?? crypto.randomUUID(),
				name: file?.name ?? '',
				progress: 0,
				phase: 'refused',
				message: error.message
			});
		});
	}

	/** Applies the media types, the size limit, and the file limit of the options to Uppy. */
	restrict(): void {
		this.uppy.setOptions({
			restrictions: {
				allowedFileTypes: this.#options.accept,
				maxFileSize: this.#options.maxSize,
				maxNumberOfFiles: this.#options.maxFiles ?? null
			}
		});
	}

	/** Stops every read of a file and removes the files from Uppy. */
	destroy(): void {
		this.#closed = true;

		for (const timer of this.#timers) {
			clearTimeout(timer);
		}

		this.#timers.clear();
		this.uppy.destroy();
	}

	/** The hub signs the PUT of one file, and the start of the upload writes the file. */
	async #sign(request: { method: string; key: string }): Promise<SignedPut> {
		const file = this.uppy.getFile(request.key);
		if (request.method !== 'PUT' || !file) {
			throw new Error('the upload signs the PUT of a known file alone');
		}

		const ticket = await this.#options.client.start({
			kind: this.#options.kind,
			name: file.name ?? '',
			media_type: file.type,
			size: file.size ?? 0
		});

		this.#update(file.id, { file: ticket.file });

		return { url: ticket.upload.url, headers: ticket.upload.headers };
	}

	async #complete(entryId: string): Promise<void> {
		const entry = this.#entryOf(entryId);
		if (!entry?.file) {
			return;
		}

		try {
			await this.#options.client.complete(entry.file.id);
		} catch (error) {
			this.#update(entryId, {
				phase: 'refused',
				message: failureMessage(error, 'The hub did not take the upload. Try again.')
			});

			return;
		}

		this.#update(entryId, { phase: 'processing', progress: 100 });
		this.#options.onuploaded?.(entry.file);

		await this.#follow(entryId, entry.file.id);
	}

	/**
	 * Reads the file every 2 seconds until it settles, for at most 60 seconds. A file
	 * that is still processing then says so, and the page shows its state at the next read.
	 */
	async #follow(entryId: string, fileId: string): Promise<void> {
		const deadline = Date.now() + POLL_LIMIT_MS;

		while (!this.#closed && Date.now() < deadline) {
			await this.#wait(POLL_INTERVAL_MS);
			if (this.#closed) {
				return;
			}

			let file: FileRecord;
			try {
				file = await this.#options.client.get(fileId);
			} catch {
				continue;
			}

			if (file.state === 'ready' || file.state === 'failed') {
				this.#update(entryId, { file, phase: file.state });
				this.#options.onsettled?.(file);

				return;
			}

			this.#update(entryId, { file });
		}

		if (!this.#closed) {
			this.#update(entryId, { message: 'Still processing. Check again later.' });
		}
	}

	#wait(milliseconds: number): Promise<void> {
		return new Promise((resolve) => {
			const timer = setTimeout(() => {
				this.#timers.delete(timer);
				resolve();
			}, milliseconds);
			this.#timers.add(timer);
		});
	}

	#entryOf(id: string): UploadEntry | undefined {
		return this.entries.find((entry) => entry.id === id);
	}

	#update(id: string, change: Partial<UploadEntry>): void {
		const entry = this.#entryOf(id);
		if (entry) {
			Object.assign(entry, change);
		}
	}
}
