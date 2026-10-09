<script lang="ts">
	import type { Snippet } from 'svelte';

	import type { Plugin } from '$lib/api';
	import { capabilitiesOf, countOf, displayNameOf, labelOf } from '$lib/plugins/capabilities';

	interface Props {
		id: string;
		plugin?: Plugin;
		detailed?: boolean;
		badges?: Snippet;
		children?: Snippet;
	}

	let { id, plugin, detailed = false, badges, children }: Props = $props();

	const capabilities = $derived(detailed && plugin ? capabilitiesOf(plugin) : []);
</script>

<div class="min-w-0 flex-1">
	<div class="flex items-center gap-2">
		<span class="font-medium">{displayNameOf(plugin, id)}</span>
		{#if detailed && plugin}
			<span class="badge badge-ghost badge-xs font-mono">v{plugin.version}</span>
		{/if}
		{@render badges?.()}
	</div>
	<div class="meta truncate font-mono">{id}</div>
	{#if plugin?.description}
		<p class="text-base-content/70 mt-1 text-sm">{plugin.description}</p>
	{/if}
	{#if plugin && capabilities.length > 0}
		<div class="mt-1.5 flex flex-wrap gap-1">
			{#each capabilities as name (name)}
				<span class="badge badge-soft badge-xs">
					{labelOf(name)}{countOf(plugin, name) > 0 ? `: ${countOf(plugin, name)}` : ''}
				</span>
			{/each}
		</div>
	{/if}
	{@render children?.()}
</div>
