<script lang="ts">
	import type { PluginHost } from '@maroid/plugin-sdk';

	import '../../app.css';
	import {
		createJasmineApi,
		type MaintenanceEntryFilter,
		type MaintenanceType,
		type Plant
	} from '../../api';
	import { MAINTENANCE_TYPES } from '../../lib/maintenance';
	import { readAll } from '../../lib/pages';
	import EntryLog from './EntryLog.svelte';

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createJasmineApi(host));
	const canWrite = $derived(host.can('maintenance.write'));

	let plants = $state<Plant[]>([]);
	let plantId = $state('');
	let type = $state<MaintenanceType | ''>('');

	const filter = $derived<MaintenanceEntryFilter>({
		plant_id: plantId === '' ? undefined : plantId,
		type: type === '' ? undefined : type
	});

	$effect(() => {
		// An inactive plant keeps its history, so the filter offers it too.
		Promise.all([
			readAll((link?: string) => api.plants.list({ status: 'active' }, link)),
			readAll((link?: string) => api.plants.list({ status: 'inactive' }, link))
		])
			.then(([active, inactive]) => {
				plants = [...active, ...inactive].sort(
					(a, b) => Number(a.display_id.slice(1)) - Number(b.display_id.slice(1))
				);
			})
			.catch((error) => console.error('Failed to load the plants', error));
	});
</script>

<div class="max-w-5xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">Maintenance log</h1>
			<p class="page-lead">Every care action of this workspace, newest first.</p>
		</div>
	</header>

	<div class="mb-4 flex flex-wrap gap-2">
		<select class="select select-sm w-auto" aria-label="Plant" bind:value={plantId}>
			<option value="">Every plant</option>
			{#each plants as plant (plant.id)}
				<option value={plant.id}>
					{plant.display_id}
					{plant.common_name}{plant.status === 'inactive' ? ' (inactive)' : ''}
				</option>
			{/each}
		</select>
		<select class="select select-sm w-auto" aria-label="Care" bind:value={type}>
			<option value="">Every care</option>
			{#each MAINTENANCE_TYPES as option (option.value)}
				<option value={option.value}>{option.label}</option>
			{/each}
		</select>
	</div>

	{#key `${plantId}|${type}`}
		<EntryLog {api} {filter} {canWrite} />
	{/key}
</div>
