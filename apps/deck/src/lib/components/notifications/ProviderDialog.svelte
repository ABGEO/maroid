<script lang="ts">
	import type { Schema, UiSchemaRoot } from '@sjsf/form';

	import {
		ApiError,
		PROBLEM_TYPE,
		api,
		createWriteIntent,
		type NotificationProvider,
		type NotificationProviderKind,
		type SettingsValues
	} from '$lib/api';
	import SchemaForm from '$lib/components/settings/SchemaForm.svelte';
	import { failuresUnder } from '$lib/notifications/form';
	import { toInput, uiSchemaFor } from '$lib/settings/form';
	import { fieldsOf } from '$lib/settings/schema';

	interface Props {
		workspaceId: string;
		kinds: NotificationProviderKind[];
		onsaved: () => void;
	}

	let { workspaceId, kinds, onsaved }: Props = $props();

	const MAX_NAME = 64;

	let dialog: HTMLDialogElement | undefined = $state();
	let child = $state<ReturnType<typeof SchemaForm>>();
	let editing = $state<NotificationProvider | null>(null);
	let kindId = $state('');
	let name = $state('');
	let values = $state<SettingsValues>({});
	let revision = $state(0);
	let saving = $state(false);
	let nameError = $state<string | null>(null);
	let failure = $state<string | null>(null);

	const intent = createWriteIntent();

	const kind = $derived(kinds.find((one) => one.id === kindId));
	const fields = $derived(kind === undefined ? [] : fieldsOf(kind.provider_schema));
	const schema = $derived((kind?.provider_schema ?? {}) as Schema);
	const uiSchema = $derived<UiSchemaRoot>(uiSchemaFor(fields, values));
	const nameLength = $derived([...name.trim()].length);

	/** Opens the dialog for a new provider, or for a change of the given one. */
	export function open(provider?: NotificationProvider): void {
		editing = provider ?? null;
		kindId = provider?.kind ?? kinds[0]?.id ?? '';
		name = provider?.name ?? '';
		values = provider?.fields ?? {};
		nameError = null;
		failure = null;
		revision += 1;
		dialog?.showModal();
	}

	function close(): void {
		dialog?.close();
	}

	function reasonOf(error: unknown): string {
		if (!(error instanceof ApiError) || error.problem === null) {
			return 'Saving failed. Try again.';
		}

		if (error.is(PROBLEM_TYPE.deliveryRefused)) {
			return `${kind?.name ?? 'The service'} refused the credential: ${error.problem.detail ?? ''}`;
		}

		if (error.is(PROBLEM_TYPE.messagingUnreachable)) {
			return `${kind?.name ?? 'The service'} did not answer. Nothing is stored. Try again later.`;
		}

		if (error.is(PROBLEM_TYPE.preconditionFailed)) {
			return 'Someone changed this provider since you opened it. Close the dialog and try again.';
		}

		return error.problem.detail ?? error.problem.title;
	}

	async function save(value: SettingsValues): Promise<void> {
		if (saving) {
			return;
		}

		if (nameLength < 1 || nameLength > MAX_NAME) {
			nameError = `A name holds 1 to ${MAX_NAME} characters.`;

			return;
		}

		saving = true;
		nameError = null;
		failure = null;

		try {
			const input = toInput(fields, value);

			if (editing === null) {
				const body = { kind: kindId, name: name.trim(), fields: input };
				await api.notifications.createProvider(workspaceId, body, intent.keyFor(body));
				intent.settle();
			} else {
				await api.notifications.changeProvider(workspaceId, editing.id, {
					name: name.trim(),
					fields: input
				});
			}

			close();
			onsaved();
		} catch (error) {
			if (error instanceof ApiError && error.is(PROBLEM_TYPE.settingsInvalid)) {
				child?.showFieldErrors(failuresUnder(error.fields, 'fields'));
			}

			if (error instanceof ApiError && error.is(PROBLEM_TYPE.validationFailed)) {
				nameError = error.fields.find((one) => one.pointer === '/name')?.detail ?? null;
			}

			failure = reasonOf(error);
		} finally {
			saving = false;
		}
	}
</script>

{#snippet footer()}
	{#if failure}
		<div role="alert" class="alert alert-error alert-soft mt-4">
			<span>{failure}</span>
		</div>
	{/if}

	<div class="modal-action">
		<button type="button" class="btn btn-ghost btn-sm" disabled={saving} onclick={close}>
			Cancel
		</button>
		<button type="submit" class="btn btn-primary btn-sm" disabled={saving || kind === undefined}>
			{#if saving}
				<span class="loading loading-spinner loading-xs"></span>
			{/if}
			{editing === null ? 'Add' : 'Save'}
		</button>
	</div>
{/snippet}

<dialog bind:this={dialog} class="modal">
	<div class="modal-box">
		<h3 class="text-lg font-semibold">
			{editing === null ? 'Add a notification provider' : `Change ${editing.name}`}
		</h3>
		<p class="meta mt-1">
			Maroid checks the credential with the service before it stores it. Once stored, a secret is
			never shown again.
		</p>

		<div class="mt-4 flex flex-col gap-3">
			{#if editing === null}
				<label class="fieldset">
					<span class="fieldset-legend">Service</span>
					<select class="select select-sm w-full" bind:value={kindId} disabled={saving}>
						{#each kinds as one (one.id)}
							<option value={one.id}>{one.name}</option>
						{/each}
					</select>
				</label>
			{/if}

			<label class="fieldset">
				<span class="fieldset-legend">Name</span>
				<input
					class={['input input-sm w-full', nameError && 'input-error']}
					bind:value={name}
					maxlength={MAX_NAME}
					placeholder="Family bot"
					disabled={saving}
				/>
				{#if nameError}
					<span class="text-error text-xs">{nameError}</span>
				{/if}
			</label>
		</div>

		{#key `${revision}-${kindId}`}
			<SchemaForm
				bind:this={child}
				{schema}
				{uiSchema}
				initialValue={values}
				onsubmit={(value) => void save(value)}
				{footer}
				disabled={saving}
			/>
		{/key}
	</div>
	<form method="dialog" class="modal-backdrop">
		<button>close</button>
	</form>
</dialog>
