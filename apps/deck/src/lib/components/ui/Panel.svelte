<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		class?: string;
		children: Snippet;
	}

	let { class: className = '', children }: Props = $props();
</script>

<fieldset class={['fieldset panel w-full min-w-0 p-5', className]}>
	{@render children()}
</fieldset>
