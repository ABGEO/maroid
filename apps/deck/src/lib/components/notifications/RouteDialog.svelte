<script lang="ts">
	import type { Schema, UiSchemaRoot } from '@sjsf/form';

	import {
		ApiError,
		PROBLEM_TYPE,
		api,
		createWriteIntent,
		type NotificationProvider,
		type NotificationProviderKind,
		type NotificationRoute,
		type NotificationTopic,
		type SettingsValues
	} from '$lib/api';
	import SchemaForm from '$lib/components/settings/SchemaForm.svelte';
	import { failuresUnder } from '$lib/notifications/form';
	import { toInput, uiSchemaFor } from '$lib/settings/form';
	import { fieldsOf } from '$lib/settings/schema';

	interface Props {
		workspaceId: string;
		kinds: NotificationProviderKind[];
		providers: NotificationProvider[];
		onsaved: () => void;
	}

	let { workspaceId, kinds, providers, onsaved }: Props = $props();

	let dialog: HTMLDialogElement | undefined = $state();
	let child = $state<ReturnType<typeof SchemaForm>>();
	let topic = $state<NotificationTopic | null>(null);
	let editing = $state<NotificationRoute | null>(null);
	let providerId = $state('');
	let values = $state<SettingsValues>({});
	let revision = $state(0);
	let saving = $state(false);
	let failure = $state<string | null>(null);

	const intent = createWriteIntent();

	const provider = $derived(providers.find((one) => one.id === providerId));
	const kind = $derived(kinds.find((one) => one.id === provider?.kind));
	const fields = $derived(kind === undefined ? [] : fieldsOf(kind.destination_schema));
	const schema = $derived((kind?.destination_schema ?? {}) as Schema);
	const uiSchema = $derived<UiSchemaRoot>(uiSchemaFor(fields, values));

	/** Opens the dialog for a new route of the topic, or for a change of the given route. */
	export function open(of: NotificationTopic, route?: NotificationRoute): void {
		topic = of;
		editing = route ?? null;
		providerId = route?.provider_id ?? providers[0]?.id ?? '';
		values = route?.destination ?? {};
		failure = null;
		revision += 1;
		dialog?.showModal();
	}

	function close(): void {
		dialog?.close();
	}

	function reasonOf(error: unknown): string {
		if (!(error instanceof ApiError) || error.problem === null) {
			return 'The hub stored nothing. Try again.';
		}

		if (error.is(PROBLEM_TYPE.notificationRouteExists)) {
			return 'Another route already sends this topic through this provider to this destination.';
		}

		if (error.is(PROBLEM_TYPE.validationFailed)) {
			return error.problem.errors?.[0]?.detail ?? error.problem.title;
		}

		return error.problem.detail ?? error.problem.title;
	}

	async function save(value: SettingsValues): Promise<void> {
		if (saving || topic === null) {
			return;
		}

		saving = true;
		failure = null;

		try {
			const destination = toInput(fields, value);

			if (editing === null) {
				const body = {
					plugin_id: topic.plugin_id,
					topic_id: topic.topic_id,
					provider_id: providerId,
					destination
				};
				await api.notifications.createRoute(workspaceId, body, intent.keyFor(body));
				intent.settle();
			} else {
				await api.notifications.changeRoute(workspaceId, editing.id, { destination });
			}

			close();
			onsaved();
		} catch (error) {
			if (error instanceof ApiError && error.is(PROBLEM_TYPE.settingsInvalid)) {
				child?.showFieldErrors(failuresUnder(error.fields, 'destination'));
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
			{editing === null ? 'Add route' : 'Save'}
		</button>
	</div>
{/snippet}

<dialog bind:this={dialog} class="modal">
	<div class="modal-box">
		<h3 class="text-lg font-semibold">
			{editing === null ? 'Route' : 'Change the route of'}
			{topic?.name ?? ''}
		</h3>
		<p class="meta mt-1">The provider delivers this topic to the destination below.</p>

		<label class="fieldset mt-4">
			<span class="fieldset-legend">Provider</span>
			<select
				class="select select-sm w-full"
				bind:value={providerId}
				disabled={saving || editing !== null}
			>
				{#each providers as one (one.id)}
					<option value={one.id}>{one.name}{one.enabled ? '' : ' (disabled)'}</option>
				{/each}
			</select>
		</label>

		{#key `${revision}-${providerId}`}
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
