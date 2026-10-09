<script lang="ts">
	interface Props {
		count?: number;
		row?: string;
	}

	let { count = 3, row = 'h-12' }: Props = $props();

	const rows = $derived(Array.from({ length: count }, (_, index) => index));
</script>

<div class="flex flex-col gap-2" aria-hidden="true">
	{#each rows as index (index)}
		<span class={['skeleton w-full', row]}></span>
	{/each}
</div>
