<script lang="ts">
	import type { PluginHost } from '@maroid/plugin-sdk';

	let { host, title, lead }: { host: PluginHost; title: string; lead: string } = $props();
</script>

<header class="page-header">
	<div>
		<h1 class="page-title">{title}</h1>
		<p class="page-lead">{lead}</p>
		<p class="meta mt-1">
			{#if host.timeZone}
				The schedule uses {host.timeZone}.
			{:else}
				The schedule uses UTC. Set your time zone in your
				<a class="link" href="/profile">profile</a>.
			{/if}
		</p>
	</div>
</header>
