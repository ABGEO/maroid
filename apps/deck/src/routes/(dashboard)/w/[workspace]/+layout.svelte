<script lang="ts">
	import { rememberWorkspace } from '$lib/state/workspaces.svelte';

	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();

	$effect(() => {
		rememberWorkspace(data.workspace.id);
	});
</script>

{@render children()}
