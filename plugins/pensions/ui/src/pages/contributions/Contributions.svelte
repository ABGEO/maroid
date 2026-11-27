<script lang="ts">
	import { failureMessage, type PluginHost } from '@maroid/plugin-sdk';
	import { createPager, Pagination } from '@maroid/ui';

	import '../../app.css';
	import {
		createPensionApi,
		type ContributionFilter,
		type ContributionTotal,
		type Employer
	} from '../../api';
	import NoAccess from '../../components/NoAccess.svelte';
	import { filterOfAddress, keepInAddress } from '../../lib/address';
	import { formatDate, formatMoney, formatMonth } from '../../lib/format';

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createPensionApi(host));
	const canRead = $derived(host.can('account.read'));

	let filter = $state<ContributionFilter>(filterOfAddress(window.location.search));
	let employers = $state<Employer[]>([]);
	let total = $state<ContributionTotal | null>(null);
	let totalFailure = $state<string | null>(null);
	let readsFirstPage = true;
	let lastReadWasFirst = true;

	const reversed = $derived(
		filter.from !== undefined && filter.to !== undefined && filter.from > filter.to
	);
	const filtered = $derived(
		filter.employer !== undefined || filter.from !== undefined || filter.to !== undefined
	);
	const unknownEmployer = $derived(
		filter.employer !== undefined && !employers.some((one) => one.code === filter.employer)
	);

	// A change of the filter retires the cursor of the earlier page, so the next read starts
	// from the first page of the new filter.
	const pager = createPager((link?: string) => {
		const follow = readsFirstPage ? undefined : link;
		readsFirstPage = false;
		lastReadWasFirst = follow === undefined;

		return api.contributions.page($state.snapshot(filter), follow);
	});

	async function loadEmployers(): Promise<void> {
		try {
			employers = (await api.contributions.employers()) ?? [];
		} catch (error) {
			console.error('Failed to load the employers', error);
		}
	}

	async function loadTotal(): Promise<void> {
		totalFailure = null;

		try {
			total = await api.contributions.total($state.snapshot(filter));
		} catch (error) {
			total = null;
			totalFailure = failureMessage(error, 'Failed to load the total.');
		}
	}

	function retry(): void {
		readsFirstPage = lastReadWasFirst;
		void pager.reload();
	}

	function choose(key: keyof ContributionFilter, value: string): void {
		filter = { ...filter, [key]: value === '' ? undefined : value };
		keepInAddress(filter);
	}

	function clear(): void {
		filter = {};
		keepInAddress(filter);
	}

	$effect(() => {
		if (canRead) {
			void loadEmployers();
		}
	});

	$effect(() => {
		// Read every member, so that a change of any one of them reads again.
		void [filter.employer, filter.from, filter.to];

		if (reversed || !canRead) {
			return;
		}

		readsFirstPage = true;
		void pager.reload();
		void loadTotal();
	});
</script>

<div>
	<header class="page-header">
		<div>
			<h1 class="page-title">Contributions</h1>
			<p class="page-lead">Each contribution to the pension account, the latest first.</p>
		</div>
	</header>

	{#if !canRead}
		<NoAccess />
	{:else}
		<form
			class="panel mb-4 flex flex-wrap items-end gap-3 p-4"
			onsubmit={(event) => event.preventDefault()}
		>
			<label class="flex flex-col gap-1">
				<span class="text-sm">Employer</span>
				<select
					class="select select-sm min-w-56"
					value={filter.employer ?? ''}
					onchange={(event) => choose('employer', event.currentTarget.value)}
				>
					<option value="">Every employer</option>
					{#each employers as employer (employer.code)}
						<option value={employer.code}>{employer.name}</option>
					{/each}
					{#if unknownEmployer && filter.employer !== undefined}
						<option value={filter.employer}>{filter.employer} (no contribution)</option>
					{/if}
				</select>
			</label>
			<label class="flex flex-col gap-1">
				<span class="text-sm">From</span>
				<input
					type="date"
					class="input input-sm"
					value={filter.from ?? ''}
					onchange={(event) => choose('from', event.currentTarget.value)}
				/>
			</label>
			<label class="flex flex-col gap-1">
				<span class="text-sm">To</span>
				<input
					type="date"
					class="input input-sm"
					value={filter.to ?? ''}
					onchange={(event) => choose('to', event.currentTarget.value)}
				/>
			</label>
			{#if filtered}
				<button type="button" class="btn btn-sm btn-ghost" onclick={clear}>Clear</button>
			{/if}
		</form>

		{#if reversed}
			<div role="alert" class="alert alert-warning alert-soft max-w-xl">
				<span>The period is not valid: the first date is after the last date.</span>
			</div>
		{:else}
			<p class="mb-3 text-sm" aria-live="polite">
				{#if totalFailure !== null}
					<span class="text-error">{totalFailure}</span>
				{:else if total !== null}
					{total.count}
					{total.count === 1 ? 'contribution' : 'contributions'}, {formatMoney(total.amount)} in all.
				{/if}
			</p>

			<section class="min-w-0" aria-label="The contributions">
				{#if pager.status === 'loading' && pager.page === null}
					<div class="flex flex-col gap-2" aria-hidden="true">
						{#each [0, 1, 2, 3] as i (i)}
							<span class="skeleton h-10 w-full"></span>
						{/each}
					</div>
				{:else if pager.status === 'error'}
					<div role="alert" class="alert alert-error alert-soft">
						<span>Failed to load the contributions.</span>
						<button type="button" class="btn btn-sm" onclick={retry}>Retry</button>
					</div>
				{:else if pager.page?.items.length === 0}
					<div class="empty-state">
						{#if filtered}
							No contribution matches the filter.
						{:else}
							No data yet. The first collection runs on the first day of the next month.
						{/if}
					</div>
				{:else}
					<div class="panel overflow-x-auto">
						<table class="table-zebra table">
							<thead>
								<tr>
									<th>Date</th>
									<th>Employer</th>
									<th>Source</th>
									<th>Type</th>
									<th>Salary month</th>
									<th class="text-right">Gross salary</th>
									<th class="text-right">Amount</th>
								</tr>
							</thead>
							<tbody>
								{#each pager.page?.items ?? [] as contribution (contribution.id)}
									<tr>
										<td class="whitespace-nowrap">{formatDate(contribution.paid_on)}</td>
										<td>{contribution.employer?.name ?? ''}</td>
										<td>{contribution.source}</td>
										<td>{contribution.type}</td>
										<td class="tabular-nums">
											{contribution.salary_month ? formatMonth(contribution.salary_month) : ''}
										</td>
										<td class="text-right whitespace-nowrap tabular-nums">
											{formatMoney(contribution.gross_salary)}
										</td>
										<td class="text-right whitespace-nowrap tabular-nums">
											{contribution.amount ? formatMoney(contribution.amount) : ''}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
					<Pagination {pager} class="mt-4 ml-auto w-fit" />
				{/if}
			</section>
		{/if}
	{/if}
</div>
