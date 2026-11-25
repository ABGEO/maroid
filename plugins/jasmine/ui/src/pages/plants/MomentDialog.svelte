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
	let takenOn = $state('');
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
			takenOn <= today(host.timeZone) &&
			!uploading &&
			!saving &&
			(editing !== null || ready.length > 0)
	);

	/** Opens the dialog for a new moment, or for the moment to edit. */
	export function open(moment?: Moment): void {
		editing = moment ?? null;
		takenOn = moment?.taken_on ?? today(host.timeZone);
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

	/**
	 * Writes the moment and answers the reason of a failure, or null. An edit writes the
	 * day and the note before the photos, so a failed second write says that the first
	 * one landed.
	 */
	async function write(): Promise<string | null> {
		const values = { taken_on: takenOn, note: note.trim() || undefined };

		try {
			if (editing === null) {
				await api.moments.create(plantId, { ...values, file_ids: ready });
			} else {
				await api.moments.update(plantId, editing.id, values);
			}
		} catch (error) {
			return failureMessage(error, 'The moment stays as it was. Try again.');
		}

		if (editing === null || ready.length === 0) {
			return null;
		}

		try {
			await api.moments.addPhotos(plantId, editing.id, ready);
		} catch (error) {
			return `The day and the note are saved. ${failureMessage(error, 'The photos did not join. Try again.')}`;
		}

		return null;
	}

	async function save(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		saving = true;
		failure = await write();
		saving = false;

		if (failure === null) {
			dialog?.close();
			onsaved();
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
				<input
					class="input input-sm"
					type="date"
					max={today(host.timeZone)}
					required
					bind:value={takenOn}
				/>
			</fieldset>

			<fieldset class="fieldset grow">
				<legend class="fieldset-legend">Note</legend>
				<textarea
					class="textarea textarea-sm w-full"
					rows="2"
					maxlength="1000"
					placeholder="Repotted, a new leaf, the first flower"
					bind:value={note}></textarea>
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
