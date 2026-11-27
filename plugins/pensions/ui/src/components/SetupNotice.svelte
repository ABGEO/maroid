<script lang="ts">
	import type { PluginHost } from '@maroid/plugin-sdk';

	import type { PageState } from '../lib/state';

	let { host, kind }: { host: PluginHost; kind: Exclude<PageState, 'ready'> } = $props();
</script>

{#if kind === 'not_set_up'}
	<div role="alert" class="alert alert-info alert-soft max-w-xl">
		<span>Connect the pension account to see the data that Maroid collects.</span>
		<a class="btn btn-sm" href={host.href('/account')}>Open the account</a>
	</div>
{:else}
	<div class="empty-state">
		No data yet. The first collection runs on the first day of the next month.
	</div>
{/if}
