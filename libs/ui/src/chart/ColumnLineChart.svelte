<script lang="ts">
	import { scaleBand, scaleLinear } from 'd3-scale';
	import { curveMonotoneX, line } from 'd3-shape';
	import type { Snippet } from 'svelte';

	import type { ChartSeries } from './series';

	const HEIGHT = 320;
	const MARGIN = { top: 12, right: 52, bottom: 28, left: 52 };
	const TICKS = 5;
	const MIN_LABEL_WIDTH = 56;

	let {
		series,
		label,
		lineFromZero = false,
		tooltipWidth = 160,
		legend,
		tooltip
	}: {
		series: ChartSeries[];
		/** The accessible name of the chart. */
		label: string;
		/** Start the right axis at zero instead of at the smallest line value. */
		lineFromZero?: boolean;
		tooltipWidth?: number;
		/** The items of the legend, each one an `<li>`. */
		legend: Snippet;
		/** The body of the tooltip for the date under the pointer. */
		tooltip: Snippet<[string]>;
	} = $props();

	let width = $state(0);
	let active = $state<string | null>(null);

	const monthLabel = new Intl.DateTimeFormat(undefined, {
		month: 'short',
		year: '2-digit',
		timeZone: 'UTC'
	});
	const dayLabel = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeZone: 'UTC' });

	const dates = $derived(
		[...new Set(series.flatMap((one) => [...one.columns.keys(), ...one.line.keys()]))].sort()
	);
	const columnValues = $derived(series.flatMap((one) => [...one.columns.values()]));
	const lineValues = $derived(series.flatMap((one) => [...one.line.values()]));

	const innerWidth = $derived(Math.max(0, width - MARGIN.left - MARGIN.right));
	const innerHeight = HEIGHT - MARGIN.top - MARGIN.bottom;

	const x = $derived(
		scaleBand<string>().domain(dates).range([0, innerWidth]).paddingInner(0.25).paddingOuter(0.1)
	);
	const xSeries = $derived(
		scaleBand<string>()
			.domain(series.map((one) => one.key))
			.range([0, x.bandwidth()])
			.padding(series.length > 1 ? 0.08 : 0)
	);

	const yColumn = $derived(
		scaleLinear()
			.domain([Math.min(0, ...columnValues), Math.max(0, ...columnValues)])
			.nice(TICKS)
			.range([innerHeight, 0])
	);
	const yLine = $derived(
		scaleLinear()
			.domain([
				lineFromZero ? Math.min(0, ...lineValues) : Math.min(...lineValues),
				lineFromZero ? Math.max(0, ...lineValues) : Math.max(...lineValues)
			])
			.nice(TICKS)
			.range([innerHeight, 0])
	);
	const columnTick = $derived(yColumn.tickFormat(TICKS));
	const lineTick = $derived(yLine.tickFormat(TICKS));

	// A label fits in about 56 pixels, so a narrow chart names every second or third month.
	const labelStep = $derived(
		Math.max(1, Math.ceil(dates.length / Math.max(1, Math.floor(innerWidth / MIN_LABEL_WIDTH))))
	);

	function center(date: string): number {
		return (x(date) ?? 0) + x.bandwidth() / 2;
	}

	function slotX(date: string): number {
		return (x(date) ?? 0) - (x.step() - x.bandwidth()) / 2;
	}

	function pathOf(one: ChartSeries): string {
		const points = dates.flatMap((date) => {
			const value = one.line.get(date);

			return value === undefined ? [] : [{ date, value }];
		});

		return (
			line<{ date: string; value: number }>()
				.x((point) => center(point.date))
				.y((point) => yLine(point.value))
				.curve(curveMonotoneX)(points) ?? ''
		);
	}

	function shownDate(date: string): Date {
		return new Date(`${date}T00:00:00Z`);
	}

	const tooltipLeft = $derived(
		active === null
			? 0
			: Math.min(Math.max(0, MARGIN.left + center(active) - tooltipWidth / 2), width - tooltipWidth)
	);
</script>

<div class="relative" bind:clientWidth={width}>
	<ul class="text-base-content/70 mb-2 flex flex-wrap gap-x-4 gap-y-1 text-xs">
		{@render legend()}
	</ul>

	{#if innerWidth > 0}
		<svg
			{width}
			height={HEIGHT}
			role="img"
			aria-label={label}
			onpointerleave={() => (active = null)}
		>
			<g transform={`translate(${MARGIN.left},${MARGIN.top})`}>
				{#each yColumn.ticks(TICKS) as tick (tick)}
					<line
						x1="0"
						x2={innerWidth}
						y1={yColumn(tick)}
						y2={yColumn(tick)}
						class={tick === 0 ? 'stroke-base-content/40' : 'stroke-base-300'}
						stroke-dasharray={tick === 0 ? undefined : '4 4'}
					/>
					<text
						x="-8"
						y={yColumn(tick)}
						text-anchor="end"
						dominant-baseline="middle"
						class="fill-base-content/70 text-xs"
					>
						{columnTick(tick)}
					</text>
				{/each}

				{#each yLine.ticks(TICKS) as tick (tick)}
					<text
						x={innerWidth + 8}
						y={yLine(tick)}
						text-anchor="start"
						dominant-baseline="middle"
						class="fill-base-content/70 text-xs"
					>
						{lineTick(tick)}
					</text>
				{/each}

				{#each dates as date, index (date)}
					{#if index % labelStep === 0}
						<text
							x={center(date)}
							y={innerHeight + 18}
							text-anchor="middle"
							class="fill-base-content/70 text-xs"
						>
							{monthLabel.format(shownDate(date))}
						</text>
					{/if}
				{/each}

				{#if active !== null}
					<rect
						x={slotX(active)}
						y="0"
						width={x.step()}
						height={innerHeight}
						class="fill-base-content/5"
					/>
				{/if}

				{#each dates as date (date)}
					{#each series as one (one.key)}
						{@const value = one.columns.get(date)}
						{#if value !== undefined}
							<rect
								x={(x(date) ?? 0) + (xSeries(one.key) ?? 0)}
								y={Math.min(yColumn(value), yColumn(0))}
								width={xSeries.bandwidth()}
								height={Math.abs(yColumn(value) - yColumn(0))}
								rx="2"
								class={[one.columnClass, 'opacity-80']}
							/>
						{/if}
					{/each}
				{/each}

				{#each series as one (one.key)}
					<path d={pathOf(one)} class={['fill-none', one.lineClass]} stroke-width="2.5" />
					{#each dates as date (date)}
						{@const value = one.line.get(date)}
						{#if value !== undefined}
							<circle
								cx={center(date)}
								cy={yLine(value)}
								r={active === date ? 4.5 : 3}
								class={['fill-base-100', one.lineClass]}
								stroke-width="2"
							/>
						{/if}
					{/each}
				{/each}

				{#each dates as date (date)}
					<rect
						role="presentation"
						x={slotX(date)}
						y="0"
						width={x.step()}
						height={innerHeight}
						class="fill-transparent"
						onpointerenter={() => (active = date)}
					/>
				{/each}
			</g>
		</svg>
	{/if}

	{#if active !== null}
		<div
			class="bg-base-100 border-base-300 rounded-box pointer-events-none absolute top-8 z-10 border p-2 text-xs shadow-md"
			style:left={`${tooltipLeft}px`}
			style:width={`${tooltipWidth}px`}
		>
			<div class="mb-1 font-medium">{dayLabel.format(shownDate(active))}</div>
			{@render tooltip(active)}
		</div>
	{/if}
</div>
