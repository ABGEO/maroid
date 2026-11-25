<script lang="ts">
	import type { PluginHost, RouteParams } from '@maroid/plugin-sdk';

	import { createCarLogApi, type Money, type Overview, type Period, type Units } from '../api';
	import PeriodPicker from '../components/PeriodPicker.svelte';
	import SpendChart from '../components/SpendChart.svelte';
	import VehicleShell from '../components/VehicleShell.svelte';
	import { consumption, distance, kindLabel, money, moneyPerDistance } from '../lib/format';
	import { periodOf, type PeriodChoice } from '../lib/period';
	import { createReader } from '../lib/reader.svelte';
	import { createFleet } from '../lib/vehicles.svelte';

	let { host, params }: { host: PluginHost; params: RouteParams } = $props();

	const api = $derived(createCarLogApi(host));
	const fleet = createFleet(
		() => host,
		() => params.vehicle
	);
	const overview = createReader<Overview>();

	let choice = $state<PeriodChoice>('last12');
	let range = $state<Period>({});
	const period = $derived(periodOf(choice, host.timeZone, range));

	function load(): void {
		const vehicle = fleet.vehicle;
		const asked = period;
		if (vehicle !== undefined) {
			void overview.load(() => api.overview(vehicle.id, asked));
		}
	}

	$effect(load);

	function perDistance(value: Money | null, units: Units): string {
		return value === null ? 'None' : moneyPerDistance(value, units);
	}
</script>

<VehicleShell
	{host}
	{fleet}
	page="overview"
	title="Overview"
	lead="The spend, the income, the distance, and the consumption of the vehicle."
>
	{#snippet actions()}
		<PeriodPicker bind:choice bind:range />
	{/snippet}

	{#snippet children(vehicle)}
		{@const units = vehicle.units}
		{@const read = overview.value}
		{#if overview.status === 'loading' && read === null}
			<div class="flex flex-col gap-6" aria-hidden="true">
				<span class="skeleton h-40 w-full"></span>
				<span class="skeleton h-80 w-full"></span>
			</div>
		{:else if overview.status === 'error'}
			<div role="alert" class="alert alert-error alert-soft max-w-xl">
				<span>Failed to load the overview.</span>
				<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
			</div>
		{:else if read !== null}
			<div
				class={[
					'flex flex-col gap-6 transition-opacity',
					overview.status === 'loading' && 'opacity-60'
				]}
			>
				<dl class="panel grid gap-4 p-4 sm:grid-cols-3">
					<div>
						<dt class="meta">Spend</dt>
						<dd class="text-xl font-semibold tabular-nums">{money(read.spend.total)}</dd>
						<dd class="meta tabular-nums">
							Fuel {money(read.spend.fuel)}, maintenance {money(read.spend.maintenance)}, expenses {money(
								read.spend.expense
							)}
						</dd>
					</div>
					<div>
						<dt class="meta">Income</dt>
						<dd class="text-xl font-semibold tabular-nums">{money(read.income)}</dd>
					</div>
					<div>
						<dt class="meta">Distance</dt>
						<dd class="text-xl font-semibold tabular-nums">
							{read.distance === null ? 'None' : distance(read.distance, units)}
						</dd>
					</div>
					<div>
						<dt class="meta">Fuel per distance</dt>
						<dd class="text-xl font-semibold tabular-nums">
							{perDistance(read.cost_per_distance.fuel, units)}
						</dd>
					</div>
					<div>
						<dt class="meta">Spend per distance</dt>
						<dd class="text-xl font-semibold tabular-nums">
							{perDistance(read.cost_per_distance.total, units)}
						</dd>
					</div>
					<div>
						<dt class="meta">Consumption</dt>
						<dd class="text-xl font-semibold tabular-nums">
							{read.consumption.value === null
								? 'None'
								: consumption(read.consumption.value, units)}
						</dd>
					</div>
				</dl>

				<section class="panel p-4" aria-labelledby="months-title">
					<h2 id="months-title" class="mb-2 font-medium">Each month</h2>
					{#if read.months.length === 0}
						<p class="meta">No log entry in the period.</p>
					{:else}
						<SpendChart months={read.months} />
					{/if}
				</section>

				<section class="panel p-4" aria-labelledby="types-title">
					<h2 id="types-title" class="mb-2 font-medium">Each type</h2>
					{#if read.types.length === 0}
						<p class="meta">No maintenance and no expense in the period.</p>
					{:else}
						<div class="overflow-x-auto">
							<table class="table-sm table">
								<thead>
									<tr>
										<th>Type</th>
										<th>Kind</th>
										<th class="text-right">Spend</th>
									</tr>
								</thead>
								<tbody>
									{#each read.types as spend (`${spend.kind}-${spend.name}`)}
										<tr>
											<td>{spend.name}</td>
											<td class="meta">{kindLabel(spend.kind)}</td>
											<td class="text-right whitespace-nowrap tabular-nums">
												{money(spend.amount)}
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					{/if}
				</section>
			</div>
		{/if}
	{/snippet}
</VehicleShell>
