<script lang="ts">
	import type { Schema, UiSchemaRoot } from '@sjsf/form';

	import { ApiError, PROBLEM_TYPE, api, type SettingsValues } from '$lib/api';
	import SchemaForm from '$lib/components/settings/SchemaForm.svelte';
	import { toInput, uiSchemaFor } from '$lib/settings/form';
	import { fieldsOf, missingFields, type SettingsField } from '$lib/settings/schema';

	interface Props {
		workspaceId: string;
		pluginId: string;
		readonly?: boolean;
	}

	let { workspaceId, pluginId, readonly = false }: Props = $props();

	let status = $state<'loading' | 'ready' | 'absent' | 'error'>('loading');
	let schema = $state<Schema>({});
	let uiSchema = $state<UiSchemaRoot>({});
	let values = $state<SettingsValues>({});
	let fields = $state<SettingsField[]>([]);
	let missing = $state<string[]>([]);
	let failure = $state<string | null>(null);
	let saving = $state(false);
	let saved = $state(false);
	let revision = $state(0);
	let etag = $state<string | undefined>(undefined);
	let child = $state<ReturnType<typeof SchemaForm>>();

	/** The detail names this occurrence, and the title names the type. */
	function reasonOf(error: unknown, fallback: string): string {
		if (error instanceof ApiError && error.problem !== null) {
			return error.problem.detail ?? error.problem.title;
		}

		return fallback;
	}

	async function load(workspace: string, id: string): Promise<void> {
		try {
			const [document, stored] = await Promise.all([
				api.settings.schema(workspace, id),
				api.settings.read(workspace, id)
			]);

			if (document === null || stored === null) {
				return;
			}

			fields = fieldsOf(document);
			schema = document as Schema;
			values = stored.value;
			etag = stored.etag;
			uiSchema = uiSchemaFor(fields, stored.value);
			missing = missingFields(fields, stored.value);
			revision += 1;
			status = 'ready';
		} catch (error) {
			if (error instanceof ApiError && error.is(PROBLEM_TYPE.settingsAbsent)) {
				status = 'absent';

				return;
			}

			console.error('Failed to load the settings', error);
			status = 'error';
		}
	}

	async function save(value: SettingsValues): Promise<void> {
		saving = true;
		saved = false;
		failure = null;

		try {
			await api.settings.save(workspaceId, pluginId, toInput(fields, value), etag);
			await load(workspaceId, pluginId);
			saved = true;
		} catch (error) {
			if (error instanceof ApiError && error.is(PROBLEM_TYPE.settingsInvalid)) {
				child?.showFieldErrors(error.fields);
				failure = reasonOf(error, 'The settings do not match the schema.');
			} else if (error instanceof ApiError && error.is(PROBLEM_TYPE.preconditionFailed)) {
				await load(workspaceId, pluginId);
				failure = 'These settings changed elsewhere. The form now shows the stored values.';
			} else {
				console.error('Failed to save the settings', error);
				failure = reasonOf(error, 'The hub stored nothing. Try again.');
			}
		} finally {
			saving = false;
		}
	}

	$effect(() => {
		void load(workspaceId, pluginId);
	});
</script>

{#snippet footer()}
	{#if failure}
		<div role="alert" class="alert alert-error alert-soft mt-4">
			<span>{failure}</span>
		</div>
	{/if}

	{#if readonly}
		<p class="border-base-300 meta mt-6 border-t pt-4">
			Your role reads these settings. An editor or a manager changes them.
		</p>
	{:else}
		<div class="border-base-300 mt-6 flex items-center gap-3 border-t pt-4">
			<button type="submit" class="btn btn-primary btn-sm" disabled={saving}>
				{#if saving}
					<span class="loading loading-spinner loading-xs"></span>
				{/if}
				Save
			</button>
			<button
				type="button"
				class="btn btn-ghost btn-sm"
				disabled={saving}
				onclick={() => void load(workspaceId, pluginId)}
			>
				Reset
			</button>
			{#if saved}
				<span class="text-success text-sm">Saved.</span>
			{/if}
		</div>
	{/if}
{/snippet}

{#if status === 'loading'}
	<div class="panel flex max-w-xl flex-col gap-4 p-5">
		{#each [0, 1, 2] as i (i)}
			<div class="flex flex-col gap-2">
				<span class="skeleton h-3 w-28" aria-hidden="true"></span>
				<span class="skeleton h-9 w-full" aria-hidden="true"></span>
			</div>
		{/each}
	</div>
{:else if status === 'absent'}
	<div class="empty-state">This plugin declares no settings.</div>
{:else if status === 'error'}
	<div role="alert" class="alert alert-error alert-soft">
		<span>Failed to load the settings.</span>
		<button type="button" class="btn btn-sm" onclick={() => void load(workspaceId, pluginId)}
			>Retry</button
		>
	</div>
{:else}
	{#if missing.length > 0}
		<div role="alert" class="alert alert-warning alert-soft mb-6">
			<span>
				{missing.length}
				{missing.length === 1 ? 'required field holds' : 'required fields hold'} no value. This plugin
				does nothing until you fill them.
			</span>
		</div>
	{/if}

	<div class="panel max-w-xl p-5">
		{#key revision}
			<SchemaForm
				bind:this={child}
				{schema}
				{uiSchema}
				initialValue={values}
				onsubmit={(value) => void save(value)}
				{footer}
				disabled={readonly}
			/>
		{/key}
	</div>
{/if}
