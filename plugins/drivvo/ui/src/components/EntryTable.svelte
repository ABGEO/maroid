<script lang="ts">
	import { formatMoment, type PluginHost } from '@maroid/plugin-sdk';
	import { createPager, Pagination } from '@maroid/ui';

	import { createCarLogApi, type Kind, type LogEntryItem, type Period, type Vehicle } from '../api';
	import { amount, consumption, distance, kindLabel, volume } from '../lib/format';

	let {
		host,
		vehicle,
		period,
		kinds,
		list
	}: {
		host: PluginHost;
		vehicle: Vehicle;
		period: Period;
		/** The kinds that the table reads. */
		kinds: Kind[];
		/** The kind of a list, whose columns the table shows. The history shows its own. */
		list?: Kind;
	} = $props();

	const api = $derived(createCarLogApi(host));
	const units = $derived(vehicle.units);

	const pager = createPager((link?: string) => api.logEntries(vehicle.id, period, kinds, link));

	function entryHref(entry: LogEntryItem): string {
		return host.href(
			`/vehicles/${encodeURIComponent(vehicle.id)}/entries/${encodeURIComponent(entry.id)}`
		);
	}

	function moment(entry: LogEntryItem): string {
		return formatMoment(entry.occurred_at, host.timeZone, {
			dateStyle: 'medium',
			timeStyle: 'short'
		});
	}

	$effect(() => {
		void pager.reload();
	});
</script>

{#snippet money(entry: LogEntryItem)}
	<td
		class={['text-right whitespace-nowrap tabular-nums', entry.kind === 'income' && 'text-success']}
	>
		{entry.amount === undefined ? '' : amount(entry.amount)}
	</td>
{/snippet}

{#snippet labels(entry: LogEntryItem)}
	<td>{entry.labels.join(', ')}</td>
{/snippet}

{#if pager.status === 'loading' && pager.page === null}
	<div class="flex flex-col gap-2" aria-hidden="true">
		{#each [0, 1, 2, 3] as i (i)}
			<span class="skeleton h-10 w-full"></span>
		{/each}
	</div>
{:else if pager.status === 'error'}
	<div role="alert" class="alert alert-error alert-soft">
		<span>Failed to load the log entries.</span>
		<button type="button" class="btn btn-sm" onclick={() => pager.reload()}>Retry</button>
	</div>
{:else if pager.page?.items.length === 0}
	<div class="empty-state">No log entry in the period.</div>
{:else}
	{@const currency = pager.page?.items.find((entry) => entry.amount)?.amount?.currency ?? ''}
	<div class="panel overflow-x-auto">
		<table class="table-zebra table">
			<thead>
				<tr>
					<th>Time</th>
					{#if list === undefined}
						<th>Kind</th>
					{/if}
					<th class="text-right">Odometer</th>
					{#if list === undefined}
						<th>Types and fuels</th>
					{:else if list === 'refuel'}
						<th>Fuels</th>
						<th class="text-right">Volume</th>
						<th class="text-right">Unit price</th>
					{:else if list === 'maintenance' || list === 'expense'}
						<th>Title</th>
						<th>Types</th>
						{#if list === 'maintenance'}
							<th>Place</th>
						{/if}
					{:else if list === 'income'}
						<th>Income type</th>
					{/if}
					<th class="text-right">Amount{currency ? `, ${currency}` : ''}</th>
					{#if list === 'refuel'}
						<th>Full tank</th>
						<th class="text-right">Consumption</th>
					{/if}
				</tr>
			</thead>
			<tbody>
				{#each pager.page?.items ?? [] as entry (entry.id)}
					<tr>
						<td class="whitespace-nowrap">
							<a class="link link-hover" href={entryHref(entry)}>{moment(entry)}</a>
						</td>
						{#if list === undefined}
							<td>{kindLabel(entry.kind)}</td>
						{/if}
						<td class="text-right whitespace-nowrap tabular-nums">
							{distance(entry.odometer, units)}
						</td>
						{#if list === undefined}
							{@render labels(entry)}
						{:else if list === 'refuel'}
							{@render labels(entry)}
							<td class="text-right whitespace-nowrap tabular-nums">
								{entry.volume === undefined ? '' : volume(entry.volume, units)}
							</td>
							<td class="text-right whitespace-nowrap tabular-nums">
								{entry.unit_price === undefined ? '' : amount(entry.unit_price)}
							</td>
						{:else if list === 'maintenance' || list === 'expense'}
							<td>{entry.title ?? ''}</td>
							{@render labels(entry)}
							{#if list === 'maintenance'}
								<td>{entry.place ?? ''}</td>
							{/if}
						{:else if list === 'income'}
							<td>{entry.income_type ?? ''}</td>
						{/if}
						{@render money(entry)}
						{#if list === 'refuel'}
							<td>
								{#if entry.full_tank}
									<span class="badge badge-soft badge-sm">Full</span>
								{/if}
							</td>
							<td class="text-right whitespace-nowrap tabular-nums">
								{entry.consumption === undefined ? '' : consumption(entry.consumption, units)}
							</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
	<Pagination {pager} class="mt-4 ml-auto w-fit" />
{/if}
