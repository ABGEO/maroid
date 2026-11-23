<script lang="ts">
	import type { PluginHost } from '@maroid/plugin-sdk';

	let { host }: { host: PluginHost } = $props();

	function open(event: MouseEvent): void {
		event.preventDefault();
		host.navigate('/preferences');
	}
</script>

<div role="status" class="alert alert-info alert-soft mb-4">
	<span>Pulse is off, so no check-in fires.</span>
	<a class="btn btn-sm" href={host.href('/preferences')} onclick={open}>Open the preferences</a>
</div>
