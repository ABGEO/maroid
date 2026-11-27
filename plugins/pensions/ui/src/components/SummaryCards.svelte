<script lang="ts">
	import { formatMoment } from '@maroid/plugin-sdk';

	import type { Summary } from '../api';
	import { formatMoney } from '../lib/format';

	let { summary, timeZone }: { summary: Summary; timeZone: string | undefined } = $props();

	const sources = $derived([
		{ label: 'Your contributions', value: summary.contributions.person },
		{ label: 'Employer contributions', value: summary.contributions.employer },
		{ label: 'State contributions', value: summary.contributions.state },
		{ label: 'All contributions', value: summary.contributions.total }
	]);
</script>

<section class="flex flex-col gap-4" aria-label="Account summary">
	<div class="stats stats-vertical sm:stats-horizontal panel">
		<div class="stat">
			<div class="stat-title">Asset value</div>
			<div class="stat-value text-2xl tabular-nums">{formatMoney(summary.asset_value)}</div>
		</div>
		<div class="stat">
			<div class="stat-title">Gain</div>
			<div class="stat-value text-2xl tabular-nums">{formatMoney(summary.gain)}</div>
		</div>
		<div class="stat">
			<div class="stat-title">Withheld</div>
			<div class="stat-value text-2xl tabular-nums">{formatMoney(summary.withheld_amount)}</div>
		</div>
	</div>

	<dl class="panel grid grid-cols-1 gap-x-6 gap-y-2 p-4 sm:grid-cols-2 lg:grid-cols-4">
		{#each sources as source (source.label)}
			<div>
				<dt class="text-base-content/70 text-sm">{source.label}</dt>
				<dd class="font-medium tabular-nums">{formatMoney(source.value)}</dd>
			</div>
		{/each}
	</dl>

	<p class="text-base-content/70 text-sm">
		Collected {formatMoment(summary.collected_at, timeZone, {
			dateStyle: 'medium',
			timeStyle: 'short'
		})}.
	</p>
</section>
