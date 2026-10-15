<script lang="ts">
	import { createWriteIntent, failureMessage, type PluginHost } from '@maroid/plugin-sdk';
	import { SvelteMap } from 'svelte/reactivity';

	import '../../app.css';
	import {
		createJasmineApi,
		type EnvironmentReference,
		type Plant,
		type PlantFilter,
		type PlantStatus,
		type PlantUpdatePayload
	} from '../../api';
	import { readAll } from '../../lib/pages';
	import LogForm from './LogForm.svelte';
	import PlantForm from './PlantForm.svelte';
	import PlantList from './PlantList.svelte';

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createJasmineApi(host));
	const canAdd = $derived(host.can('plants.write'));
	const canLog = $derived(host.can('maintenance.write'));
	const intent = createWriteIntent();

	let environments = $state<EnvironmentReference[]>([]);
	let environmentId = $state('');
	let status = $state<PlantStatus>('active');
	let adding = $state(false);
	let busy = $state(false);
	let failure = $state<string | null>(null);
	let revision = $state(0);

	const selected = new SvelteMap<string, Plant>();
	const filter = $derived<PlantFilter>({
		environment_id: environmentId === '' ? undefined : environmentId,
		status
	});

	function toggle(plant: Plant): void {
		if (selected.has(plant.id)) {
			selected.delete(plant.id);
		} else {
			selected.set(plant.id, plant);
		}
	}

	async function add(payload: PlantUpdatePayload): Promise<void> {
		busy = true;
		failure = null;

		const { status: _, ...created } = payload;

		try {
			const plant = await api.plants.create(created, { idempotencyKey: intent.keyFor(created) });
			if (plant === null) {
				return;
			}

			intent.settle();
			adding = false;
			revision++;
		} catch (error) {
			failure = failureMessage(error, 'The plant was not added. Try again.');
		} finally {
			busy = false;
		}
	}

	function recorded(): void {
		selected.clear();
		revision++;
	}

	$effect(() => {
		readAll((link?: string) => api.environments.list(link))
			.then((items) => (environments = items))
			.catch((error) => console.error('Failed to load the environments', error));
	});
</script>

<div class="max-w-5xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">Plants</h1>
			<p class="page-lead">The plants of this workspace. Open a plant for its care and history.</p>
		</div>
		{#if canAdd && !adding}
			<button
				class="btn btn-primary btn-sm"
				type="button"
				disabled={environments.length === 0}
				title={environments.length === 0 ? 'Add an environment first.' : undefined}
				onclick={() => (adding = true)}
			>
				Add plant
			</button>
		{/if}
	</header>

	{#if failure}
		<div role="alert" class="alert alert-error alert-soft mb-4">
			<span>{failure}</span>
		</div>
	{/if}

	{#if adding}
		<div class="mb-6">
			<PlantForm {environments} {busy} onsave={add} oncancel={() => (adding = false)} />
		</div>
	{/if}

	<div class="mb-4 flex flex-wrap gap-2">
		<select class="select select-sm w-auto" aria-label="Environment" bind:value={environmentId}>
			<option value="">Every environment</option>
			{#each environments as environment (environment.id)}
				<option value={environment.id}>{environment.display_id} {environment.name}</option>
			{/each}
		</select>
		<select class="select select-sm w-auto" aria-label="Status" bind:value={status}>
			<option value="active">Active</option>
			<option value="inactive">Inactive</option>
		</select>
	</div>

	{#if canLog && selected.size > 0}
		<div class="mb-4">
			<LogForm {api} plants={[...selected.values()]} onrecorded={recorded} />
			<button class="btn btn-ghost btn-xs mt-1" type="button" onclick={() => selected.clear()}>
				Clear the selection
			</button>
		</div>
	{/if}

	{#key `${environmentId}|${status}|${revision}`}
		<PlantList
			{api}
			{filter}
			{selected}
			selectable={canLog}
			hrefOf={(plant) => host.href(`/plants/${plant.id}`)}
			onopen={(plant) => host.navigate(`/plants/${plant.id}`)}
			ontoggle={toggle}
		/>
	{/key}
</div>
