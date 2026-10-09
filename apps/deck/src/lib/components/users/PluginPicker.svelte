<script lang="ts">
	import { displayNameOf } from '$lib/plugins/capabilities';
	import { pluginState } from '$lib/state/plugins.svelte';

	interface Props {
		selected: string[];
		administrator: boolean;
	}

	let { selected = $bindable(), administrator }: Props = $props();

	const summary = $derived(
		selected.length === 0
			? 'No plugin'
			: pluginState.plugins
					.filter((plugin) => selected.includes(plugin.id))
					.map((plugin) => displayNameOf(plugin, plugin.id))
					.join(', ')
	);
</script>

<div class="flex flex-col gap-1">
	<span class="label">Plugins</span>
	{#if administrator}
		<input
			class="input input-sm w-full"
			value="Every plugin"
			aria-label="The plugins that this person can turn on"
			disabled
		/>
	{:else}
		<div class="dropdown w-full">
			<div
				tabindex="0"
				role="button"
				class="input input-sm w-full cursor-pointer justify-between"
				aria-label="The plugins that this person can turn on"
			>
				<span class="truncate">{summary}</span>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					width="12"
					height="12"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					opacity="0.5"
				>
					<path d="m6 9 6 6 6-6" />
				</svg>
			</div>
			<ul
				tabindex="-1"
				class="dropdown-content menu menu-sm bg-base-100 border-base-300 rounded-box z-50 mt-2 w-full border p-2 shadow-lg"
			>
				{#each pluginState.plugins as plugin (plugin.id)}
					<li>
						<label class="flex items-center gap-2">
							<input
								type="checkbox"
								class="checkbox checkbox-sm"
								value={plugin.id}
								bind:group={selected}
							/>
							<span class="truncate">{displayNameOf(plugin, plugin.id)}</span>
						</label>
					</li>
				{/each}
			</ul>
		</div>
	{/if}
	<span class="label whitespace-normal">
		{administrator
			? 'An administrator already reaches every plugin.'
			: 'The plugins that this person can turn on in a workspace.'}
	</span>
</div>
