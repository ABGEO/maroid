<script lang="ts">
	import { ApiError, type PluginHost } from '@maroid/plugin-sdk';

	import { createParkingApi, type Account } from '../api';
	import { formatMoney } from '../lib/format';

	const SERVICE_UNAVAILABLE = '/problems/dev.maroid.parking/service-unavailable';

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createParkingApi(host));

	let account = $state<Account | null>(null);
	let status = $state<'loading' | 'ready' | 'not_connected' | 'unavailable' | 'error'>('loading');

	async function load(): Promise<void> {
		status = 'loading';

		try {
			const read = await api.connection.read();
			if (read === null) {
				return;
			}

			if (read.state !== 'connected' || read.account === undefined) {
				status = 'not_connected';

				return;
			}

			account = read.account;
			status = 'ready';
		} catch (error) {
			status = error instanceof ApiError && error.is(SERVICE_UNAVAILABLE) ? 'unavailable' : 'error';
		}
	}

	$effect(() => {
		void load();
	});
</script>

<section class="panel p-4" aria-labelledby="balance-title">
	<h2 id="balance-title" class="mb-2 font-medium">Balance</h2>
	{#if status === 'loading'}
		<span class="skeleton h-12 w-48" aria-hidden="true"></span>
	{:else if status === 'ready' && account !== null}
		<div class="flex flex-wrap items-baseline gap-x-6 gap-y-1">
			<span class="text-2xl font-semibold tabular-nums">{formatMoney(account.balance)}</span>
			<span class="text-sm">
				{account.free_parkings_left === 1
					? '1 free parking left today'
					: `${account.free_parkings_left} free parkings left today`}
			</span>
		</div>
	{:else if status === 'not_connected'}
		<div class="flex flex-wrap items-center gap-3">
			<span>Not connected</span>
			<a class="btn btn-sm" href={host.href('/account')}>Open the account</a>
		</div>
	{:else if status === 'unavailable'}
		<div role="alert" class="alert alert-warning alert-soft">
			<span>The parking service is unavailable.</span>
			<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
		</div>
	{:else}
		<div role="alert" class="alert alert-error alert-soft">
			<span>Failed to load the balance.</span>
			<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
		</div>
	{/if}
</section>
