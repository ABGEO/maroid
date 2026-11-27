<script lang="ts">
	import { today, type PluginHost } from '@maroid/plugin-sdk';

	import '../../app.css';
	import { createPensionApi, type PortfolioShare, type UnitPrice } from '../../api';
	import NoAccess from '../../components/NoAccess.svelte';
	import PeriodPicker from '../../components/PeriodPicker.svelte';
	import SetupNotice from '../../components/SetupNotice.svelte';
	import UnitPriceChart from '../../components/UnitPriceChart.svelte';
	import { formatMoney, formatPercent, formatUnits } from '../../lib/format';
	import { DEFAULT_PERIOD, periodFrom, type Period } from '../../lib/period';
	import { pageState, type PageState } from '../../lib/state';

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createPensionApi(host));
	const canRead = $derived(host.can('account.read'));

	let shown = $state<PageState | null>(null);
	let shares = $state<PortfolioShare[]>([]);
	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let period = $state<Period>(DEFAULT_PERIOD);
	let prices = $state<UnitPrice[]>([]);
	let chartStatus = $state<'loading' | 'ready' | 'error'>('loading');
	let latestChart = 0;

	async function load(): Promise<void> {
		status = 'loading';

		try {
			const view = await api.summary.read();
			if (view === null) {
				return;
			}

			shown = pageState(view);
			if (shown === 'ready') {
				const read = await api.portfolios.shares();
				if (read === null) {
					return;
				}

				shares = read;
			}

			status = 'ready';
		} catch (error) {
			console.error('Failed to load the portfolio shares', error);
			status = 'error';
		}
	}

	async function loadChart(chosen: Period): Promise<void> {
		const request = ++latestChart;

		chartStatus = 'loading';

		try {
			const read = await api.portfolios.unitPrices(periodFrom(chosen, today(host.timeZone)));
			if (request !== latestChart || read === null) {
				return;
			}

			prices = read;
			chartStatus = 'ready';
		} catch (error) {
			if (request !== latestChart) {
				return;
			}

			console.error('Failed to load the unit prices', error);
			chartStatus = 'error';
		}
	}

	$effect(() => {
		if (canRead) {
			void load();
		}
	});

	$effect(() => {
		if (shown === 'ready' && shares.length > 0) {
			void loadChart(period);
		}
	});
</script>

<div>
	<header class="page-header">
		<div>
			<h1 class="page-title">Portfolios</h1>
			<p class="page-lead">
				The share of the account in each portfolio, and the unit price of each portfolio over time.
			</p>
		</div>
	</header>

	{#if !canRead}
		<NoAccess />
	{:else if status === 'loading'}
		<div class="flex flex-col gap-4" aria-hidden="true">
			<span class="skeleton h-32 w-full"></span>
			<span class="skeleton h-80 w-full"></span>
		</div>
	{:else if status === 'error'}
		<div role="alert" class="alert alert-error alert-soft">
			<span>Failed to load the portfolio shares.</span>
			<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
		</div>
	{:else if shown !== null && shown !== 'ready'}
		<SetupNotice {host} kind={shown} />
	{:else if shares.length === 0}
		<div class="empty-state">No portfolio share yet.</div>
	{:else}
		<div class="flex flex-col gap-6">
			<div class="panel overflow-x-auto">
				<table class="table-zebra table">
					<thead>
						<tr>
							<th>Portfolio</th>
							<th>Asset manager</th>
							<th>Ticker</th>
							<th>Risk tier</th>
							<th class="text-right">Chosen</th>
							<th class="text-right">Current</th>
							<th class="text-right">Amount</th>
							<th class="text-right">Units</th>
							<th class="text-right">Unit price</th>
						</tr>
					</thead>
					<tbody>
						{#each shares as share (share.portfolio_id)}
							<tr>
								<td>{share.portfolio_name}</td>
								<td>{share.asset_manager_name}</td>
								<td>{share.ticker}</td>
								<td>{share.risk_tier_name}</td>
								<td class="text-right tabular-nums">{formatPercent(share.target_share)}</td>
								<td class="text-right tabular-nums">{formatPercent(share.current_share)}</td>
								<td class="text-right whitespace-nowrap tabular-nums">
									{formatMoney(share.amount)}
								</td>
								<td class="text-right tabular-nums">{formatUnits(share.units)}</td>
								<td class="text-right tabular-nums">{formatUnits(share.unit_price)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			<section class="panel p-4" aria-labelledby="chart-title">
				<div class="mb-2 flex flex-wrap items-center justify-between gap-2">
					<h2 id="chart-title" class="font-medium">Unit price</h2>
					<PeriodPicker bind:value={period} label="The period of the chart" />
				</div>
				{#if chartStatus === 'loading'}
					<span class="skeleton h-80 w-full" aria-hidden="true"></span>
				{:else if chartStatus === 'error'}
					<div role="alert" class="alert alert-error alert-soft">
						<span>Failed to load the unit prices.</span>
						<button type="button" class="btn btn-sm" onclick={() => loadChart(period)}>
							Retry
						</button>
					</div>
				{:else}
					<UnitPriceChart {shares} {prices} />
				{/if}
			</section>
		</div>
	{/if}
</div>
