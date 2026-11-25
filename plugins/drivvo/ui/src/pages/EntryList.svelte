<script lang="ts">
	import type { PluginHost, RouteParams } from '@maroid/plugin-sdk';

	import { createCarLogApi, type Kind, type LogEntryTotals, type Period } from '../api';
	import EntryTable from '../components/EntryTable.svelte';
	import PeriodPicker from '../components/PeriodPicker.svelte';
	import VehicleShell from '../components/VehicleShell.svelte';
	import { money, volume } from '../lib/format';
	import { periodOf, type PeriodChoice } from '../lib/period';
	import { createReader } from '../lib/reader.svelte';
	import { createFleet } from '../lib/vehicles.svelte';

	type ListKind = Exclude<Kind, 'odometer'>;

	const PAGES: Record<ListKind, { page: string; title: string; lead: string }> = {
		refuel: {
			page: 'refuels',
			title: 'Refuels',
			lead: 'The refuels of the vehicle, the latest first.'
		},
		maintenance: {
			page: 'maintenances',
			title: 'Maintenance',
			lead: 'The maintenances of the vehicle, the latest first.'
		},
		expense: {
			page: 'expenses',
			title: 'Expenses',
			lead: 'The expenses of the vehicle, the latest first.'
		},
		income: {
			page: 'incomes',
			title: 'Income',
			lead: 'The incomes of the vehicle, the latest first.'
		}
	};

	let { host, params, kind }: { host: PluginHost; params: RouteParams; kind: ListKind } = $props();

	const api = $derived(createCarLogApi(host));
	const about = $derived(PAGES[kind]);
	const fleet = createFleet(
		() => host,
		() => params.vehicle
	);
	const totals = createReader<LogEntryTotals>();

	let choice = $state<PeriodChoice>('all');
	let range = $state<Period>({});
	const period = $derived(periodOf(choice, host.timeZone, range));

	function loadTotals(): void {
		const vehicle = fleet.vehicle;
		const asked = period;
		if (vehicle !== undefined) {
			void totals.load(() => api.logEntryTotals(vehicle.id, asked, kind));
		}
	}

	$effect(loadTotals);
</script>

<VehicleShell {host} {fleet} page={about.page} title={about.title} lead={about.lead}>
	{#snippet actions()}
		<PeriodPicker bind:choice bind:range />
	{/snippet}

	{#snippet children(vehicle)}
		<div class="flex flex-col gap-6">
			{#if totals.status === 'error'}
				<div role="alert" class="alert alert-error alert-soft max-w-xl">
					<span>Failed to load the totals.</span>
					<button type="button" class="btn btn-sm" onclick={loadTotals}>Retry</button>
				</div>
			{:else if totals.value === null}
				<span class="skeleton h-20 w-full" aria-hidden="true"></span>
			{:else}
				{@const read = totals.value}
				<dl
					class={[
						'panel grid gap-4 p-4 transition-opacity sm:grid-cols-3',
						totals.status === 'loading' && 'opacity-60'
					]}
				>
					<div>
						<dt class="meta">Log entries</dt>
						<dd class="text-xl font-semibold tabular-nums">{read.count}</dd>
					</div>
					<div>
						<dt class="meta">Amount</dt>
						<dd class="text-xl font-semibold tabular-nums">{money(read.amount)}</dd>
					</div>
					{#if read.volume !== undefined}
						<div>
							<dt class="meta">Volume</dt>
							<dd class="text-xl font-semibold tabular-nums">
								{volume(read.volume, vehicle.units)}
							</dd>
						</div>
					{/if}
				</dl>
			{/if}

			<section class="min-w-0" aria-label={about.title}>
				{#key `${vehicle.id} ${period.from} ${period.to}`}
					<EntryTable {host} {vehicle} {period} kinds={[kind]} list={kind} page={about.page} />
				{/key}
			</section>
		</div>
	{/snippet}
</VehicleShell>
