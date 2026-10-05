<script lang="ts">
	import { afterNavigate } from '$app/navigation';
	import { MediaQuery } from 'svelte/reactivity';

	import Breadcrumbs from '$lib/components/layout/Breadcrumbs.svelte';
	import Footer from '$lib/components/layout/Footer.svelte';
	import Sidebar from '$lib/components/layout/Sidebar.svelte';
	import Header from '$lib/components/layout/Header.svelte';

	import { loadPlugins, pluginState } from '$lib/state/plugins.svelte';
	import { loadWorkspaces, workspaceState } from '$lib/state/workspaces.svelte';

	let { children } = $props();

	const STORAGE_KEY = 'maroid.sidebar.expanded';

	// The breakpoint of `lg`, where the sidebar stays beside the page.
	const wide = new MediaQuery('(min-width: 64rem)');

	let expanded = $state(rememberedExpanded());
	let overlayOpen = $state(false);

	const open = $derived(wide.current ? expanded : overlayOpen);

	function rememberedExpanded(): boolean {
		try {
			return localStorage.getItem(STORAGE_KEY) !== 'false';
		} catch {
			return true;
		}
	}

	function setOpen(next: boolean): void {
		if (!wide.current) {
			overlayOpen = next;

			return;
		}

		expanded = next;

		try {
			localStorage.setItem(STORAGE_KEY, String(next));
		} catch {
			// The storage is unavailable in a private window. The sidebar opens expanded.
		}
	}

	function closeOnEscape(event: KeyboardEvent): void {
		if (event.key === 'Escape' && overlayOpen) {
			overlayOpen = false;
		}
	}

	afterNavigate(() => {
		overlayOpen = false;
	});

	// The page behind the open overlay must not scroll under the finger.
	$effect(() => {
		document.documentElement.style.overflow = !wide.current && overlayOpen ? 'hidden' : '';
	});

	$effect(() => {
		if (pluginState.status === 'idle') {
			loadPlugins();
		}
	});

	$effect(() => {
		if (workspaceState.status === 'idle') {
			loadWorkspaces();
		}
	});
</script>

<svelte:window onkeydown={closeOnEscape} />

<div class="drawer lg:drawer-open bg-base-100 text-base-content">
	<input
		id="app-drawer"
		type="checkbox"
		class="drawer-toggle"
		checked={open}
		onchange={(event) => setOpen(event.currentTarget.checked)}
	/>

	<div class="drawer-content flex min-h-screen flex-col pt-14 pb-10">
		<Header />

		<main class="paper-bg min-w-0 flex-1 overflow-y-auto">
			<div class="px-4 pt-5 sm:px-8">
				<Breadcrumbs />
			</div>

			<section class="px-4 pt-6 pb-8 sm:px-8 sm:pt-8">
				{@render children?.()}
			</section>
		</main>

		<Footer />
	</div>

	<div class="drawer-side is-drawer-close:overflow-visible z-40">
		<label for="app-drawer" aria-label="close sidebar" class="drawer-overlay"></label>

		<Sidebar />
	</div>
</div>
