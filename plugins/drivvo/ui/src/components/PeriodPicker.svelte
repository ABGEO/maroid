<script lang="ts">
	import type { Period } from '../api';
	import { PERIOD_CHOICES, type PeriodChoice } from '../lib/period';

	let {
		choice = $bindable(),
		range = $bindable()
	}: {
		choice: PeriodChoice;
		range: Period;
	} = $props();
</script>

<select class="select select-sm w-auto" aria-label="Period" bind:value={choice}>
	{#each PERIOD_CHOICES as option (option.value)}
		<option value={option.value}>{option.label}</option>
	{/each}
</select>

{#if choice === 'range'}
	<input
		type="date"
		class="input input-sm w-auto"
		aria-label="From"
		max={range.to}
		bind:value={range.from}
	/>
	<input
		type="date"
		class="input input-sm w-auto"
		aria-label="To"
		min={range.from}
		bind:value={range.to}
	/>
{/if}
