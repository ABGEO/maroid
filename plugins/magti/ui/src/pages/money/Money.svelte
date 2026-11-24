<script lang="ts">
	import { ApiError, formatMoment, type Page, type PluginHost } from '@maroid/plugin-sdk';
	import { createPager, Pagination } from '@maroid/ui';

	import '../../app.css';
	import { createTelecomApi, type MoneyEntry, type MonthTotal } from '../../api';
	import { addMonths, currentTbilisiMonth } from '../../lib/dates';

	const NOT_ASSIGNED = '/problems/dev.maroid.magti/not-assigned';
	const HOME_LINE = '/problems/dev.maroid.magti/home-line';
	const MONTHS_BEFORE = 11;

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createTelecomApi(host));

	let setUp = $state<'ready' | 'not_assigned' | 'home_line'>('ready');
	let totals = $state<MonthTotal[] | null>(null);
	let totalsStatus = $state<'loading' | 'ready' | 'error'>('loading');

	function missing(error: unknown): boolean {
		if (error instanceof ApiError && error.is(NOT_ASSIGNED)) {
			setUp = 'not_assigned';

			return true;
		}

		if (error instanceof ApiError && error.is(HOME_LINE)) {
			setUp = 'home_line';

			return true;
		}

		return false;
	}

	// The pager reports any failure as one error, so the read answers the missing
	// assignment itself: an empty page, with the state that the page shows first.
	const pager = createPager(async (link?: string): Promise<Page<MoneyEntry> | null> => {
		try {
			return await api.money.page(link);
		} catch (error) {
			if (missing(error)) {
				return { items: [], self: '', first: '' };
			}

			throw error;
		}
	});

	async function loadTotals(): Promise<void> {
		totalsStatus = 'loading';

		try {
			const read = await api.money.monthTotals(addMonths(currentTbilisiMonth(), -MONTHS_BEFORE));
			if (read !== null) {
				totals = read.items;
				totalsStatus = 'ready';
			}
		} catch (error) {
			if (missing(error)) {
				totals = [];
				totalsStatus = 'ready';

				return;
			}

			console.error('Failed to load the totals of each month', error);
			totalsStatus = 'error';
		}
	}

	const monthLabel = new Intl.DateTimeFormat(undefined, {
		month: 'short',
		year: 'numeric',
		timeZone: 'UTC'
	});

	function shownMonth(month: string): string {
		return monthLabel.format(new Date(`${month}-01T00:00:00Z`));
	}

	function shownAmount(entry: MoneyEntry): string {
		const sign = entry.direction === 'in' ? '+' : '−';

		return `${sign}${entry.amount.amount.toFixed(2)} ₾`;
	}

	$effect(() => {
		void pager.reload();
		void loadTotals();
	});
</script>

<div>
	<header class="page-header">
		<div>
			<h1 class="page-title">Money</h1>
			<p class="page-lead">The top ups and the charges of the assigned subscriber.</p>
		</div>
	</header>

	{#if setUp === 'not_assigned'}
		<div role="alert" class="alert alert-info alert-soft max-w-xl">
			<span>Connect the Magti account and assign a mobile subscriber first.</span>
			<a class="btn btn-sm" href={host.href('/account')}>Open the account</a>
		</div>
	{:else if setUp === 'home_line'}
		<div class="empty-state max-w-xl">
			The assigned subscriber is a home line. Magti answers money entries for a mobile subscriber
			alone.
		</div>
	{:else}
		<div class="flex flex-col gap-6">
			<section class="panel p-4" aria-labelledby="totals-title">
				<h2 id="totals-title" class="mb-2 font-medium">The last 12 months</h2>
				{#if totalsStatus === 'loading'}
					<span class="skeleton h-40 w-full" aria-hidden="true"></span>
				{:else if totalsStatus === 'error'}
					<div role="alert" class="alert alert-error alert-soft">
						<span>Failed to load the totals of each month.</span>
						<button type="button" class="btn btn-sm" onclick={loadTotals}>Retry</button>
					</div>
				{:else}
					<div class="overflow-x-auto">
						<table class="table-sm table">
							<thead>
								<tr>
									<th>Month</th>
									<th class="text-right">Top ups, ₾</th>
									<th class="text-right">Charges, ₾</th>
								</tr>
							</thead>
							<tbody>
								{#each totals ?? [] as total (total.month)}
									<tr>
										<td class="whitespace-nowrap">{shownMonth(total.month)}</td>
										<td class="text-success text-right tabular-nums"
											>{total.top_ups.amount.toFixed(2)}</td
										>
										<td class="text-right tabular-nums">{total.charges.amount.toFixed(2)}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</section>

			<section class="min-w-0" aria-label="Every money entry">
				{#if pager.status === 'loading' && pager.page === null}
					<div class="flex flex-col gap-2" aria-hidden="true">
						{#each [0, 1, 2, 3] as i (i)}
							<span class="skeleton h-10 w-full"></span>
						{/each}
					</div>
				{:else if pager.status === 'error'}
					<div role="alert" class="alert alert-error alert-soft">
						<span>Failed to load the money entries.</span>
						<button type="button" class="btn btn-sm" onclick={() => pager.reload()}>Retry</button>
					</div>
				{:else if pager.page?.items.length === 0}
					<div class="empty-state">No money entry yet.</div>
				{:else}
					<div class="panel overflow-x-auto">
						<table class="table-zebra table">
							<thead>
								<tr>
									<th>Time</th>
									<th>Kind</th>
									<th class="text-right">Amount</th>
								</tr>
							</thead>
							<tbody>
								{#each pager.page?.items ?? [] as entry (entry.id)}
									<tr>
										<td class="whitespace-nowrap"
											>{formatMoment(entry.occurred_at, host.timeZone, {
												dateStyle: 'medium',
												timeStyle: 'short'
											})}</td
										>
										<td>{entry.kind}</td>
										<td
											class={[
												'text-right whitespace-nowrap tabular-nums',
												entry.direction === 'in' && 'text-success'
											]}>{shownAmount(entry)}</td
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
