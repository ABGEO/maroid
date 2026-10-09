<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		kind: 'error' | 'success' | 'info';
		class?: string;
		children: Snippet;
	}

	let { kind, class: className = '', children }: Props = $props();

	const KIND_CLASS = {
		error: 'alert-error',
		success: 'alert-success',
		info: 'alert-info'
	};
</script>

<div
	role={kind === 'error' ? 'alert' : 'status'}
	class={['alert alert-soft', KIND_CLASS[kind], className]}
>
	{@render children()}
</div>
