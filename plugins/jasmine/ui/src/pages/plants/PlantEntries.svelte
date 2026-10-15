<script lang="ts">
	import { createPager, Pagination } from '@maroid/ui';

	import type { JasmineApi } from '../../api';
	import { formatMoment } from '../../lib/format';
	import { typeLabel } from '../../lib/maintenance';

	interface Props {
		api: JasmineApi;
		plantId: string;
		/** Changes when an entry of the plant lands, so the first page reads again. */
		revision: number;
	}

	let { api, plantId, revision }: Props = $props();

	const pager = createPager((link?: string) =>
		api.maintenanceEntries.list({ plant_id: plantId }, link)
	);

	$effect(() => {
		void revision;
		void pager.reload();
	});
</script>

<section class="section">
	<h2 class="section-title">Maintenance entries</h2>

	{#if pager.status === 'loading' && pager.page === null}
		<span class="skeleton h-24 w-full" aria-hidden="true"></span>
	{:else if pager.status === 'error'}
		<div role="alert" class="alert alert-error alert-soft">
			<span>Failed to load the entries.</span>
			<button type="button" class="btn btn-sm" onclick={() => pager.reload()}>Retry</button>
		</div>
	{:else if pager.page?.items.length === 0}
		<div class="empty-state">This plant has no entry yet.</div>
	{:else}
		<ul class="list panel">
			{#each pager.page?.items ?? [] as entry (entry.id)}
				<li class="list-row items-center">
					<span class="meta whitespace-nowrap">{formatMoment(entry.performed_at)}</span>
					<span class="font-medium">{typeLabel(entry.type)}</span>
					<span class="list-col-grow">{entry.note ?? ''}</span>
				</li>
			{/each}
		</ul>
		<Pagination {pager} class="mt-4 ml-auto w-fit" />
	{/if}
</section>
