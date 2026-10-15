<script lang="ts">
	import { createPager, Pagination } from '@maroid/ui';

	import type { JasmineApi, Plant, PlantFilter } from '../../api';

	interface Props {
		api: JasmineApi;
		filter: PlantFilter;
		/** The plants that the bulk log takes, by identifier. */
		selected: Map<string, Plant>;
		selectable: boolean;
		/** The deck address of the page of a plant. */
		hrefOf: (plant: Plant) => string;
		onopen: (plant: Plant) => void;
		ontoggle: (plant: Plant) => void;
	}

	let { api, filter, selected, selectable, hrefOf, onopen, ontoggle }: Props = $props();

	const pager = createPager((link?: string) => api.plants.list(filter, link));

	$effect(() => {
		void pager.reload();
	});
</script>

{#if pager.status === 'loading' && pager.page === null}
	<div class="flex flex-col gap-2" aria-hidden="true">
		{#each [0, 1, 2] as i (i)}
			<span class="skeleton h-12 w-full"></span>
		{/each}
	</div>
{:else if pager.status === 'error'}
	<div role="alert" class="alert alert-error alert-soft">
		<span>Failed to load the plants.</span>
		<button type="button" class="btn btn-sm" onclick={() => pager.reload()}>Retry</button>
	</div>
{:else if pager.page?.items.length === 0}
	<div class="empty-state">No plant matches.</div>
{:else}
	<div class="panel overflow-x-auto">
		<table class="table">
			<thead>
				<tr>
					{#if selectable}<th class="w-0"><span class="sr-only">Select</span></th>{/if}
					<th>ID</th>
					<th>Name</th>
					<th>Environment</th>
					<th>Status</th>
				</tr>
			</thead>
			<tbody>
				{#each pager.page?.items ?? [] as plant (plant.id)}
					<tr class="hover:bg-base-200 cursor-pointer" onclick={() => onopen(plant)}>
						{#if selectable}
							<td>
								<input
									type="checkbox"
									class="checkbox checkbox-sm"
									aria-label="Select {plant.display_id}"
									checked={selected.has(plant.id)}
									onclick={(event) => event.stopPropagation()}
									onchange={() => ontoggle(plant)}
								/>
							</td>
						{/if}
						<td class="meta">{plant.display_id}</td>
						<td>
							<a
								class="link link-hover font-medium"
								href={hrefOf(plant)}
								onclick={(event) => event.stopPropagation()}>{plant.common_name}</a
							>
							{#if plant.scientific_name}<span class="meta ml-1 italic"
									>{plant.scientific_name}</span
								>{/if}
						</td>
						<td>{plant.environment.display_id} {plant.environment.name}</td>
						<td>
							{#if plant.status === 'active'}
								<span class="badge badge-sm badge-soft badge-success">Active</span>
							{:else}
								<span class="badge badge-sm badge-ghost">Inactive</span>
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
	<Pagination {pager} class="mt-4 ml-auto w-fit" />
{/if}
