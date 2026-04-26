<script lang="ts">
	import { pluginState } from '$lib/state/plugins.svelte';

	const pluginsCountLabel = $derived.by(() => {
		if (pluginState.status !== 'ready') {
			return '- plugins running';
		}

		const pluginsCount = pluginState.plugins.length;
		return `${pluginsCount} ${pluginsCount === 1 ? 'plugin' : 'plugins'} running`;
	});
</script>

<footer
	class="footer sm:footer-horizontal border-base-300 bg-base-100/90 text-base-content/55 fixed inset-x-0 bottom-0 z-50 flex h-10 items-center justify-between border-t px-4 font-mono text-[11px] backdrop-blur"
>
	<aside class="flex items-center gap-3">
		<span>© 2026 Maroid</span>
		<span class="hidden sm:inline">·</span>
		<span class="hidden sm:inline">self-hosted</span>
		<span class="hidden md:inline">·</span>
		<span class="hidden md:inline">your data stays with you</span>
	</aside>
	<aside class="flex items-center gap-3">
		<span class="hidden md:inline">v0.1.0</span>
		<span class="hidden md:inline">·</span>
		<span class="hidden md:inline">uptime 14d 02h</span>
		<span class="hidden md:inline">·</span>
		<span class="flex items-center gap-1.5">
			<span class="bg-success h-1.5 w-1.5 rounded-full"></span>
			{pluginsCountLabel}
		</span>
	</aside>
</footer>
