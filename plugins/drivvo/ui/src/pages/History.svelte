<script lang="ts">
	import type { PluginHost, RouteParams } from '@maroid/plugin-sdk';

	import type { Kind, Period } from '../api';
	import EntryTable from '../components/EntryTable.svelte';
	import PeriodPicker from '../components/PeriodPicker.svelte';
	import VehicleShell from '../components/VehicleShell.svelte';
	import { KINDS, kindLabel } from '../lib/format';
	import { periodOf, type PeriodChoice } from '../lib/period';
	import { createFleet } from '../lib/vehicles.svelte';

	let { host, params }: { host: PluginHost; params: RouteParams } = $props();

	const fleet = createFleet(
		() => host,
		() => params.vehicle
	);

	let choice = $state<PeriodChoice>('all');
	let range = $state<Period>({});
	let shown = $state<Kind[]>([...KINDS]);
	const period = $derived(periodOf(choice, host.timeZone, range));

	function toggle(kind: Kind): void {
		shown = shown.includes(kind)
			? shown.filter((held) => held !== kind)
			: KINDS.filter((held) => held === kind || shown.includes(held));
	}
</script>

<VehicleShell
	{host}
	{fleet}
	page="history"
	title="History"
	lead="Every log entry of the vehicle, the latest first."
>
	{#snippet actions()}
		<PeriodPicker bind:choice bind:range />
	{/snippet}

	{#snippet children(vehicle)}
		<div class="join mb-4 flex-wrap" role="group" aria-label="Kinds">
			{#each KINDS as kind (kind)}
				{@const on = shown.includes(kind)}
				<button
					type="button"
					class={['join-item btn btn-sm', on && 'btn-active']}
					aria-pressed={on}
					disabled={on && shown.length === 1}
					onclick={() => toggle(kind)}
				>
					{kindLabel(kind)}
				</button>
			{/each}
		</div>

		{#key `${vehicle.id} ${period.from} ${period.to} ${shown.join()}`}
			<EntryTable {host} {vehicle} {period} kinds={shown} page="history" />
		{/key}
	{/snippet}
</VehicleShell>
