<script lang="ts">
	import { today, type PluginHost } from '@maroid/plugin-sdk';

	import '../../app.css';
	import { createPensionApi, type SummaryView } from '../../api';
	import NoAccess from '../../components/NoAccess.svelte';
	import PeriodPicker from '../../components/PeriodPicker.svelte';
	import SetupNotice from '../../components/SetupNotice.svelte';
	import SummaryCards from '../../components/SummaryCards.svelte';
	import ValueChart from '../../components/ValueChart.svelte';
	import { valueSeries, type ValueSeries } from '../../lib/chart';
	import { DEFAULT_PERIOD, periodFrom, type Period } from '../../lib/period';
	import { pageState } from '../../lib/state';

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createPensionApi(host));
	const canRead = $derived(host.can('account.read'));

	let view = $state<SummaryView | null>(null);
	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let period = $state<Period>(DEFAULT_PERIOD);
	let chart = $state<ValueSeries | null>(null);
	let chartStatus = $state<'loading' | 'ready' | 'error'>('loading');
	let latestChart = 0;

	const shown = $derived(view === null ? null : pageState(view));

	async function load(): Promise<void> {
		status = 'loading';

		try {
			const read = await api.summary.read();
			if (read !== null) {
				view = read;
				status = 'ready';
			}
		} catch (error) {
			console.error('Failed to load the summary', error);
			status = 'error';
		}
	}

	async function loadChart(chosen: Period): Promise<void> {
		const request = ++latestChart;
		const now = today(host.timeZone);
		const from = periodFrom(chosen, now);

		chartStatus = 'loading';

		try {
			const [values, months] = await Promise.all([
				api.assetValues.since(from),
				api.contributions.months(from)
			]);
			if (request !== latestChart || values === null || months === null) {
				return;
			}

			chart = valueSeries(values, months, from, now);
			chartStatus = 'ready';
		} catch (error) {
			if (request !== latestChart) {
				return;
			}

			console.error('Failed to load the chart', error);
			chartStatus = 'error';
		}
	}

	const empty = $derived(chart !== null && chart.columns.size === 0 && chart.line.size === 0);

	$effect(() => {
		if (canRead) {
			void load();
		}
	});

	$effect(() => {
		if (shown === 'ready') {
			void loadChart(period);
		}
	});
</script>

<div>
	<header class="page-header">
		<div>
			<h1 class="page-title">Overview</h1>
			<p class="page-lead">
				The pension account as the last collection stored it. Maroid collects it on the first day of
				each month.
			</p>
		</div>
	</header>

	{#if !canRead}
		<NoAccess />
	{:else if status === 'loading'}
		<div class="flex flex-col gap-4" aria-hidden="true">
			<span class="skeleton h-24 w-full"></span>
			<span class="skeleton h-80 w-full"></span>
		</div>
	{:else if status === 'error'}
		<div role="alert" class="alert alert-error alert-soft">
			<span>Failed to load the summary.</span>
			<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
		</div>
	{:else if view?.summary !== undefined && shown === 'ready'}
		<div class="flex flex-col gap-6">
			<SummaryCards summary={view.summary} timeZone={host.timeZone} />

			<section class="panel p-4" aria-labelledby="chart-title">
				<div class="mb-2 flex flex-wrap items-center justify-between gap-2">
					<h2 id="chart-title" class="font-medium">Asset value and contributions</h2>
					<PeriodPicker bind:value={period} label="The period of the chart" />
				</div>
				{#if chartStatus === 'loading'}
					<span class="skeleton h-80 w-full" aria-hidden="true"></span>
				{:else if chartStatus === 'error'}
					<div role="alert" class="alert alert-error alert-soft">
						<span>Failed to load the chart.</span>
						<button type="button" class="btn btn-sm" onclick={() => loadChart(period)}>
							Retry
						</button>
					</div>
				{:else if chart === null || empty}
					<div class="empty-state">No asset value and no contribution in the period.</div>
				{:else}
					<ValueChart values={chart} />
				{/if}
			</section>
		</div>
	{:else if shown !== null && shown !== 'ready'}
		<SetupNotice {host} kind={shown} />
	{/if}
</div>
