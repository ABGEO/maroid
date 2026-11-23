<script lang="ts">
	import { ColumnLineChart, type ChartSeries } from '@maroid/ui';

	import type { Day, Question } from '../../api';
	import { formatDay } from '../../lib/days';

	let { question, days }: { question: Question; days: Day[] } = $props();

	const yesOrNo = $derived(question.type === 'boolean');
	const byDate = $derived(new Map(days.map((day) => [day.date, day])));

	const series = $derived<ChartSeries[]>([
		{
			key: question.id,
			columns: new Map(days.map((day) => [day.date, day.count])),
			line: new Map(
				days.flatMap((day) => {
					const value = yesOrNo ? day.yes_count : day.mean;

					return value === undefined ? [] : [[day.date, value] as const];
				})
			),
			columnClass: 'fill-primary/40',
			lineClass: 'stroke-accent'
		}
	]);

	function rounded(value: number | undefined): string {
		return value === undefined ? 'None' : String(Math.round(value * 100) / 100);
	}
</script>

<ColumnLineChart
	{series}
	lineFromZero={yesOrNo}
	label="The values of each day: the count as columns, and {yesOrNo
		? 'the count of yes'
		: 'the mean'} as a line. The list below holds the same values."
>
	{#snippet legend()}
		<li class="flex items-center gap-1.5">
			<span class="bg-primary/40 inline-block size-2.5 rounded-sm"></span>
			Count of values (left)
		</li>
		<li class="flex items-center gap-1.5">
			<span class="bg-accent inline-block h-0.5 w-3"></span>
			{yesOrNo ? 'Count of yes' : 'Mean'} (right)
		</li>
	{/snippet}

	{#snippet tooltip(date: string)}
		{@const day = byDate.get(date)}
		<div class="font-medium">{formatDay(date)}</div>
		{#if day}
			<div class="tabular-nums">{day.count} {day.count === 1 ? 'value' : 'values'}</div>
			{#if yesOrNo}
				<div class="tabular-nums">{day.yes_count ?? 0} yes</div>
			{:else}
				<div class="tabular-nums">
					mean {rounded(day.mean)}, from {rounded(day.min)} to {rounded(day.max)}
				</div>
			{/if}
		{/if}
	{/snippet}
</ColumnLineChart>
