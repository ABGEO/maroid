<script lang="ts">
	import { ColumnLineChart, type ChartSeries } from '@maroid/ui';

	import type { TrafficDay } from '../../api';
	import type { Measure } from '../../lib/measure';

	let {
		days,
		dates,
		measures,
		label,
		shown
	}: {
		days: TrafficDay[];
		/** Every day of the period, so a day with no row keeps its slot. */
		dates: string[];
		measures: Measure[];
		label: string;
		shown: (value: number) => string;
	} = $props();

	const SWATCHES = { primary: 'bg-primary', secondary: 'bg-secondary' } as const;
	const FILLS = { primary: 'fill-primary', secondary: 'fill-secondary' } as const;

	const byDate = $derived(new Map(days.map((day) => [day.day, day])));
	const series = $derived<ChartSeries[]>(
		measures.map((measure) => ({
			key: measure.key,
			columns: new Map(
				days.flatMap((day) => {
					const value = measure.read(day);

					return value === undefined ? [] : [[day.day, value] as const];
				})
			),
			line: new Map(),
			columnClass: FILLS[measure.color],
			lineClass: 'stroke-none'
		}))
	);
</script>

<ColumnLineChart {series} {dates} {label} lineFromZero>
	{#snippet legend()}
		{#each measures as measure (measure.key)}
			<li class="flex items-center gap-1.5">
				<span class={['inline-block size-2.5 rounded-sm', SWATCHES[measure.color]]}></span>
				<span>{measure.name}</span>
			</li>
		{/each}
	{/snippet}

	{#snippet tooltip(date: string)}
		{@const day = byDate.get(date)}
		{#if day !== undefined}
			{#each measures as measure (measure.key)}
				{@const value = measure.read(day)}
				<div class="flex items-center gap-1.5">
					<span class={['inline-block size-2 rounded-sm', SWATCHES[measure.color]]}></span>
					<span>{measure.name}</span>
					<span class="ml-auto tabular-nums">{value === undefined ? 'None' : shown(value)}</span>
				</div>
			{/each}
		{/if}
	{/snippet}
</ColumnLineChart>
