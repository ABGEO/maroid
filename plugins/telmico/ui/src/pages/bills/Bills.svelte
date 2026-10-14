<script lang="ts">
	import { ApiError, type Page, type PluginHost } from '@maroid/plugin-sdk';
	import { createPager, Pagination } from '@maroid/ui';

	import '../../app.css';
	import { createElectricityApi, type Bill } from '../../api';

	const NOT_ASSIGNED = '/problems/dev.maroid.telmico/not-assigned';

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createElectricityApi(host));

	let notAssigned = $state(false);

	// The pager reports any failure as one error, so the read answers the missing
	// assignment itself: an empty page, with the flag that the page shows first.
	const pager = createPager(async (link?: string): Promise<Page<Bill> | null> => {
		try {
			const answered = await api.bills.page(link);
			notAssigned = false;

			return answered;
		} catch (error) {
			if (error instanceof ApiError && error.is(NOT_ASSIGNED)) {
				notAssigned = true;

				return { items: [], self: '', first: '' };
			}

			throw error;
		}
	});

	const monthLabel = new Intl.DateTimeFormat(undefined, {
		month: 'long',
		year: 'numeric',
		timeZone: 'UTC'
	});

	/** A month YYYY-MM by its name, with no conversion of time zone. */
	function shownMonth(month: string): string {
		return monthLabel.format(new Date(`${month}-01T00:00:00Z`));
	}

	/** A date YYYY-MM-DD as dd/mm/yyyy, with no conversion of time zone. */
	function shownDate(date: string): string {
		const [year, month, day] = date.split('-');

		return `${day}/${month}/${year}`;
	}

	$effect(() => {
		void pager.reload();
	});
</script>

<div>
	<header class="page-header">
		<div>
			<h1 class="page-title">Bills</h1>
			<p class="page-lead">
				The bill of each month of the assigned customer. A download fetches the document from
				Telmico, and Maroid keeps no copy.
			</p>
		</div>
	</header>

	{#if notAssigned}
		<div role="alert" class="alert alert-info alert-soft max-w-xl">
			<span>Connect the electricity account and assign the customer of this workspace first.</span>
			<a class="btn btn-sm" href={host.href('/account')}>Open the account</a>
		</div>
	{:else if pager.status === 'loading' && pager.page === null}
		<div class="flex flex-col gap-2" aria-hidden="true">
			{#each [0, 1, 2, 3] as i (i)}
				<span class="skeleton h-10 w-full"></span>
			{/each}
		</div>
	{:else if pager.status === 'error'}
		<div role="alert" class="alert alert-error alert-soft">
			<span>Failed to load the bills.</span>
			<button type="button" class="btn btn-sm" onclick={() => pager.reload()}>Retry</button>
		</div>
	{:else if pager.page?.items.length === 0}
		<div class="empty-state">No bill yet.</div>
	{:else}
		<div class="panel overflow-x-auto">
			<table class="table-zebra table">
				<thead>
					<tr>
						<th>Month</th>
						<th>Bill number</th>
						<th>Issued</th>
						<th><span class="sr-only">Document</span></th>
					</tr>
				</thead>
				<tbody>
					{#each pager.page?.items ?? [] as bill (bill.id)}
						<tr>
							<td class="whitespace-nowrap">{shownMonth(bill.month)}</td>
							<td class="font-mono">{bill.bill_number}</td>
							<td class="whitespace-nowrap">{shownDate(bill.issued_on)}</td>
							<td class="text-right">
								<a
									class="btn btn-ghost btn-sm"
									href={bill.document}
									download
									aria-label={`Download the bill of ${shownMonth(bill.month)}`}>Download</a
								>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		<Pagination {pager} class="mt-4 ml-auto w-fit" />
	{/if}
</div>
