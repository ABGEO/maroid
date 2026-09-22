<script lang="ts">
	import type { Schema, UiSchemaRoot } from '@sjsf/form';

	import { ApiError, PROBLEM_TYPE, api, type SettingsValues } from '$lib/api';
	import SchemaForm from '$lib/components/settings/SchemaForm.svelte';
	import { toInput, uiSchemaFor } from '$lib/settings/form';
	import { fieldsOf, missingFields, type SettingsField } from '$lib/settings/schema';

	interface Props {
		pluginId: string;
	}

	let { pluginId }: Props = $props();

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
	let child = $state<ReturnType<typeof SchemaForm>>();

	/** The detail names this occurrence, and the title names the type. */
	function reasonOf(error: unknown, fallback: string): string {
		if (error instanceof ApiError && error.problem !== null) {
			return error.problem.detail ?? error.problem.title;
		}

		return fallback;
	}

	async function load(id: string): Promise<void> {
		try {
			const [document, stored] = await Promise.all([
				api.settings.schema(id),
				api.settings.read(id)
			]);

			if (document === null || stored === null) {
				return;
			}

			fields = fieldsOf(document);
			schema = document as Schema;
			values = stored;
			uiSchema = uiSchemaFor(fields, stored);
			missing = missingFields(fields, stored);
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
			await api.settings.save(pluginId, toInput(fields, value));
			await load(pluginId);
			saved = true;
		} catch (error) {
			if (error instanceof ApiError && error.is(PROBLEM_TYPE.settingsInvalid)) {
				child?.showFieldErrors(error.fields);
				failure = reasonOf(error, 'The settings do not match the schema.');
			} else {
				console.error('Failed to save the settings', error);
				failure = reasonOf(error, 'The hub stored nothing. Try again.');
			}
		} finally {
			saving = false;
		}
	}

	$effect(() => {
		void load(pluginId);
	});
</script>

{#snippet footer()}
	{#if failure}
		<div class="alert alert-error mt-4">
			<span>{failure}</span>
		</div>
	{/if}

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
			onclick={() => void load(pluginId)}
		>
			Reset
		</button>
		{#if saved}
			<span class="text-success font-mono text-[11px]">Saved.</span>
		{/if}
	</div>
{/snippet}

{#if status === 'loading'}
	<div class="flex flex-col gap-4">
		{#each [0, 1, 2] as i (i)}
			<div class="flex flex-col gap-2">
				<span class="skeleton h-3 w-28" aria-hidden="true"></span>
				<span class="skeleton h-9 w-full" aria-hidden="true"></span>
			</div>
		{/each}
	</div>
{:else if status === 'absent'}
	<div class="alert alert-warning">
		<span>This plugin declares no settings.</span>
	</div>
{:else if status === 'error'}
	<div class="alert alert-error">
		<span>Failed to load the settings.</span>
		<button type="button" class="btn btn-sm" onclick={() => void load(pluginId)}>Retry</button>
	</div>
{:else}
	{#if missing.length > 0}
		<div class="alert alert-warning mb-6">
			<span>
				{missing.length}
				{missing.length === 1 ? 'required field holds' : 'required fields hold'} no value. This plugin
				does nothing until you fill them.
			</span>
		</div>
	{/if}

	<div class="max-w-xl">
		{#key revision}
			<SchemaForm
				bind:this={child}
				{schema}
				{uiSchema}
				initialValue={values}
				onsubmit={(value) => void save(value)}
				{footer}
			/>
		{/key}
	</div>
{/if}
