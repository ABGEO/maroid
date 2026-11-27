<script lang="ts">
	import { PERIODS, type Period } from '../lib/period';

	let { value = $bindable(), label }: { value: Period; label: string } = $props();

	const name = $props.id();
</script>

<div class="join" role="radiogroup" aria-label={label}>
	{#each PERIODS as period (period.value)}
		<input
			type="radio"
			{name}
			value={period.value}
			aria-label={period.label}
			class={['join-item btn btn-sm', value === period.value ? 'btn-primary' : 'btn-outline']}
			bind:group={value}
		/>
	{/each}
</div>
