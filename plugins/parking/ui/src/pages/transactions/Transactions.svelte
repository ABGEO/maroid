<script lang="ts">
	import { failureMessage, formatMoment, type PluginHost } from '@maroid/plugin-sdk';
	import { createPager, Pagination } from '@maroid/ui';

	import '../../app.css';
	import {
		createParkingApi,
		type TransactionFilter,
		type TransactionKind,
		type TransactionTotal
	} from '../../api';
	import FilterBar from '../../components/FilterBar.svelte';
	import NoAccess from '../../components/NoAccess.svelte';
	import NoData from '../../components/NoData.svelte';
	import { filterOfAddress, keepInAddress } from '../../lib/address';
	import { formatMoney, KIND_LABELS, signOf } from '../../lib/format';

	const KEYS = ['from', 'to', 'kind'] as const;
	const KINDS = Object.keys(KIND_LABELS) as TransactionKind[];

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createParkingApi(host));
	const canRead = $derived(host.can('history.read'));

	let filter = $state<TransactionFilter>(
		filterOfAddress(window.location.search, KEYS) as TransactionFilter
	);
	let total = $state<TransactionTotal | null>(null);
	let totalFailure = $state<string | null>(null);
	let readsFirstPage = true;
	let lastReadWasFirst = true;

	const reversed = $derived(
		filter.from !== undefined && filter.to !== undefined && filter.from > filter.to
	);
	const filtered = $derived(KEYS.some((key) => filter[key] !== undefined));

	// A change of the filter retires the cursor of the earlier page, so the next read starts
	// from the first page of the new filter.
	const pager = createPager((link?: string) => {
		const follow = readsFirstPage ? undefined : link;
		readsFirstPage = false;
		lastReadWasFirst = follow === undefined;

		return api.transactions.page($state.snapshot(filter), follow);
	});

	async function loadTotal(): Promise<void> {
		totalFailure = null;

		try {
			total = await api.transactions.total($state.snapshot(filter));
		} catch (error) {
			total = null;
			totalFailure = failureMessage(error, 'Failed to load the total.');
		}
	}

	function retry(): void {
		readsFirstPage = lastReadWasFirst;
		void pager.reload();
	}

	function choose(key: keyof TransactionFilter, value: string): void {
		filter = { ...filter, [key]: value === '' ? undefined : value };
		keepInAddress(filter, KEYS);
	}

	function clear(): void {
		filter = {};
		keepInAddress(filter, KEYS);
	}

	$effect(() => {
		// Read every member, so that a change of any one of them reads again.
		void [filter.from, filter.to, filter.kind];

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
			<h1 class="page-title">Transactions</h1>
			<p class="page-lead">Each top up and each charge on the balance, the latest first.</p>
		</div>
	</header>

	{#if !canRead}
		<NoAccess />
	{:else}
		<FilterBar {filtered} onclear={clear}>
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
			<label class="flex flex-col gap-1">
				<span class="text-sm">Kind</span>
				<select
					class="select select-sm min-w-44"
					value={filter.kind ?? ''}
					onchange={(event) => choose('kind', event.currentTarget.value)}
				>
					<option value="">Every kind</option>
					{#each KINDS as kind (kind)}
						<option value={kind}>{KIND_LABELS[kind]}</option>
					{/each}
				</select>
			</label>
		</FilterBar>

		{#if reversed}
			<div role="alert" class="alert alert-warning alert-soft max-w-xl">
				<span>The period is not valid: the first date is after the last date.</span>
			</div>
		{:else}
			<p class="mb-3 text-sm" aria-live="polite">
				{#if totalFailure !== null}
					<span class="text-error">{totalFailure}</span>
				{:else if total !== null && total.count > 0}
					{total.count}
					{total.count === 1 ? 'transaction' : 'transactions'}: {formatMoney(total.top_ups)} topped up,
					{formatMoney(total.charges)} charged.
				{/if}
			</p>

			<section class="min-w-0" aria-label="The transactions">
				{#if pager.status === 'loading' && pager.page === null}
					<div class="flex flex-col gap-2" aria-hidden="true">
						{#each [0, 1, 2, 3] as i (i)}
							<span class="skeleton h-10 w-full"></span>
						{/each}
					</div>
				{:else if pager.status === 'error'}
					<div role="alert" class="alert alert-error alert-soft">
						<span>Failed to load the transactions.</span>
						<button type="button" class="btn btn-sm" onclick={retry}>Retry</button>
					</div>
				{:else if pager.page?.items.length === 0}
					{#if filtered}
						<div class="empty-state">No transaction matches the filter.</div>
					{:else}
						<NoData {host} />
					{/if}
				{:else}
					<div class="panel overflow-x-auto">
						<table class="table-zebra table">
							<thead>
								<tr>
									<th>Moment</th>
									<th>Kind</th>
									<th>Description</th>
									<th>Parking lot</th>
									<th class="text-right">Amount</th>
									<th class="text-right">Balance after</th>
								</tr>
							</thead>
							<tbody>
								{#each pager.page?.items ?? [] as transaction (transaction.id)}
									{@const sign = signOf(transaction.kind)}
									<tr>
										<td class="whitespace-nowrap">
											{formatMoment(transaction.occurred_at, host.timeZone, {
												dateStyle: 'medium',
												timeStyle: 'short'
											})}
										</td>
										<td class="whitespace-nowrap">{KIND_LABELS[transaction.kind]}</td>
										<td>{transaction.description}</td>
										<td>
											{#if transaction.session}
												<span class="font-medium">{transaction.session.lot_number}</span>
												<span class="text-sm opacity-70">{transaction.session.address}</span>
											{/if}
										</td>
										<td
											class={[
												'text-right whitespace-nowrap tabular-nums',
												sign === '+' && 'text-success'
											]}
										>
											{sign}{formatMoney(transaction.amount)}
										</td>
										<td class="text-right whitespace-nowrap tabular-nums">
											{formatMoney(transaction.balance_after)}
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
