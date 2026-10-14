<script lang="ts">
	import { ApiError, type Page, type PluginHost } from '@maroid/plugin-sdk';
	import { createPager, Pagination } from '@maroid/ui';

	import '../../app.css';
	import { createElectricityApi, type Transaction } from '../../api';
	import { kilowattHours } from '../../lib/chart';
	import UsageChart from './UsageChart.svelte';

	const NOT_ASSIGNED = '/problems/dev.maroid.telmico/not-assigned';
	const CHART_MONTHS = 12;

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createElectricityApi(host));

	let notAssigned = $state(false);
	let series = $state<Transaction[] | null>(null);
	let seriesStatus = $state<'loading' | 'ready' | 'error'>('loading');

	// The pager reports any failure as one error, so the read answers the missing
	// assignment itself: an empty page, with the flag that the page shows first.
	const pager = createPager(async (link?: string): Promise<Page<Transaction> | null> => {
		try {
			const answered = await api.transactions.page(link);
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
			const read = await api.transactions.since(chartStart());
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

			console.error('Failed to load the transactions of the chart', error);
			seriesStatus = 'error';
		}
	}

	/** A date YYYY-MM-DD as dd/mm/yyyy, with no conversion of time zone. */
	function shownDate(date: string): string {
		const [year, month, day] = date.split('-');

		return `${day}/${month}/${year}`;
	}

	function shownEnergy(value: number | undefined): string {
		return value === undefined ? '' : kilowattHours(value);
	}

	const hasReading = $derived(
		(series ?? []).some((transaction) => transaction.usage !== undefined)
	);

	$effect(() => {
		void pager.reload();
		void loadSeries();
	});
</script>

<div>
	<header class="page-header">
		<div>
			<h1 class="page-title">Transactions</h1>
			<p class="page-lead">
				The charges and the payments of the assigned customer. The bill of each month is on the
				bills page.
			</p>
		</div>
	</header>

	{#if notAssigned}
		<div role="alert" class="alert alert-info alert-soft max-w-xl">
			<span>Connect the electricity account and assign the customer of this workspace first.</span>
			<a class="btn btn-sm" href={host.href('/account')}>Open the account</a>
		</div>
	{:else}
		<div class="flex flex-col gap-6">
			<section class="panel p-4" aria-labelledby="chart-title">
				<h2 id="chart-title" class="mb-2 font-medium">The last 12 months</h2>
				{#if seriesStatus === 'loading'}
					<span class="skeleton h-80 w-full" aria-hidden="true"></span>
				{:else if seriesStatus === 'error'}
					<div role="alert" class="alert alert-error alert-soft">
						<span>Failed to load the transactions of the chart.</span>
						<button type="button" class="btn btn-sm" onclick={loadSeries}>Retry</button>
					</div>
				{:else if series === null || !hasReading}
					<div class="empty-state">No reading in the last 12 months.</div>
				{:else}
					<UsageChart transactions={series} />
				{/if}
			</section>

			<section class="min-w-0" aria-label="Every transaction">
				{#if pager.status === 'loading' && pager.page === null}
					<div class="flex flex-col gap-2" aria-hidden="true">
						{#each [0, 1, 2, 3] as i (i)}
							<span class="skeleton h-10 w-full"></span>
						{/each}
					</div>
				{:else if pager.status === 'error'}
					<div role="alert" class="alert alert-error alert-soft">
						<span>Failed to load the transactions.</span>
						<button type="button" class="btn btn-sm" onclick={() => pager.reload()}>Retry</button>
					</div>
				{:else if pager.page?.items.length === 0}
					<div class="empty-state">No transaction yet.</div>
				{:else}
					<div class="panel overflow-x-auto">
						<table class="table-zebra table">
							<thead>
								<tr>
									<th>Date</th>
									<th>Operation</th>
									<th class="text-right">Usage, kWh</th>
									<th class="text-right">Meter, kWh</th>
									<th class="text-right">Tariff, ₾/kWh</th>
									<th class="text-right">Amount, ₾</th>
									<th class="text-right">Balance, ₾</th>
								</tr>
							</thead>
							<tbody>
								{#each pager.page?.items ?? [] as transaction (transaction.id)}
									<tr>
										<td class="whitespace-nowrap">{shownDate(transaction.transaction_date)}</td>
										<td>{transaction.operation}</td>
										<td class="text-right tabular-nums">{shownEnergy(transaction.usage)}</td>
										<td class="text-right tabular-nums">{shownEnergy(transaction.meter_value)}</td>
										<td class="text-right tabular-nums">{transaction.tariff?.toFixed(4) ?? ''}</td>
										<td class="text-right tabular-nums"
											>{transaction.amount?.amount.toFixed(2) ?? ''}</td
										>
										<td class="text-right tabular-nums"
											>{transaction.balance?.amount.toFixed(2) ?? ''}</td
										>
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
