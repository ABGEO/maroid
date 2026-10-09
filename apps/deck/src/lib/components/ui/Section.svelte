<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title: string;
		danger?: boolean;
		children: Snippet;
	}

	let { title, danger = false, children }: Props = $props();
</script>

<section class="section">
	<h2 class={['section-title', danger && 'text-error']}>{title}</h2>
	{@render children()}
</section>
