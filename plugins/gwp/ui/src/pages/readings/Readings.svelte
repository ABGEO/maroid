<script lang="ts">
	import { ApiError, type Page, type PluginHost } from '@maroid/plugin-sdk';
	import { createPager, Pagination } from '@maroid/ui';

	import '../../app.css';
	import { createGwpApi, type Reading } from '../../api';
	import UsageChart from './UsageChart.svelte';

	const NOT_ASSIGNED = '/problems/dev.maroid.gwp/not-assigned';
	const CHART_MONTHS = 12;

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createGwpApi(host));

	let notAssigned = $state(false);
	let series = $state<Reading[] | null>(null);
	let seriesStatus = $state<'loading' | 'ready' | 'error'>('loading');

	// The pager reports any failure as one error, so the read answers the missing
	// assignment itself: an empty page, with the flag that the page shows first.
	const pager = createPager(async (link?: string): Promise<Page<Reading> | null> => {
		try {
			const answered = await api.readings.page(link);
			notAssigned = false;

			return answered;
		} catch (error) {
			if (error instanceof ApiError && error.is(NOT_ASSIGNED)) {
				notAssigned = true;

				return { items: [], self: '', first: '' };
			}

			throw error;
		}
	});

	/** The day 12 months before today, YYYY-MM-DD, in the time of the person. */
	function chartStart(): string {
		const start = new Date();
		start.setMonth(start.getMonth() - CHART_MONTHS);

		const month = String(start.getMonth() + 1).padStart(2, '0');
		const day = String(start.getDate()).padStart(2, '0');

		return `${start.getFullYear()}-${month}-${day}`;
	}

	async function loadSeries(): Promise<void> {
		seriesStatus = 'loading';

		try {
			const read = await api.readings.since(chartStart());
			if (read !== null) {
				series = read;
				seriesStatus = 'ready';
			}
		} catch (error) {
			if (error instanceof ApiError && error.is(NOT_ASSIGNED)) {
				series = [];
				seriesStatus = 'ready';

				return;
			}

			console.error('Failed to load the readings of the chart', error);
			seriesStatus = 'error';
		}
	}

	/** A reading date YYYY-MM-DD as dd/mm/yyyy, with no conversion of time zone. */
	function shownDate(readingDate: string): string {
		const [year, month, day] = readingDate.split('-');

		return `${day}/${month}/${year}`;
	}

	function volume(cubicMetres: number): string {
		return cubicMetres.toFixed(1);
	}

	$effect(() => {
		void pager.reload();
		void loadSeries();
	});
</script>

<div>
	<header class="page-header">
		<div>
			<h1 class="page-title">Readings</h1>
			<p class="page-lead">
				The value of each meter of the assigned customer, and the usage between two readings.
			</p>
		</div>
	</header>

	{#if notAssigned}
		<div role="alert" class="alert alert-info alert-soft max-w-xl">
			<span>Connect the water account and assign the customer of this workspace first.</span>
			<a class="btn btn-sm" href={host.href('/account')}>Open the account</a>
		</div>
	{:else}
		<div class="grid grid-cols-1 gap-6 lg:grid-cols-2 lg:items-start">
			<section
				class="panel order-first p-4 lg:sticky lg:top-4 lg:order-last"
				aria-labelledby="chart-title"
			>
				<h2 id="chart-title" class="mb-2 font-medium">The last 12 months</h2>
				{#if seriesStatus === 'loading'}
					<span class="skeleton h-80 w-full" aria-hidden="true"></span>
				{:else if seriesStatus === 'error'}
					<div role="alert" class="alert alert-error alert-soft">
						<span>Failed to load the readings of the chart.</span>
						<button type="button" class="btn btn-sm" onclick={loadSeries}>Retry</button>
					</div>
				{:else if series === null || series.length === 0}
					<div class="empty-state">No reading in the last 12 months.</div>
				{:else}
					<UsageChart readings={series} />
				{/if}
			</section>

			<section class="min-w-0" aria-label="Every reading">
				{#if pager.status === 'loading' && pager.page === null}
					<div class="flex flex-col gap-2" aria-hidden="true">
						{#each [0, 1, 2, 3] as i (i)}
							<span class="skeleton h-10 w-full"></span>
						{/each}
					</div>
				{:else if pager.status === 'error'}
					<div role="alert" class="alert alert-error alert-soft">
						<span>Failed to load the readings.</span>
						<button type="button" class="btn btn-sm" onclick={() => pager.reload()}>Retry</button>
					</div>
				{:else if pager.page?.items.length === 0}
					<div class="empty-state">No reading yet.</div>
				{:else}
					<div class="panel overflow-x-auto">
						<table class="table-zebra table">
							<thead>
								<tr>
									<th>Date</th>
									<th>Meter</th>
									<th class="text-right">Value, m³</th>
									<th class="text-right">Usage, m³</th>
								</tr>
							</thead>
							<tbody>
								{#each pager.page?.items ?? [] as reading (reading.id)}
									<tr>
										<td>{shownDate(reading.reading_date)}</td>
										<td class="font-mono">{reading.serial_number}</td>
										<td class="text-right tabular-nums">{volume(reading.value)}</td>
										<td class="text-right tabular-nums">
											{#if reading.usage === undefined}
												<span class="text-base-content/50">None</span>
											{:else}
												<span class:text-error={reading.usage < 0}>{volume(reading.usage)}</span>
											{/if}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
					<Pagination {pager} class="mt-4 ml-auto w-fit" />
				{/if}
			</section>
		</div>
	{/if}
</div>
