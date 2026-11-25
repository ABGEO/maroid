<script lang="ts">
	import { formatMoment, type PluginHost, type RouteParams } from '@maroid/plugin-sdk';

	import { createCarLogApi, type LogEntry } from '../api';
	import EntryDetails, { hasDetails } from '../components/EntryDetails.svelte';
	import VehicleShell from '../components/VehicleShell.svelte';
	import { distance, kindLabel, money } from '../lib/format';
	import { isProblem, LOG_ENTRY_NOT_FOUND } from '../lib/problems';
	import { createReader } from '../lib/reader.svelte';
	import { createFleet } from '../lib/vehicles.svelte';

	let {
		host,
		params,
		page,
		title: pageTitle
	}: {
		host: PluginHost;
		params: RouteParams;
		/** The first segment of the address: the history or the list that opened the entry. */
		page: string;
		/** The title of that page. */
		title: string;
	} = $props();

	const api = $derived(createCarLogApi(host));
	const fleet = createFleet(
		() => host,
		() => params.vehicle
	);
	const entry = createReader<LogEntry>();

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
		return host.href(`/${page}/${encodeURIComponent(vehicleId)}`);
	}
</script>

<VehicleShell {host} {fleet} {page} {title} crumb={title}>
	{#snippet children(vehicle)}
		{@const units = vehicle.units}
		{@const read = entry.value}
		{#if notFound}
			<div role="alert" class="alert alert-info alert-soft max-w-xl">
				<span>This log entry is not in Maroid.</span>
				<a class="btn btn-sm" href={listHref(vehicle.id)}>Open {pageTitle.toLowerCase()}</a>
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
						Back to {pageTitle.toLowerCase()}
					</a>
				</div>

				<dl class="panel grid gap-4 p-4 sm:grid-cols-3">
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
				</dl>

				{#if hasDetails(read)}
					<div class="panel p-4">
						<EntryDetails entry={read} {units} />
					</div>
				{/if}
			</div>
		{/if}
	{/snippet}
</VehicleShell>
