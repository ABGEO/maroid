<script lang="ts">
	import { ColumnLineChart, type ChartSeries } from '@maroid/ui';

	import type { MonthFigures } from '../api';
	import { money } from '../lib/format';

	let { months }: { months: MonthFigures[] } = $props();

	interface Measure {
		key: 'fuel' | 'maintenance' | 'expense' | 'income';
		name: string;
		swatch: string;
	}

	const SPEND: (Measure & { fill: string })[] = [
		{ key: 'fuel', name: 'Fuel', swatch: 'bg-primary', fill: 'fill-primary' },
		{ key: 'maintenance', name: 'Maintenance', swatch: 'bg-secondary', fill: 'fill-secondary' },
		{ key: 'expense', name: 'Expenses', swatch: 'bg-accent', fill: 'fill-accent' }
	];
	const INCOME: Measure = { key: 'income', name: 'Income', swatch: 'bg-success' };
	const MEASURES: Measure[] = [...SPEND, INCOME];

	function dateOf(month: string): string {
		return `${month}-01`;
	}

	const dates = $derived(months.map((month) => dateOf(month.month)));
	const byDate = $derived(new Map(months.map((month) => [dateOf(month.month), month])));

	function valuesOf(key: Measure['key']): Map<string, number> {
		return new Map(months.map((month) => [dateOf(month.month), month[key].amount]));
	}

	const series = $derived<ChartSeries[]>([
		...SPEND.map((measure) => ({
			key: measure.key,
			columns: valuesOf(measure.key),
			line: new Map<string, number>(),
			columnClass: measure.fill,
			lineClass: 'stroke-none'
		})),
		{
			key: INCOME.key,
			columns: new Map<string, number>(),
			line: valuesOf(INCOME.key),
			columnClass: 'fill-none',
			lineClass: 'stroke-success'
		}
	]);
</script>

<ColumnLineChart
	{series}
	{dates}
	label="The spend on fuel, on maintenance, and on expenses of each month, and the income."
	lineFromZero
	tooltipWidth={200}
>
	{#snippet legend()}
		{#each MEASURES as measure (measure.key)}
			<li class="flex items-center gap-1.5">
				<span
					class={[
						'inline-block',
						measure === INCOME ? 'h-0.5 w-3' : 'size-2.5 rounded-sm',
						measure.swatch
					]}
				></span>
				<span>{measure.name}</span>
			</li>
		{/each}
	{/snippet}

	{#snippet tooltip(date: string)}
		{@const month = byDate.get(date)}
		{#if month !== undefined}
			{#each MEASURES as measure (measure.key)}
				<div class="flex items-center gap-1.5">
					<span class={['inline-block size-2 rounded-sm', measure.swatch]}></span>
					<span>{measure.name}</span>
					<span class="ml-auto tabular-nums">{money(month[measure.key])}</span>
				</div>
			{/each}
		{/if}
	{/snippet}
</ColumnLineChart>
