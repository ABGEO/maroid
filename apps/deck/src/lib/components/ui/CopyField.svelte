<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		value: string;
		label: string;
		children?: Snippet;
	}

	let { value, label, children }: Props = $props();

	let copiedValue = $state<string | null>(null);

	const copied = $derived(copiedValue === value);

	async function copy(): Promise<void> {
		try {
			await navigator.clipboard.writeText(value);
			copiedValue = value;
		} catch (error) {
			console.error('Failed to copy the value', error);
		}
	}
</script>

<div class="flex w-full gap-2">
	<input class="input input-sm w-full font-mono text-xs" readonly {value} aria-label={label} />
	<button type="button" class="btn btn-sm" onclick={copy}>
		{copied ? 'Copied' : 'Copy'}
	</button>
	{@render children?.()}
</div>
