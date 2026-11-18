<script lang="ts">
	import { failureMessage, type FileRecord, type PluginHost } from '@maroid/plugin-sdk';
	import { FileUpload } from '@maroid/ui';

	import type { JasmineApi, Moment } from '../../api';
	import { today } from '../../lib/format';
	import { PHOTO_KIND, PHOTO_MAX_SIZE, PHOTO_TYPES } from '../../lib/photo';

	interface Props {
		host: PluginHost;
		api: JasmineApi;
		plantId: string;
		/** Called once the moment and its photos are stored. */
		onsaved: () => void;
	}

	let { host, api, plantId, onsaved }: Props = $props();

	let dialog = $state<HTMLDialogElement>();
	let editing = $state<Moment | null>(null);
	let takenOn = $state(today());
	let note = $state('');
	/** The uploaded files, in the order their bytes landed. */
	let files = $state<FileRecord[]>([]);
	let uploading = $state(false);
	let saving = $state(false);
	let failure = $state<string | null>(null);
	/** Changes at each open, so the upload control starts empty. */
	let session = $state(0);

	const ready = $derived(files.filter((file) => file.state === 'ready').map((file) => file.id));
	const canSave = $derived(
		takenOn !== '' &&
			takenOn <= today() &&
			!uploading &&
			!saving &&
			(editing !== null || ready.length > 0)
	);

	/** Opens the dialog for a new moment, or for the moment to edit. */
	export function open(moment?: Moment): void {
		editing = moment ?? null;
		takenOn = moment?.taken_on ?? today();
		note = moment?.note ?? '';
		files = [];
		failure = null;
		session++;
		dialog?.showModal();
	}

	function keep(file: FileRecord): void {
		const known = files.findIndex((item) => item.id === file.id);
		if (known === -1) {
			files.push(file);
		} else {
			files[known] = file;
		}
	}

	async function write(): Promise<void> {
		const values = { taken_on: takenOn, note: note.trim() || undefined };

		if (editing === null) {
			await api.moments.create(plantId, { ...values, file_ids: ready });

			return;
		}

		await api.moments.update(plantId, editing.id, values);

		if (ready.length > 0) {
			await api.moments.addPhotos(plantId, editing.id, ready);
		}
	}

	async function save(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		saving = true;
		failure = null;

		try {
			await write();
			dialog?.close();
			onsaved();
		} catch (error) {
			failure = failureMessage(error, 'The moment stays as it was. Try again.');
		} finally {
			saving = false;
		}
	}
</script>

<dialog bind:this={dialog} class="modal">
	<form class="modal-box max-w-2xl" onsubmit={save}>
		<h3 class="text-lg font-semibold">{editing ? 'Edit moment' : 'Add a moment'}</h3>
		<p class="meta mt-1">A day in the life of the plant, with a note and its photos.</p>

		<div class="mt-4 flex flex-wrap gap-4">
			<fieldset class="fieldset">
				<legend class="fieldset-legend">Day</legend>
				<input class="input input-sm" type="date" max={today()} required bind:value={takenOn} />
			</fieldset>

			<fieldset class="fieldset grow">
				<legend class="fieldset-legend">Note</legend>
				<textarea
					class="textarea textarea-sm w-full"
					rows="2"
					maxlength="1000"
					placeholder="Repotted, a new leaf, the first flower"
					bind:value={note}
				></textarea>
			</fieldset>
		</div>

		<fieldset class="fieldset mt-2">
			<legend class="fieldset-legend">
				{editing ? `Add photos (the moment holds ${editing.photos.length})` : 'Photos'}
			</legend>
			{#key session}
				<FileUpload
					client={host.files}
					kind={PHOTO_KIND}
					accept={PHOTO_TYPES}
					maxSize={PHOTO_MAX_SIZE}
					label="Drop photos here or choose them"
					onuploaded={keep}
					onsettled={keep}
					bind:busy={uploading}
				/>
			{/key}
		</fieldset>

		{#if failure}
			<div role="alert" class="alert alert-error alert-soft mt-4">
				<span>{failure}</span>
			</div>
		{/if}

		<div class="modal-action">
			<button
				class="btn btn-ghost btn-sm"
				type="button"
				disabled={saving}
				onclick={() => dialog?.close()}
			>
				Cancel
			</button>
			<button class="btn btn-primary btn-sm" type="submit" disabled={!canSave}>
				{#if saving}<span class="loading loading-spinner loading-xs"></span>{/if}
				{editing ? 'Save' : 'Add moment'}
			</button>
		</div>
	</form>
	<form method="dialog" class="modal-backdrop">
		<button>close</button>
	</form>
</dialog>
