<script lang="ts">
	import type { PluginHost } from '@maroid/plugin-sdk';

	let { host }: { host: PluginHost } = $props();
</script>

<div role="status" class="alert alert-info alert-soft max-w-xl">
	<span>
		No data yet. Connect the parking account on the Account page. The collection runs each hour.
	</span>
	<a class="btn btn-sm" href={host.href('/account')}>Open the account</a>
</div>
