<script lang="ts">
	import { today, type PluginHost } from '@maroid/plugin-sdk';

	import '../../app.css';
	import { createParkingApi, type LotUsage, type SessionTotal } from '../../api';
	import BalanceCard from '../../components/BalanceCard.svelte';
	import ChargeChart from '../../components/ChargeChart.svelte';
	import LotList from '../../components/LotList.svelte';
	import NoAccess from '../../components/NoAccess.svelte';
	import NoData from '../../components/NoData.svelte';
	import PeriodPicker from '../../components/PeriodPicker.svelte';
	import { chargeSeries, type ChargeSeries } from '../../lib/chart';
	import { formatDuration, formatMoney } from '../../lib/format';
	import { DEFAULT_PERIOD, periodFrom, type Period } from '../../lib/period';

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createParkingApi(host));
	const canRead = $derived(host.can('history.read'));
	const canConnect = $derived(host.can('account.connect'));

	let held = $state<'loading' | 'some' | 'none' | 'error'>('loading');
	let period = $state<Period>(DEFAULT_PERIOD);
	let chart = $state<ChargeSeries | null>(null);
	let total = $state<SessionTotal | null>(null);
	let lots = $state<LotUsage[]>([]);
	let periodStatus = $state<'loading' | 'ready' | 'error'>('loading');
	let latestPeriod = 0;

	async function loadHeld(): Promise<void> {
		held = 'loading';

		try {
			const [sessions, transactions] = await Promise.all([
				api.sessions.any(),
				api.transactions.any()
			]);
			if (sessions === null || transactions === null) {
				return;
			}

			held = sessions || transactions ? 'some' : 'none';
		} catch (error) {
			console.error('Failed to read the stored history', error);
			held = 'error';
		}
	}

	async function loadPeriod(chosen: Period): Promise<void> {
		const request = ++latestPeriod;
		const now = today(host.timeZone);
		const from = periodFrom(chosen, now);

		periodStatus = 'loading';

		try {
			const [months, sums, usages] = await Promise.all([
				api.transactions.chargeMonths(from),
				api.sessions.total({ from }),
				api.sessions.lotUsages(from)
			]);
			if (request !== latestPeriod || months === null || sums === null || usages === null) {
				return;
			}

			chart = chargeSeries(months, from, now);
			total = sums;
			lots = usages;
			periodStatus = 'ready';
		} catch (error) {
			if (request !== latestPeriod) {
				return;
			}

			console.error('Failed to load the period', error);
			periodStatus = 'error';
		}
	}

	const emptyChart = $derived(chart !== null && chart.columns.size === 0 && chart.line.size === 0);

	$effect(() => {
		if (canRead) {
			void loadHeld();
		}
	});

	$effect(() => {
		if (held === 'some') {
			void loadPeriod(period);
		}
	});
</script>

<div>
	<header class="page-header">
		<div>
			<h1 class="page-title">Overview</h1>
			<p class="page-lead">
				The balance as the parking service reports it now, and the history that Maroid collects each
				hour.
			</p>
		</div>
	</header>

	<div class="flex flex-col gap-6">
		{#if canConnect}
			<BalanceCard {host} />
		{/if}

		{#if !canRead}
			<NoAccess />
		{:else if held === 'loading'}
			<span class="skeleton h-80 w-full" aria-hidden="true"></span>
		{:else if held === 'error'}
			<div role="alert" class="alert alert-error alert-soft">
				<span>Failed to load the history.</span>
				<button type="button" class="btn btn-sm" onclick={loadHeld}>Retry</button>
			</div>
		{:else if held === 'none'}
			<NoData {host} />
		{:else}
			<div class="flex justify-end">
				<PeriodPicker bind:value={period} label="The period of the overview" />
			</div>

			{#if periodStatus === 'loading'}
				<span class="skeleton h-80 w-full" aria-hidden="true"></span>
			{:else if periodStatus === 'error'}
				<div role="alert" class="alert alert-error alert-soft">
					<span>Failed to load the period.</span>
					<button type="button" class="btn btn-sm" onclick={() => loadPeriod(period)}>Retry</button>
				</div>
			{:else}
				{#if total !== null}
					<section class="grid gap-4 sm:grid-cols-3" aria-label="The sessions of the period">
						<div class="panel p-4">
							<div class="text-sm">Sessions</div>
							<div class="text-xl font-semibold tabular-nums">{total.count}</div>
						</div>
						<div class="panel p-4">
							<div class="text-sm">Time parked</div>
							<div class="text-xl font-semibold tabular-nums">
								{formatDuration(total.duration_seconds)}
							</div>
						</div>
						<div class="panel p-4">
							<div class="text-sm">Paid for sessions</div>
							<div class="text-xl font-semibold tabular-nums">{formatMoney(total.amount)}</div>
						</div>
					</section>
				{/if}

				<section class="panel p-4" aria-labelledby="chart-title">
					<h2 id="chart-title" class="mb-2 font-medium">Charges and balance</h2>
					{#if chart === null || emptyChart}
						<div class="empty-state">No transaction in the period.</div>
					{:else}
						<ChargeChart values={chart} />
					{/if}
				</section>

				<section class="panel p-4" aria-labelledby="lots-title">
					<h2 id="lots-title" class="mb-2 font-medium">Parking lots you use most</h2>
					<LotList {lots} />
				</section>
			{/if}
		{/if}
	</div>
</div>
