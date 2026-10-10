<script lang="ts">
	import { resolve } from '$app/paths';

	import type { Provider } from '$lib/api';
	import { presetLabel } from '$lib/providers/form';

	interface Props {
		providers: Provider[];
	}

	let { providers }: Props = $props();
</script>

<ul class="list panel">
	{#each providers as provider (provider.id)}
		<li class="list-row flex flex-wrap items-center justify-between gap-3">
			<span class="min-w-0">
				<a
					class="link link-hover text-sm font-semibold"
					href={resolve('/(dashboard)/admin/providers/[provider]', { provider: provider.id })}
				>
					{provider.name}
				</a>
				<span class="meta block truncate font-mono">
					{provider.id}
				</span>
			</span>
			<span class="flex items-center gap-2">
				<span class="badge badge-outline badge-sm">{presetLabel(provider)}</span>
				<span class="text-base-content/60 w-20 text-right text-xs">
					{provider.identity_count}
					{provider.identity_count === 1 ? 'identity' : 'identities'}
				</span>
			</span>
		</li>
	{/each}
</ul>
