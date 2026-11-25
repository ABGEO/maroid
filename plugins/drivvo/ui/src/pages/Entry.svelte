<script lang="ts">
	import { formatMoment, type PluginHost, type RouteParams } from '@maroid/plugin-sdk';

	import { createCarLogApi, type Kind, type LogEntry } from '../api';
	import VehicleShell from '../components/VehicleShell.svelte';
	import { amount, day, distance, kindLabel, money, volume } from '../lib/format';
	import { isProblem, LOG_ENTRY_NOT_FOUND } from '../lib/problems';
	import { createReader } from '../lib/reader.svelte';
	import { createFleet } from '../lib/vehicles.svelte';

	const LIST_PAGES: Record<Kind, { page: string; label: string }> = {
		refuel: { page: 'refuels', label: 'Refuels' },
		maintenance: { page: 'maintenances', label: 'Maintenance' },
		expense: { page: 'expenses', label: 'Expenses' },
		income: { page: 'incomes', label: 'Income' },
		odometer: { page: 'history', label: 'History' }
	};

	let { host, params }: { host: PluginHost; params: RouteParams } = $props();

	const api = $derived(createCarLogApi(host));
	const fleet = createFleet(
		() => host,
		() => params.vehicle
	);
	const entry = createReader<LogEntry>();

	const list = $derived(LIST_PAGES[entry.value?.kind ?? 'odometer']);
	const title = $derived(
		entry.value === null
			? 'Log entry'
			: `${kindLabel(entry.value.kind)}, ${formatMoment(entry.value.occurred_at, host.timeZone, { dateStyle: 'medium' })}`
	);
	const notFound = $derived(
		entry.status === 'error' && isProblem(entry.error, LOG_ENTRY_NOT_FOUND)
	);

	function load(): void {
		const vehicle = fleet.vehicle;
		const entryId = params.entry;
		if (vehicle !== undefined) {
			void entry.load(() => api.logEntry(vehicle.id, entryId));
		}
	}

	$effect(load);

	function listHref(vehicleId: string): string {
		return host.href(`/vehicles/${encodeURIComponent(vehicleId)}/${list.page}`);
	}

	function percent(value: number | undefined): string {
		return value === undefined ? '' : `${value}%`;
	}
</script>

<VehicleShell {host} {fleet} page={list.page} {title}>
	{#snippet children(vehicle)}
		{@const units = vehicle.units}
		{@const read = entry.value}
		{#if notFound}
			<div role="alert" class="alert alert-info alert-soft max-w-xl">
				<span>This log entry is not in Maroid.</span>
				<a class="btn btn-sm" href={listHref(vehicle.id)}>Open the history</a>
			</div>
		{:else if entry.status === 'error'}
			<div role="alert" class="alert alert-error alert-soft max-w-xl">
				<span>Failed to load the log entry.</span>
				<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
			</div>
		{:else if read === null}
			<span class="skeleton h-48 w-full max-w-3xl" aria-hidden="true"></span>
		{:else}
			<div class="flex max-w-3xl flex-col gap-6">
				<div>
					<a class="link link-hover text-sm" href={listHref(vehicle.id)}>
						Back to {list.label.toLowerCase()}
					</a>
				</div>

				<dl class="panel grid gap-4 p-4 sm:grid-cols-2">
					<div>
						<dt class="meta">Time</dt>
						<dd>
							{formatMoment(read.occurred_at, host.timeZone, {
								dateStyle: 'medium',
								timeStyle: 'short'
							})}
						</dd>
					</div>
					<div>
						<dt class="meta">Odometer</dt>
						<dd class="tabular-nums">{distance(read.odometer, units)}</dd>
					</div>
					{#if read.amount}
						<div>
							<dt class="meta">Amount</dt>
							<dd class="font-semibold tabular-nums">{money(read.amount)}</dd>
						</div>
					{/if}
					{#each [['Title', read.title], ['Income type', read.income_type], ['Station', read.station], ['Place', read.place], ['Payment method', read.payment_method], ['Reason', read.reason]] as [label, value] (label)}
						{#if value}
							<div>
								<dt class="meta">{label}</dt>
								<dd>{value}</dd>
							</div>
						{/if}
					{/each}
					{#if read.missed_refuel !== undefined}
						<div>
							<dt class="meta">Missed refuel</dt>
							<dd>{read.missed_refuel ? 'Yes' : 'No'}</dd>
						</div>
					{/if}
					{#if read.battery_start_pct !== undefined || read.battery_end_pct !== undefined}
						<div>
							<dt class="meta">Battery charge</dt>
							<dd class="tabular-nums">
								{percent(read.battery_start_pct)} to {percent(read.battery_end_pct)}
							</dd>
						</div>
					{/if}
					{#if read.charge_minutes !== undefined}
						<div>
							<dt class="meta">Charge time</dt>
							<dd class="tabular-nums">{read.charge_minutes} min</dd>
						</div>
					{/if}
					{#if read.note}
						<div class="sm:col-span-2">
							<dt class="meta">Note</dt>
							<dd class="whitespace-pre-line">{read.note}</dd>
						</div>
					{/if}
				</dl>

				{#if read.lines && read.lines.length > 0}
					{@const fuel = read.kind === 'refuel'}
					<section class="panel overflow-x-auto" aria-labelledby="lines-title">
						<h2 id="lines-title" class="px-4 pt-4 font-medium">{fuel ? 'Fuels' : 'Lines'}</h2>
						<table class="table">
							<thead>
								<tr>
									<th>{fuel ? 'Fuel' : 'Type'}</th>
									{#if fuel}
										<th class="text-right">Volume</th>
										<th class="text-right">Unit price</th>
										<th>Full tank</th>
									{/if}
									<th class="text-right">Amount</th>
								</tr>
							</thead>
							<tbody>
								{#each read.lines as line (line.position)}
									<tr>
										<td>
											{line.item ?? ''}
											{#if line.free}
												<span class="badge badge-soft badge-sm">Free</span>
											{/if}
										</td>
										{#if fuel}
											<td class="text-right whitespace-nowrap tabular-nums">
												{line.volume === undefined ? '' : volume(line.volume, units)}
											</td>
											<td class="text-right whitespace-nowrap tabular-nums">
												{line.unit_price === undefined ? '' : amount(line.unit_price)}
											</td>
											<td>
												{#if line.full_tank}
													<span class="badge badge-soft badge-sm">Full</span>
												{/if}
											</td>
										{/if}
										<td class="text-right whitespace-nowrap tabular-nums">{money(line.amount)}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</section>
				{/if}

				{#if read.instalments && read.instalments.length > 0}
					<section class="panel overflow-x-auto" aria-labelledby="instalments-title">
						<h2 id="instalments-title" class="px-4 pt-4 font-medium">Instalments</h2>
						<table class="table">
							<thead>
								<tr>
									<th>Number</th>
									<th>Due</th>
									<th class="text-right">Amount</th>
								</tr>
							</thead>
							<tbody>
								{#each read.instalments as instalment (instalment.number)}
									<tr>
										<td class="tabular-nums">{instalment.number}</td>
										<td class="whitespace-nowrap">{day(instalment.due_on)}</td>
										<td class="text-right whitespace-nowrap tabular-nums">
											{money(instalment.amount)}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</section>
				{/if}
			</div>
		{/if}
	{/snippet}
</VehicleShell>
