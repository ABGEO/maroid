<script lang="ts">
  import { scaleBand, scaleLinear } from 'd3-scale';
  import { curveMonotoneX, line } from 'd3-shape';

  import type { Transaction } from '../../api';
  import { chartPoints, type ChartPoint } from '../../lib/chart';

  const HEIGHT = 320;
  const MARGIN = { top: 12, right: 52, bottom: 28, left: 52 };
  const TICKS = 5;
  const MIN_LABEL_WIDTH = 56;
  const TOOLTIP_WIDTH = 160;

  let { transactions }: { transactions: Transaction[] } = $props();

  let width = $state(0);
  let active = $state<string | null>(null);

  const monthLabel = new Intl.DateTimeFormat(undefined, { month: 'short', year: '2-digit', timeZone: 'UTC' });
  const dayLabel = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeZone: 'UTC' });

  const points = $derived(chartPoints(transactions));
  const dates = $derived(points.map((point) => point.date));
  const activePoint = $derived(points.find((point) => point.date === active));
  const innerWidth = $derived(Math.max(0, width - MARGIN.left - MARGIN.right));
  const innerHeight = HEIGHT - MARGIN.top - MARGIN.bottom;

  const x = $derived(scaleBand<string>().domain(dates).range([0, innerWidth]).paddingInner(0.25).paddingOuter(0.1));

  const yUsage = $derived(
    scaleLinear()
      .domain([0, Math.max(0, ...points.map((point) => point.usage))])
      .nice(TICKS)
      .range([innerHeight, 0])
  );
  const yAmount = $derived(
    scaleLinear()
      .domain([0, Math.max(0, ...points.map((point) => point.amount))])
      .nice(TICKS)
      .range([innerHeight, 0])
  );
  const usageTick = $derived(yUsage.tickFormat(TICKS));
  const amountTick = $derived(yAmount.tickFormat(TICKS));

  // A label fits in about 56 pixels, so a narrow chart names every second or third month.
  const labelStep = $derived(Math.max(1, Math.ceil(dates.length / Math.max(1, Math.floor(innerWidth / MIN_LABEL_WIDTH)))));

  function center(date: string): number {
    return (x(date) ?? 0) + x.bandwidth() / 2;
  }

  const amountPath = $derived(
    line<ChartPoint>()
      .x((point) => center(point.date))
      .y((point) => yAmount(point.amount))
      .curve(curveMonotoneX)(points) ?? ''
  );

  function shownDate(date: string): Date {
    return new Date(`${date}T00:00:00Z`);
  }

  const tooltipLeft = $derived(
    active === null ? 0 : Math.min(Math.max(0, MARGIN.left + center(active) - TOOLTIP_WIDTH / 2), width - TOOLTIP_WIDTH)
  );
</script>

<div class="relative" bind:clientWidth={width}>
  <ul class="text-base-content/70 mb-2 flex flex-wrap gap-x-4 gap-y-1 text-xs">
    <li class="flex items-center gap-1.5">
      <span class="bg-primary inline-block size-2.5 rounded-sm"></span>
      <span>Usage, m³ (left)</span>
    </li>
    <li class="flex items-center gap-1.5">
      <span class="bg-accent inline-block h-0.5 w-3"></span>
      <span>Amount, ₾ (right)</span>
    </li>
  </ul>

  {#if innerWidth > 0}
    <svg
      {width}
      height={HEIGHT}
      role="img"
      aria-label="The usage and the amount of each reading over the last 12 months. The table lists the same transactions."
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

        {#each yAmount.ticks(TICKS) as tick (tick)}
          <text
            x={innerWidth + 8}
            y={yAmount(tick)}
            text-anchor="start"
            dominant-baseline="middle"
            class="fill-base-content/70 text-xs"
          >
            {amountTick(tick)}
          </text>
        {/each}

        {#each dates as date, index (date)}
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

        {#each points as point (point.date)}
          <rect
            x={x(point.date) ?? 0}
            y={yUsage(point.usage)}
            width={x.bandwidth()}
            height={Math.max(0, yUsage(0) - yUsage(point.usage))}
            rx="2"
            class="fill-primary opacity-80"
          />
        {/each}

        <path d={amountPath} class="stroke-accent fill-none" stroke-width="2.5" />
        {#each points as point (point.date)}
          <circle
            cx={center(point.date)}
            cy={yAmount(point.amount)}
            r={active === point.date ? 4.5 : 3}
            class="fill-base-100 stroke-accent"
            stroke-width="2"
          />
        {/each}

        {#each dates as date (date)}
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

  {#if activePoint !== undefined}
    <div
      class="bg-base-100 border-base-300 rounded-box pointer-events-none absolute top-8 z-10 border p-2 text-xs shadow-md"
      style:left={`${tooltipLeft}px`}
      style:width={`${TOOLTIP_WIDTH}px`}
    >
      <div class="mb-1 font-medium">{dayLabel.format(shownDate(activePoint.date))}</div>
      <div class="flex items-center gap-1.5">
        <span class="bg-primary inline-block size-2 rounded-sm"></span>
        <span>Usage</span>
        <span class="ml-auto tabular-nums">{activePoint.usage.toFixed(0)} m³</span>
      </div>
      <div class="flex items-center gap-1.5">
        <span class="bg-accent inline-block size-2 rounded-sm"></span>
        <span>Amount</span>
        <span class="ml-auto tabular-nums">{activePoint.amount.toFixed(2)} ₾</span>
      </div>
    </div>
  {/if}
</div>
