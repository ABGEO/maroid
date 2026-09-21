<script lang="ts">
	import type { PageProps } from './$types';
	import SettingsForm from '$lib/components/settings/SettingsForm.svelte';
	import { pluginState } from '$lib/state/plugins.svelte';
	import { uiOf } from '$lib/plugins/capabilities';

	let { data }: PageProps = $props();

	const plugin = $derived(pluginState.plugins.find((p) => p.id === data.pluginId));
	const name = $derived(uiOf(plugin)?.name ?? data.pluginId.split('.').pop() ?? data.pluginId);
</script>

<div class="max-w-4xl">
	<h1 class="font-display text-[42px] leading-[1.05] tracking-tight">{name}</h1>
	<p class="text-base-content/50 mt-1 font-mono text-[11px]">{data.pluginId}</p>
	<p class="text-base-content/60 mt-3 text-sm">
		These values belong to you. Nobody else reads them, and a secret never leaves the hub once you
		store it.
	</p>

	<div class="mt-8">
		<SettingsForm pluginId={data.pluginId} />
	</div>
</div>
