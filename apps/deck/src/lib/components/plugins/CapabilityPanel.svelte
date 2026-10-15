<script lang="ts">
	import type { ResolvedPathname } from '$app/types';

	import type { Plugin } from '$lib/api';
	import {
		cellOf,
		columnsOf,
		itemsOf,
		labelOf,
		partsOf,
		uiOf,
		type CapabilityName
	} from '$lib/plugins/capabilities';

	interface Props {
		plugin: Plugin;
		name: CapabilityName;
		/** The page of a navigation entry. Absent while the workspace does not use the plugin. */
		entryHref?: (path: string) => ResolvedPathname;
	}

	let { plugin, name, entryHref }: Props = $props();

	const columns = $derived(columnsOf(name));
	const items = $derived(itemsOf(plugin, name));
	const entries = $derived(name === 'ui' ? (uiOf(plugin)?.routes ?? []) : []);
</script>

{#snippet breakable(value: string)}
	{#each partsOf(value) as part, index (index)}{part}<wbr />{/each}
{/snippet}

<div class="panel min-w-0 p-4">
	<h3 class="mb-2 text-sm font-medium">{labelOf(name)}</h3>

	{#if name === 'ui'}
		<div class="overflow-x-auto">
			<table class="table-sm table">
				<thead>
					<tr>
						<th>Page</th>
						<th>Path</th>
					</tr>
				</thead>
				<tbody>
					{#each entries as entry (entry.path)}
						<tr>
							<td>
								{#if entryHref}
									<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- entryHref resolves every target -->
									<a class="link link-hover" href={entryHref(entry.path)}>{entry.label}</a>
								{:else}
									{entry.label}
								{/if}
							</td>
							<td class="font-mono wrap-break-word">{@render breakable(entry.path)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else if columns.length === 0}
		<p class="meta">Declared. It holds no item.</p>
	{:else}
		<div class="overflow-x-auto">
			<table class="table-sm table">
				<thead>
					<tr>
						{#each columns as column (column.field)}
							<th>{column.label}</th>
						{/each}
					</tr>
				</thead>
				<tbody>
					{#each items as item, index (index)}
						<tr>
							{#each columns as column (column.field)}
								<td class={['wrap-break-word', column.mono && 'font-mono']}>
									{@render breakable(cellOf(item, column.field))}
								</td>
							{/each}
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>
