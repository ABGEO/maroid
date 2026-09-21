<script lang="ts">
	import Breadcrumbs from '$lib/components/layout/Breadcrumbs.svelte';
	import Footer from '$lib/components/layout/Footer.svelte';
	import Sidebar from '$lib/components/layout/Sidebar.svelte';
	import Header from '$lib/components/layout/Header.svelte';

	import { loadPlugins, pluginState } from '$lib/state/plugins.svelte';

	let { children } = $props();

	$effect(() => {
		if (pluginState.status === 'idle') {
			loadPlugins();
		}
	});
</script>

<div class="drawer lg:drawer-open bg-base-100 text-base-content">
	<!-- @todo: store value in localStorage -->
	<input id="app-drawer" type="checkbox" class="drawer-toggle" checked />

	<div class="drawer-content flex min-h-screen flex-col pt-14 pb-10">
		<Header />

		<main class="paper-bg min-w-0 flex-1 overflow-y-auto">
			<div class="px-8 pt-5">
				<Breadcrumbs />
			</div>

			<section class="px-8 pt-8 pb-8">
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
