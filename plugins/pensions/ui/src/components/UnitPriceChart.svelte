<script lang="ts">
	import { ColumnLineChart, type ChartSeries } from '@maroid/ui';

	import type { PortfolioShare, UnitPrice } from '../api';
	import { formatUnits } from '../lib/format';

	const COLORS = [
		{ line: 'stroke-primary', swatch: 'bg-primary' },
		{ line: 'stroke-secondary', swatch: 'bg-secondary' },
		{ line: 'stroke-accent', swatch: 'bg-accent' },
		{ line: 'stroke-info', swatch: 'bg-info' },
		{ line: 'stroke-success', swatch: 'bg-success' },
		{ line: 'stroke-warning', swatch: 'bg-warning' }
	];

	let { shares, prices }: { shares: PortfolioShare[]; prices: UnitPrice[] } = $props();

	const portfolios = $derived(
		shares.map((share, index) => ({
			share,
			color: COLORS[index % COLORS.length],
			line: new Map(
				prices
					.filter((price) => price.portfolio_id === share.portfolio_id)
					.map((price) => [price.priced_on, price.unit_price])
			)
		}))
	);

	const series = $derived<ChartSeries[]>(
		portfolios.map((portfolio) => ({
			key: String(portfolio.share.portfolio_id),
			columns: new Map(),
			line: portfolio.line,
			columnClass: '',
			lineClass: portfolio.color.line
		}))
	);
</script>

<ColumnLineChart
	{series}
	label="The unit price of each portfolio of the account, one point each week."
>
	{#snippet legend()}
		{#each portfolios as portfolio (portfolio.share.portfolio_id)}
			<li class="flex items-center gap-1.5">
				<span class={['inline-block h-0.5 w-3', portfolio.color.swatch]}></span>
				<span>{portfolio.share.ticker}</span>
				{#if portfolio.line.size === 0}
					<span class="text-base-content/70">(no price in the period)</span>
				{/if}
			</li>
		{/each}
	{/snippet}

	{#snippet tooltip(date: string)}
		{#each portfolios as portfolio (portfolio.share.portfolio_id)}
			{@const price = portfolio.line.get(date)}
			{#if price !== undefined}
				<div class="flex items-center gap-1.5">
					<span class={['inline-block size-2 rounded-sm', portfolio.color.swatch]}></span>
					<span>{portfolio.share.ticker}</span>
					<span class="ml-auto tabular-nums">{formatUnits(price)}</span>
				</div>
			{/if}
		{/each}
	{/snippet}
</ColumnLineChart>
