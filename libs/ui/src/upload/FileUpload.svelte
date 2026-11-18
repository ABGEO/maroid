<script lang="ts">
	import { UppyContextProvider } from '@uppy/svelte';
	import { onDestroy } from 'svelte';

	import type { FileClient, FileFailure, FileRecord } from '@maroid/plugin-sdk';

	import DropZone from './DropZone.svelte';
	import { Uploader, type UploadEntry } from './uploader.svelte';

	interface Props {
		client: FileClient;
		/** The key of the file kind of the plugin. */
		kind: string;
		/** The media types of the file kind. */
		accept: string[];
		/** The size limit of the file kind, in bytes. */
		maxSize: number;
		maxFiles?: number;
		label?: string;
		/** Called once the bytes landed, so the page saves the identifier of the file. */
		onuploaded?: (file: FileRecord) => void;
		/** Called once the file is ready or failed. */
		onsettled?: (file: FileRecord) => void;
	}

	let {
		client,
		kind,
		accept,
		maxSize,
		maxFiles,
		label = 'Drop files here or choose them',
		onuploaded,
		onsettled
	}: Props = $props();

	const MEGABYTE = 1024 * 1024;

	const failures: Record<FileFailure, string> = {
		media_type: 'The content is not a type that this field accepts.',
		damaged: 'The file is damaged and does not open.',
		bytes_absent: 'The bytes never reached the file store.'
	};

	const uploader = new Uploader({
		get client() {
			return client;
		},
		get kind() {
			return kind;
		},
		get accept() {
			return accept;
		},
		get maxSize() {
			return maxSize;
		},
		get maxFiles() {
			return maxFiles;
		},
		onuploaded: (file) => onuploaded?.(file),
		onsettled: (file) => onsettled?.(file)
	});

	$effect(() => uploader.restrict());

	onDestroy(() => uploader.destroy());

	const hint = $derived(`Up to ${Math.floor(maxSize / MEGABYTE)} MB each`);

	function stateOf(entry: UploadEntry): string {
		switch (entry.phase) {
			case 'uploading':
				return `Uploading ${entry.progress}%`;
			case 'processing':
				return entry.message ?? 'Processing';
			case 'ready':
				return 'Ready';
			case 'failed':
				return entry.file?.failure ? failures[entry.file.failure] : 'The file failed.';
			case 'refused':
				return entry.message ?? 'The upload did not finish.';
		}
	}
</script>

<div class="flex flex-col gap-3">
	<UppyContextProvider uppy={uploader.uppy}>
		<DropZone {label} {hint} {accept} />
	</UppyContextProvider>

	{#if uploader.entries.length > 0}
		<ul class="flex flex-col gap-2">
			{#each uploader.entries as entry (entry.id)}
				<li class="flex flex-col gap-1">
					<div class="flex items-center justify-between gap-2 text-sm">
						<span class="truncate">{entry.name}</span>
						<span
							class={[
								'shrink-0 text-xs',
								entry.phase === 'ready' && 'text-success',
								(entry.phase === 'failed' || entry.phase === 'refused') && 'text-error',
								(entry.phase === 'uploading' || entry.phase === 'processing') &&
									'text-base-content/60'
							]}
						>
							{stateOf(entry)}
						</span>
					</div>
					{#if entry.phase === 'uploading'}
						<progress
							class="progress progress-primary w-full"
							value={entry.progress}
							max="100"
							aria-label={`Upload of ${entry.name}`}
						></progress>
					{:else if entry.phase === 'processing' && !entry.message}
						<progress class="progress w-full" aria-label={`Processing of ${entry.name}`}></progress>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</div>
