<script lang="ts">
	import { resolve } from '$app/paths';

	import { api, type Plugin } from '$lib/api';
	import { pluginState } from '$lib/state/plugins.svelte';
	import { fieldsOf, missingFields } from '$lib/settings/schema';
	import {
		capabilitiesOf,
		countOf,
		displayNameOf,
		hasCapability,
		labelOf
	} from '$lib/plugins/capabilities';

	type Health = 'complete' | 'incomplete';

	let health = $state<Record<string, Health>>({});

	async function check(plugin: Plugin): Promise<void> {
		try {
			const [schema, values] = await Promise.all([
				api.settings.schema(plugin.id),
				api.settings.read(plugin.id)
			]);

			if (schema === null || values === null) {
				return;
			}

			health[plugin.id] =
				missingFields(fieldsOf(schema), values).length > 0 ? 'incomplete' : 'complete';
		} catch (error) {
			console.error('Failed to read the settings of a plugin', error);
		}
	}

	$effect(() => {
		for (const plugin of pluginState.plugins.filter((p) => hasCapability(p, 'settings'))) {
			void check(plugin);
		}
	});
</script>

<div class="max-w-4xl">
	<h1 class="font-display text-[42px] leading-[1.05] tracking-tight">Plugins</h1>
	<p class="text-base-content/60 mt-1 text-sm">
		Every plugin that the hub loaded. A plugin that declares settings carries a configure action.
	</p>

	{#if pluginState.status === 'idle' || pluginState.status === 'loading'}
		<div class="mt-8 flex flex-col gap-2">
			{#each [0, 1, 2] as i (i)}
				<span class="skeleton h-16 w-full" aria-hidden="true"></span>
			{/each}
		</div>
	{:else if pluginState.status === 'error'}
		<div class="alert alert-error mt-8">
			<span>Failed to load the plugins.</span>
		</div>
	{:else if pluginState.plugins.length === 0}
		<div class="border-base-300 text-base-content/60 mt-8 rounded-md border border-dashed p-6">
			The hub loaded no plugin.
		</div>
	{:else}
		<ul class="border-base-300 divide-base-300 mt-8 divide-y rounded-md border">
			{#each pluginState.plugins as plugin (plugin.id)}
				<li class="flex items-center gap-4 px-4 py-3">
					<div class="min-w-0 flex-1">
						<div class="flex items-center gap-2">
							<span class="text-[15px] font-semibold">{displayNameOf(plugin, plugin.id)}</span>
							<span class="badge badge-ghost badge-xs font-mono">v{plugin.version}</span>
							{#if health[plugin.id] === 'incomplete'}
								<span class="badge badge-warning badge-xs">Needs attention</span>
							{/if}
						</div>
						<div class="text-base-content/50 truncate font-mono text-[11px]">{plugin.id}</div>
						{#if capabilitiesOf(plugin).length > 0}
							<div class="mt-1.5 flex flex-wrap gap-1">
								{#each capabilitiesOf(plugin) as name (name)}
									<span class="badge badge-soft badge-xs">
										{labelOf(name)}{countOf(plugin, name) > 0 ? `: ${countOf(plugin, name)}` : ''}
									</span>
								{/each}
							</div>
						{/if}
					</div>

					{#if hasCapability(plugin, 'settings')}
						<a
							class="btn btn-ghost btn-sm"
							href={resolve('/(dashboard)/plugins/[plugin]/settings', { plugin: plugin.id })}
						>
							<svg
								xmlns="http://www.w3.org/2000/svg"
								width="16"
								height="16"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="1.75"
							>
								<circle cx="12" cy="12" r="3" />
								<path
									d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.6a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"
								/>
							</svg>
							Configure
						</a>
					{:else}
						<span class="text-base-content/35 font-mono text-[11px]">No settings</span>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</div>
