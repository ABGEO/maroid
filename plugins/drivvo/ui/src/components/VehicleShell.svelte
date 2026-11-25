<script lang="ts">
	import type { PluginHost } from '@maroid/plugin-sdk';
	import type { Snippet } from 'svelte';

	import '../app.css';
	import type { Vehicle } from '../api';
	import type { Fleet } from '../lib/vehicles.svelte';

	let {
		host,
		fleet,
		page,
		title,
		crumb,
		lead,
		actions,
		children
	}: {
		host: PluginHost;
		fleet: Fleet;
		/** The first segment of the address, kept when the member picks another vehicle. */
		page: string;
		title: string;
		/** The last link of the trail. The name of the vehicle when absent. */
		crumb?: string;
		lead?: string;
		actions?: Snippet;
		children: Snippet<[Vehicle]>;
	} = $props();

	function choose(event: Event): void {
		const chosen = (event.currentTarget as HTMLSelectElement).value;
		if (chosen !== fleet.vehicle?.id) {
			host.navigate(`/${page}/${encodeURIComponent(chosen)}`);
		}
	}

	$effect(() => {
		const last = crumb ?? fleet.vehicle?.name;
		if (last !== undefined) {
			host.setTitle(last);
		}
	});

	$effect(() => {
		void fleet.load();
	});
</script>

<div>
	<header class="page-header">
		<div>
			<h1 class="page-title">{title}</h1>
			{#if lead}
				<p class="page-lead">{lead}</p>
			{/if}
		</div>
		{#if fleet.vehicle !== undefined}
			<div class="flex flex-wrap items-center gap-2">
				<select
					class="select select-sm w-auto"
					aria-label="Vehicle"
					value={fleet.vehicle.id}
					onchange={choose}
				>
					{#each fleet.vehicles ?? [] as option (option.id)}
						<option value={option.id}>
							{option.name}{option.active ? '' : ' (inactive)'}
						</option>
					{/each}
				</select>
				{@render actions?.()}
			</div>
		{/if}
	</header>

	{#if fleet.status === 'loading' && fleet.vehicles === null}
		<div class="flex flex-col gap-6" aria-hidden="true">
			<span class="skeleton h-20 w-full"></span>
			<span class="skeleton h-64 w-full"></span>
		</div>
	{:else if fleet.status === 'error'}
		<div role="alert" class="alert alert-error alert-soft max-w-xl">
			<span>Failed to load the vehicles.</span>
			<button type="button" class="btn btn-sm" onclick={fleet.load}>Retry</button>
		</div>
	{:else if fleet.vehicle === undefined}
		<div role="alert" class="alert alert-info alert-soft max-w-xl">
			<span>This vehicle is not in Maroid.</span>
			<a class="btn btn-sm" href={host.href(`/${page}`)}>Open the principal vehicle</a>
		</div>
	{:else}
		{@render children(fleet.vehicle)}
	{/if}
</div>
