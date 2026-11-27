<script lang="ts">
	import { failureMessage, type PluginHost } from '@maroid/plugin-sdk';
	import { createPager, Pagination } from '@maroid/ui';

	import '../../app.css';
	import { createParkingApi, type SessionFilter, type SessionTotal } from '../../api';
	import FilterBar from '../../components/FilterBar.svelte';
	import NoAccess from '../../components/NoAccess.svelte';
	import NoData from '../../components/NoData.svelte';
	import { filterOfAddress, keepInAddress } from '../../lib/address';
	import { formatDay, formatDuration, formatMoney, formatTime } from '../../lib/format';

	const KEYS = ['from', 'to', 'plate', 'free'] as const;

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createParkingApi(host));
	const canRead = $derived(host.can('history.read'));

	let filter = $state<SessionFilter>(
		filterOfAddress(window.location.search, KEYS) as SessionFilter
	);
	let plates = $state<string[]>([]);
	let held = $state<boolean | null>(null);
	let total = $state<SessionTotal | null>(null);
	let totalFailure = $state<string | null>(null);
	let readsFirstPage = true;
	let lastReadWasFirst = true;

	const reversed = $derived(
		filter.from !== undefined && filter.to !== undefined && filter.from > filter.to
	);
	const filtered = $derived(KEYS.some((key) => filter[key] !== undefined));
	const unknownPlate = $derived(filter.plate !== undefined && !plates.includes(filter.plate));

	// A change of the filter retires the cursor of the earlier page, so the next read starts
	// from the first page of the new filter.
	const pager = createPager((link?: string) => {
		const follow = readsFirstPage ? undefined : link;
		readsFirstPage = false;
		lastReadWasFirst = follow === undefined;

		return api.sessions.page($state.snapshot(filter), follow);
	});

	async function loadPlates(): Promise<void> {
		try {
			plates = (await api.sessions.plates()) ?? [];
		} catch (error) {
			console.error('Failed to load the plates', error);
		}
	}

	// The first read carries no filter, so a filter in the address cannot hide that Maroid
	// holds no row yet.
	async function loadHeld(): Promise<void> {
		try {
			held = await api.sessions.any();
		} catch (error) {
			console.error('Failed to read whether Maroid holds a session', error);
		}
	}

	async function loadTotal(): Promise<void> {
		totalFailure = null;

		try {
			total = await api.sessions.total($state.snapshot(filter));
		} catch (error) {
			total = null;
			totalFailure = failureMessage(error, 'Failed to load the total.');
		}
	}

	function retry(): void {
		readsFirstPage = lastReadWasFirst;
		void pager.reload();
	}

	function choose(key: keyof SessionFilter, value: string): void {
		filter = { ...filter, [key]: value === '' ? undefined : value };
		keepInAddress(filter, KEYS);
	}

	function clear(): void {
		filter = {};
		keepInAddress(filter, KEYS);
	}

	$effect(() => {
		if (canRead) {
			void loadHeld();
			void loadPlates();
		}
	});

	$effect(() => {
		// Read every member, so that a change of any one of them reads again.
		void [filter.from, filter.to, filter.plate, filter.free];

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
			<h1 class="page-title">Sessions</h1>
			<p class="page-lead">Each finished parking session, the latest start first.</p>
		</div>
	</header>

	{#if !canRead}
		<NoAccess />
	{:else if held === false}
		<NoData {host} />
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
				<span class="text-sm">Vehicle</span>
				<select
					class="select select-sm min-w-40"
					value={filter.plate ?? ''}
					onchange={(event) => choose('plate', event.currentTarget.value)}
				>
					<option value="">Every vehicle</option>
					{#each plates as plate (plate)}
						<option value={plate}>{plate}</option>
					{/each}
					{#if unknownPlate && filter.plate !== undefined}
						<option value={filter.plate}>{filter.plate} (no session)</option>
					{/if}
				</select>
			</label>
			<label class="flex flex-col gap-1">
				<span class="text-sm">Charge</span>
				<select
					class="select select-sm min-w-40"
					value={filter.free ?? ''}
					onchange={(event) => choose('free', event.currentTarget.value)}
				>
					<option value="">Free and paid</option>
					<option value="true">Free parkings</option>
					<option value="false">Paid sessions</option>
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
					{total.count === 1 ? 'session' : 'sessions'}, {formatDuration(total.duration_seconds)}
					parked, {formatMoney(total.amount)} in all.
				{/if}
			</p>

			<section class="min-w-0" aria-label="The sessions">
				{#if pager.status === 'loading' && pager.page === null}
					<div class="flex flex-col gap-2" aria-hidden="true">
						{#each [0, 1, 2, 3] as i (i)}
							<span class="skeleton h-10 w-full"></span>
						{/each}
					</div>
				{:else if pager.status === 'error'}
					<div role="alert" class="alert alert-error alert-soft">
						<span>Failed to load the sessions.</span>
						<button type="button" class="btn btn-sm" onclick={retry}>Retry</button>
					</div>
				{:else if pager.page?.items.length === 0}
					<div class="empty-state">No session matches the filter.</div>
				{:else}
					<div class="panel overflow-x-auto">
						<table class="table-zebra table">
							<thead>
								<tr>
									<th>Date</th>
									<th>Start</th>
									<th>End</th>
									<th>Duration</th>
									<th>Parking lot</th>
									<th>Vehicle</th>
									<th class="text-right">Amount</th>
								</tr>
							</thead>
							<tbody>
								{#each pager.page?.items ?? [] as session (session.id)}
									<tr>
										<td class="whitespace-nowrap">
											{formatDay(session.started_at, host.timeZone)}
										</td>
										<td class="tabular-nums">{formatTime(session.started_at, host.timeZone)}</td>
										<td class="tabular-nums">{formatTime(session.ended_at, host.timeZone)}</td>
										<td class="whitespace-nowrap tabular-nums">
											{formatDuration(session.duration_seconds)}
										</td>
										<td>
											<span class="font-medium">{session.lot_number}</span>
											<span class="text-sm opacity-70">{session.address}</span>
										</td>
										<td class="whitespace-nowrap">{session.plate}</td>
										<td class="text-right whitespace-nowrap tabular-nums">
											{#if session.free}
												<span class="badge badge-sm badge-success badge-soft">Free</span>
											{:else}
												{formatMoney(session.amount)}
											{/if}
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
