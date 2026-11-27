<script lang="ts">
	import { ColumnLineChart, type ChartSeries } from '@maroid/ui';

	import type { ChargeSeries } from '../lib/chart';

	let { values }: { values: ChargeSeries } = $props();

	const amount = new Intl.NumberFormat(undefined, {
		minimumFractionDigits: 2,
		maximumFractionDigits: 2
	});

	const series = $derived<ChartSeries[]>([
		{
			key: 'parking',
			columns: values.columns,
			line: values.line,
			columnClass: 'fill-primary',
			lineClass: 'stroke-accent'
		}
	]);
</script>

<ColumnLineChart
	{series}
	dates={values.slots}
	label="The charges of each month and the balance at the end of each month."
>
	{#snippet legend()}
		<li class="flex items-center gap-1.5">
			<span class="bg-primary inline-block size-2.5 rounded-sm"></span>
			<span>Charges, ₾ (left)</span>
		</li>
		<li class="flex items-center gap-1.5">
			<span class="bg-accent inline-block h-0.5 w-3"></span>
			<span>Balance, ₾ (right)</span>
		</li>
	{/snippet}

	{#snippet tooltip(slot: string)}
		{@const column = values.columns.get(slot)}
		{@const point = values.line.get(slot)}
		<div class="flex items-center gap-1.5">
			<span class="bg-primary inline-block size-2 rounded-sm"></span>
			<span>Charges</span>
			<span class="ml-auto tabular-nums">{amount.format(column ?? 0)} ₾</span>
		</div>
		{#if point !== undefined}
			<div class="flex items-center gap-1.5">
				<span class="bg-accent inline-block size-2 rounded-sm"></span>
				<span>Balance</span>
				<span class="ml-auto tabular-nums">{amount.format(point)} ₾</span>
			</div>
		{/if}
	{/snippet}
</ColumnLineChart>
