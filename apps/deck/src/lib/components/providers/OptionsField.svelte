<script lang="ts">
	import FieldError from '$lib/components/providers/FieldError.svelte';

	interface Props {
		value: string;
		errors: Record<string, string>;
		hint: string;
		rows?: number;
		disabled?: boolean;
		placeholder?: string;
	}

	let {
		value = $bindable(),
		errors,
		hint,
		rows = 4,
		disabled = false,
		placeholder
	}: Props = $props();

	const PREFIX = '/options/';

	const nested = $derived(Object.entries(errors).filter(([pointer]) => pointer.startsWith(PREFIX)));
</script>

<label class="mt-2 flex flex-col gap-1">
	<span class="label">Options</span>
	<textarea
		class="textarea textarea-sm w-full font-mono text-xs"
		{rows}
		bind:value
		{disabled}
		{placeholder}
	></textarea>
	<span class="label whitespace-normal">{hint}</span>
	<FieldError {errors} pointer="/options" />
	{#each nested as [pointer, detail] (pointer)}
		<span class="text-error text-xs">{pointer.slice(PREFIX.length)}: {detail}</span>
	{/each}
</label>
