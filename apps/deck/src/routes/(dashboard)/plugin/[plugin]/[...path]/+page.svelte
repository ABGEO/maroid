<script lang="ts">
	import type { PageProps } from './$types';
	import type { PluginHost } from '@maroid/plugin-sdk';
	import { userState } from '$lib/state/user.svelte';

	let { data }: PageProps = $props();
	let target: HTMLElement | undefined = $state();

	const host: PluginHost = userState;

	$effect(() => {
		if (!target) {
			return;
		}

		return data.mount(target, { host });
	});
</script>

<div bind:this={target}></div>
