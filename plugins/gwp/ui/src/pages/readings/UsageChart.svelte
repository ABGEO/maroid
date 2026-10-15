<script lang="ts">
	import { ColumnLineChart, type ChartSeries } from '@maroid/ui';

	import type { Reading } from '../../api';
	import { chartSeries, meterTone } from '../../lib/chart';

	let { readings }: { readings: Reading[] } = $props();

	const grouped = $derived(chartSeries(readings));

	const series = $derived<ChartSeries[]>(
		grouped.meters.map((serial, index) => {
			const columns = new Map<string, number>();
			const line = new Map<string, number>();

			for (const date of grouped.dates) {
				const point = grouped.points.get(date)?.get(serial);
				if (point === undefined) {
					continue;
				}

				if (point.usage !== undefined) {
					columns.set(date, point.usage);
				}
				line.set(date, point.value);
			}

			const tone = meterTone(index);

			return { key: serial, columns, line, columnClass: tone.column, lineClass: tone.line };
		})
	);

	function volume(cubicMetres: number): string {
		return cubicMetres.toFixed(1);
	}
</script>

<ColumnLineChart
	{series}
	tooltipWidth={176}
	label="The usage and the value of each meter over the last 12 months. The table below lists the same readings."
>
	{#snippet legend()}
		{#each grouped.meters as serial, index (serial)}
			{@const tone = meterTone(index)}
			<li class="flex items-center gap-1.5">
				<span class={['inline-block size-2.5 rounded-sm', tone.swatch]}></span>
				<span class={['inline-block h-0.5 w-3', tone.swatch]}></span>
				<span class="font-mono">{serial}</span>
			</li>
		{/each}
		<li class="ml-auto">Columns: usage, m³ (left). Lines: meter value, m³ (right).</li>
	{/snippet}

	{#snippet tooltip(date: string)}
		{#each grouped.meters as serial, index (serial)}
			{@const point = grouped.points.get(date)?.get(serial)}
			{#if point !== undefined}
				<div class="flex items-center gap-1.5">
					<span class={['inline-block size-2 rounded-sm', meterTone(index).swatch]}></span>
					<span class="font-mono">{serial}</span>
					<span class="ml-auto tabular-nums">
						{point.usage === undefined ? 'None' : `${volume(point.usage)} m³`}
					</span>
				</div>
				<div class="text-base-content/60 pl-3.5 tabular-nums">meter {volume(point.value)} m³</div>
			{/if}
		{/each}
	{/snippet}
</ColumnLineChart>
