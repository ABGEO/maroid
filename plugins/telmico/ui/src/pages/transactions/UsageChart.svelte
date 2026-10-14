<script lang="ts">
	import { ColumnLineChart, type ChartSeries } from '@maroid/ui';

	import type { Transaction } from '../../api';
	import { chartPoints, kilowattHours } from '../../lib/chart';

	let { transactions }: { transactions: Transaction[] } = $props();

	const points = $derived(chartPoints(transactions));
	const byDate = $derived(new Map(points.map((point) => [point.date, point])));
	const series = $derived<ChartSeries[]>([
		{
			key: 'reading',
			columns: new Map(points.map((point) => [point.date, point.usage])),
			line: new Map(points.map((point) => [point.date, point.amount])),
			columnClass: 'fill-primary',
			lineClass: 'stroke-accent'
		}
	]);
</script>

<ColumnLineChart
	{series}
	lineFromZero
	label="The usage and the amount of each reading over the last 12 months. The table lists the same transactions."
>
	{#snippet legend()}
		<li class="flex items-center gap-1.5">
			<span class="bg-primary inline-block size-2.5 rounded-sm"></span>
			<span>Usage, kWh (left)</span>
		</li>
		<li class="flex items-center gap-1.5">
			<span class="bg-accent inline-block h-0.5 w-3"></span>
			<span>Amount, ₾ (right)</span>
		</li>
	{/snippet}

	{#snippet tooltip(date: string)}
		{@const point = byDate.get(date)}
		{#if point !== undefined}
			<div class="flex items-center gap-1.5">
				<span class="bg-primary inline-block size-2 rounded-sm"></span>
				<span>Usage</span>
				<span class="ml-auto tabular-nums">{kilowattHours(point.usage)} kWh</span>
			</div>
			<div class="flex items-center gap-1.5">
				<span class="bg-accent inline-block size-2 rounded-sm"></span>
				<span>Amount</span>
				<span class="ml-auto tabular-nums">{point.amount.toFixed(2)} ₾</span>
			</div>
		{/if}
	{/snippet}
</ColumnLineChart>
