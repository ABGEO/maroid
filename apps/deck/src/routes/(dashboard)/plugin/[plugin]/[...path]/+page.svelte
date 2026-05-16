<script lang="ts">
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let target: HTMLElement | undefined = $state();

	$effect(() => {
		if (!target) {
			return;
		}

		return data.mount(target);
	});
</script>

<div bind:this={target}></div>
