<script lang="ts">
	import { page } from '$app/state';

	import type { Workspace } from '$lib/api';
	import { crumbsFor, needsPlugins } from '$lib/navigation/breadcrumbs';
	import { enabledState } from '$lib/state/plugins.svelte';
	import { workspaceState } from '$lib/state/workspaces.svelte';

	const loading = $derived(enabledState.status === 'idle' || enabledState.status === 'loading');
	const pending = $derived(loading && needsPlugins(page.route.id));

	// A workspace of another person is absent from the list, so its name comes from the
	// read of the page.
	const loaded = $derived(page.data.workspace as Workspace | undefined);
	const workspaces = $derived(
		loaded && !workspaceState.workspaces.some((workspace) => workspace.id === loaded.id)
			? [...workspaceState.workspaces, loaded]
			: workspaceState.workspaces
	);
	const crumbs = $derived(crumbsFor(page.route.id, page.params, enabledState.plugins, workspaces));
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
