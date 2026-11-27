<script lang="ts">
	import type { LotUsage } from '../api';
	import { formatMoney } from '../lib/format';

	let { lots }: { lots: LotUsage[] } = $props();
</script>

{#if lots.length === 0}
	<div class="empty-state">No session in the period.</div>
{:else}
	<ol class="divide-base-300 flex flex-col divide-y">
		{#each lots as lot (lot.lot_number)}
			<li class="flex flex-wrap items-baseline justify-between gap-2 py-2">
				<span>
					<span class="font-medium">{lot.lot_number}</span>
					<span class="text-sm opacity-70">{lot.address}</span>
				</span>
				<span class="text-sm tabular-nums">
					{lot.count}
					{lot.count === 1 ? 'session' : 'sessions'}, {formatMoney(lot.amount)}
				</span>
			</li>
		{/each}
	</ol>
{/if}
