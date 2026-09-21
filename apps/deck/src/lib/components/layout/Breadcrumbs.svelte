<script lang="ts">
	import { page } from '$app/state';

	import { crumbsFor, needsPlugins } from '$lib/navigation/breadcrumbs';
	import { pluginState } from '$lib/state/plugins.svelte';

	const loading = $derived(pluginState.status === 'idle' || pluginState.status === 'loading');
	const pending = $derived(loading && needsPlugins(page.route.id));
	const crumbs = $derived(crumbsFor(page.route.id, page.params, pluginState.plugins));
</script>

{#if pending}
	<div class="flex h-4 items-center">
		<span class="skeleton h-3.5 w-56" aria-hidden="true"></span>
	</div>
{:else}
	<nav
		class="breadcrumbs text-base-content/55 py-0 font-mono text-[11px] leading-4"
		aria-label="Breadcrumb"
	>
		<ul>
			{#each crumbs as crumb, index (index)}
				<li>
					{#if crumb.href && index < crumbs.length - 1}
						<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- crumbsFor resolves every target -->
						<a class="hover:text-base-content" href={crumb.href}>{crumb.label}</a>
					{:else if index === crumbs.length - 1}
						<span class="text-base-content" aria-current="page">{crumb.label}</span>
					{:else}
						<span>{crumb.label}</span>
					{/if}
				</li>
			{/each}
		</ul>
	</nav>
{/if}
