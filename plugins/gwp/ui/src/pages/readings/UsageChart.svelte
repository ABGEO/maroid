<script lang="ts">
  import { scaleBand, scaleLinear } from 'd3-scale';
  import { curveMonotoneX, line } from 'd3-shape';

  import type { Reading } from '../../api';
  import { chartSeries, meterTone } from '../../lib/chart';

  const HEIGHT = 320;
  const MARGIN = { top: 12, right: 52, bottom: 28, left: 52 };
  const TICKS = 5;
  const MIN_LABEL_WIDTH = 56;
  const TOOLTIP_WIDTH = 176;

  let { readings }: { readings: Reading[] } = $props();

  let width = $state(0);
  let active = $state<string | null>(null);

  const monthLabel = new Intl.DateTimeFormat(undefined, { month: 'short', year: '2-digit', timeZone: 'UTC' });
  const dayLabel = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeZone: 'UTC' });

  const series = $derived(chartSeries(readings));
  const innerWidth = $derived(Math.max(0, width - MARGIN.left - MARGIN.right));
  const innerHeight = HEIGHT - MARGIN.top - MARGIN.bottom;

  const x = $derived(scaleBand<string>().domain(series.dates).range([0, innerWidth]).paddingInner(0.25).paddingOuter(0.1));
  const xMeter = $derived(scaleBand<string>().domain(series.meters).range([0, x.bandwidth()]).padding(0.08));

  const usages = $derived(
    series.dates.flatMap((date) =>
      [...(series.points.get(date)?.values() ?? [])].flatMap((point) => (point.usage === undefined ? [] : [point.usage]))
    )
  );
  const values = $derived(readings.map((reading) => reading.value));

  const yUsage = $derived(
    scaleLinear()
      .domain([Math.min(0, ...usages), Math.max(0, ...usages)])
      .nice(TICKS)
      .range([innerHeight, 0])
  );
  const yValue = $derived(
    scaleLinear()
      .domain([Math.min(...values), Math.max(...values)])
      .nice(TICKS)
      .range([innerHeight, 0])
  );
  const usageTick = $derived(yUsage.tickFormat(TICKS));
  const valueTick = $derived(yValue.tickFormat(TICKS));

  // A label fits in about 56 pixels, so a narrow chart names every second or third month.
  const labelStep = $derived(Math.max(1, Math.ceil(series.dates.length / Math.max(1, Math.floor(innerWidth / MIN_LABEL_WIDTH)))));

  function center(date: string): number {
    return (x(date) ?? 0) + x.bandwidth() / 2;
  }

  function columnOf(date: string, serial: string, usage: number) {
    const top = Math.min(yUsage(usage), yUsage(0));

    return {
      x: (x(date) ?? 0) + (xMeter(serial) ?? 0),
      y: top,
      width: xMeter.bandwidth(),
      height: Math.abs(yUsage(usage) - yUsage(0))
    };
  }

  function pathOf(serial: string): string {
    const points = series.dates.flatMap((date) => {
      const point = series.points.get(date)?.get(serial);

      return point === undefined ? [] : [{ date, value: point.value }];
    });

    return (
      line<{ date: string; value: number }>()
        .x((point) => center(point.date))
        .y((point) => yValue(point.value))
        .curve(curveMonotoneX)(points) ?? ''
    );
  }

  function shownDate(date: string): Date {
    return new Date(`${date}T00:00:00Z`);
  }

  function volume(cubicMetres: number): string {
    return cubicMetres.toFixed(1);
  }

  const tooltipLeft = $derived(
    active === null ? 0 : Math.min(Math.max(0, MARGIN.left + center(active) - TOOLTIP_WIDTH / 2), width - TOOLTIP_WIDTH)
  );
</script>

<div class="relative" bind:clientWidth={width}>
  <ul class="text-base-content/70 mb-2 flex flex-wrap gap-x-4 gap-y-1 text-xs">
    {#each series.meters as serial, index (serial)}
      {@const tone = meterTone(index)}
      <li class="flex items-center gap-1.5">
        <span class={['inline-block size-2.5 rounded-sm', tone.swatch]}></span>
        <span class={['inline-block h-0.5 w-3', tone.swatch]}></span>
        <span class="font-mono">{serial}</span>
      </li>
    {/each}
    <li class="ml-auto">Columns: usage, m³ (left). Lines: meter value, m³ (right).</li>
  </ul>

  {#if innerWidth > 0}
    <svg
      {width}
      height={HEIGHT}
      role="img"
      aria-label="The usage and the value of each meter over the last 12 months. The table below lists the same readings."
      onpointerleave={() => (active = null)}
    >
      <g transform={`translate(${MARGIN.left},${MARGIN.top})`}>
        {#each yUsage.ticks(TICKS) as tick (tick)}
          <line
            x1="0"
            x2={innerWidth}
            y1={yUsage(tick)}
            y2={yUsage(tick)}
            class={tick === 0 ? 'stroke-base-content/40' : 'stroke-base-300'}
            stroke-dasharray={tick === 0 ? undefined : '4 4'}
          />
          <text x="-8" y={yUsage(tick)} text-anchor="end" dominant-baseline="middle" class="fill-base-content/70 text-xs">
            {usageTick(tick)}
          </text>
        {/each}

        {#each yValue.ticks(TICKS) as tick (tick)}
          <text
            x={innerWidth + 8}
            y={yValue(tick)}
            text-anchor="start"
            dominant-baseline="middle"
            class="fill-base-content/70 text-xs"
          >
            {valueTick(tick)}
          </text>
        {/each}

        {#each series.dates as date, index (date)}
          {#if index % labelStep === 0}
            <text x={center(date)} y={innerHeight + 18} text-anchor="middle" class="fill-base-content/70 text-xs">
              {monthLabel.format(shownDate(date))}
            </text>
          {/if}
        {/each}

        {#if active !== null}
          <rect
            x={(x(active) ?? 0) - (x.step() - x.bandwidth()) / 2}
            y="0"
            width={x.step()}
            height={innerHeight}
            class="fill-base-content/5"
          />
        {/if}

        {#each series.dates as date (date)}
          {#each series.meters as serial, index (serial)}
            {@const usage = series.points.get(date)?.get(serial)?.usage}
            {#if usage !== undefined}
              {@const column = columnOf(date, serial, usage)}
              <rect
                x={column.x}
                y={column.y}
                width={column.width}
                height={column.height}
                rx="2"
                class={[meterTone(index).column, 'opacity-80']}
              />
            {/if}
          {/each}
        {/each}

        {#each series.meters as serial, index (serial)}
          {@const tone = meterTone(index)}
          <path d={pathOf(serial)} class={['fill-none', tone.line]} stroke-width="2.5" />
          {#each series.dates as date (date)}
            {@const point = series.points.get(date)?.get(serial)}
            {#if point !== undefined}
              <circle
                cx={center(date)}
                cy={yValue(point.value)}
                r={active === date ? 4.5 : 3}
                class={['fill-base-100', tone.line]}
                stroke-width="2"
              />
            {/if}
          {/each}
        {/each}

        {#each series.dates as date (date)}
          <rect
            role="presentation"
            x={(x(date) ?? 0) - (x.step() - x.bandwidth()) / 2}
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
      style:width={`${TOOLTIP_WIDTH}px`}
    >
      <div class="mb-1 font-medium">{dayLabel.format(shownDate(active))}</div>
      {#each series.meters as serial, index (serial)}
        {@const point = series.points.get(active)?.get(serial)}
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
    </div>
  {/if}
</div>
