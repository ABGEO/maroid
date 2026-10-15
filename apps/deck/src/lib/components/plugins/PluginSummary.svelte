<script lang="ts">
	import type { Snippet } from 'svelte';

	import type { ResolvedPathname } from '$app/types';

	import type { Plugin } from '$lib/api';
	import { capabilitiesOf, countOf, displayNameOf, labelOf } from '$lib/plugins/capabilities';

	interface Props {
		id: string;
		plugin?: Plugin;
		detailed?: boolean;
		/** The detail page of the plugin. With it, the name is a link. */
		href?: ResolvedPathname;
		badges?: Snippet;
		children?: Snippet;
	}

	let { id, plugin, detailed = false, href, badges, children }: Props = $props();

	const capabilities = $derived(detailed && plugin ? capabilitiesOf(plugin) : []);
</script>

<div class="min-w-0 flex-1">
	<div class="flex items-center gap-2">
		{#if href}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the caller resolves href -->
			<a class="link link-hover font-medium" {href}>{displayNameOf(plugin, id)}</a>
		{:else}
			<span class="font-medium">{displayNameOf(plugin, id)}</span>
		{/if}
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
