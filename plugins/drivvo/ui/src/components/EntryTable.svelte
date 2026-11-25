<script lang="ts">
	import { formatMoment, type PluginHost } from '@maroid/plugin-sdk';
	import { createPager, Pagination } from '@maroid/ui';
	import { SvelteMap } from 'svelte/reactivity';

	import {
		createCarLogApi,
		type Kind,
		type LogEntry,
		type LogEntryItem,
		type Period,
		type Vehicle
	} from '../api';
	import { amount, consumption, distance, kindLabel, volume } from '../lib/format';
	import { createReader, type Reader } from '../lib/reader.svelte';
	import EntryDetails, { hasDetails } from './EntryDetails.svelte';

	let {
		host,
		vehicle,
		period,
		kinds,
		list,
		page
	}: {
		host: PluginHost;
		vehicle: Vehicle;
		period: Period;
		/** The kinds that the table reads. */
		kinds: Kind[];
		/** The kind of a list, whose columns the table shows. The history shows its own. */
		list?: Kind;
		/** The first segment of the address, under which a row opens its detail. */
		page: string;
	} = $props();

	const api = $derived(createCarLogApi(host));
	const units = $derived(vehicle.units);
	const columns = $derived(list === undefined ? 6 : list === 'refuel' ? 7 : 5);

	const pager = createPager((link?: string) => api.logEntries(vehicle.id, period, kinds, link));
	const details = new SvelteMap<string, Reader<LogEntry>>();
	let opened = $state<string | null>(null);

	function entryHref(entry: LogEntryItem): string {
		return host.href(`/${page}/${encodeURIComponent(vehicle.id)}/${encodeURIComponent(entry.id)}`);
	}

	function moment(entry: LogEntryItem): string {
		return formatMoment(entry.occurred_at, host.timeZone, {
			dateStyle: 'medium',
			timeStyle: 'short'
		});
	}

	function summary(entry: LogEntryItem): string {
		return entry.kind === 'income' ? (entry.income_type ?? '') : entry.labels.join(', ');
	}

	function load(id: string): void {
		const reader = details.get(id) ?? createReader<LogEntry>();
		details.set(id, reader);
		void reader.load(() => api.logEntry(vehicle.id, id));
	}

	function toggle(entry: LogEntryItem): void {
		if (opened === entry.id) {
			opened = null;
			return;
		}

		opened = entry.id;
		if (details.get(entry.id)?.status !== 'ready') {
			load(entry.id);
		}
	}

	/** A click on the row opens it too. A click on its button or a link acts alone. */
	function rowClick(event: MouseEvent, entry: LogEntryItem): void {
		if (!(event.target instanceof Element && event.target.closest('a, button'))) {
			toggle(entry);
		}
	}

	$effect(() => {
		void pager.reload();
	});
</script>

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
		<table class="table">
			<thead>
				<tr>
					<th class="w-0"><span class="sr-only">Details</span></th>
					<th>Time</th>
					{#if list === undefined}
						<th>Kind</th>
					{/if}
					<th class="text-right">Odometer</th>
					<th>
						{#if list === 'refuel'}
							Fuels
						{:else if list === 'income'}
							Income type
						{:else if list === undefined}
							Summary
						{:else}
							Title and types
						{/if}
					</th>
					{#if list === 'refuel'}
						<th class="text-right">Volume</th>
					{/if}
					<th class="text-right">Amount{currency ? `, ${currency}` : ''}</th>
					{#if list === 'refuel'}
						<th class="text-right">Consumption</th>
					{/if}
				</tr>
			</thead>
			<tbody>
				{#each pager.page?.items ?? [] as entry (entry.id)}
					{@const open = opened === entry.id}
					{@const reader = details.get(entry.id)}
					<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
					<tr
						class={['hover:bg-base-200 cursor-pointer', open && 'bg-base-200']}
						onclick={(event) => rowClick(event, entry)}
					>
						<td class="pr-0">
							<button
								type="button"
								class="btn btn-ghost btn-xs btn-square"
								aria-expanded={open}
								aria-controls={open ? `entry-${entry.id}` : undefined}
								aria-label="Details of the {kindLabel(entry.kind).toLowerCase()} of {moment(entry)}"
								onclick={() => toggle(entry)}
							>
								<svg
									class={['size-4 transition-transform', open && 'rotate-90']}
									viewBox="0 0 24 24"
									fill="none"
									stroke="currentColor"
									stroke-width="2"
									stroke-linecap="round"
									stroke-linejoin="round"
									aria-hidden="true"
								>
									<path d="m9 18 6-6-6-6" />
								</svg>
							</button>
						</td>
						<td class="whitespace-nowrap">{moment(entry)}</td>
						{#if list === undefined}
							<td>{kindLabel(entry.kind)}</td>
						{/if}
						<td class="text-right whitespace-nowrap tabular-nums">
							{distance(entry.odometer, units)}
						</td>
						<td>
							{#if entry.title}
								<div>{entry.title}</div>
								{#if summary(entry)}
									<div class="meta text-sm">{summary(entry)}</div>
								{/if}
							{:else}
								{summary(entry)}
							{/if}
						</td>
						{#if list === 'refuel'}
							<td class="text-right whitespace-nowrap tabular-nums">
								{entry.volume === undefined ? '' : volume(entry.volume, units)}
							</td>
						{/if}
						<td
							class={[
								'text-right whitespace-nowrap tabular-nums',
								entry.kind === 'income' && 'text-success'
							]}
						>
							{entry.amount === undefined ? '' : amount(entry.amount)}
						</td>
						{#if list === 'refuel'}
							<td class="text-right whitespace-nowrap tabular-nums">
								{entry.consumption === undefined ? '' : consumption(entry.consumption, units)}
							</td>
						{/if}
					</tr>
					{#if open}
						<tr id="entry-{entry.id}" class="bg-base-200">
							<td colspan={columns} class="px-4 pt-2 pb-4 sm:pl-14">
								{#if reader?.status === 'error'}
									<div role="alert" class="alert alert-error alert-soft">
										<span>Failed to load the details.</span>
										<button type="button" class="btn btn-sm" onclick={() => load(entry.id)}>
											Retry
										</button>
									</div>
								{:else if !reader?.value}
									<span class="skeleton block h-20 w-full" aria-hidden="true"></span>
								{:else}
									{#if hasDetails(reader.value)}
										<EntryDetails entry={reader.value} {units} />
									{:else}
										<p class="meta">No other detail.</p>
									{/if}
									<a class="btn btn-ghost btn-sm mt-3 -ml-3" href={entryHref(entry)}>
										Open the {kindLabel(entry.kind).toLowerCase()}
									</a>
								{/if}
							</td>
						</tr>
					{/if}
				{/each}
			</tbody>
		</table>
	</div>
	<Pagination {pager} class="mt-4 ml-auto w-fit" />
{/if}
