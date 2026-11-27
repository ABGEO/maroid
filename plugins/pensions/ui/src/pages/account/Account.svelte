<script lang="ts">
	import { ApiError, failureMessage, type PluginHost } from '@maroid/plugin-sdk';
	import { ConfirmDialog } from '@maroid/ui';

	import '../../app.css';
	import { createPensionApi, type Connection } from '../../api';
	import { formatMoney } from '../../lib/format';

	const SERVICE_UNAVAILABLE = '/problems/dev.maroid.pensions/service-unavailable';

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createPensionApi(host));
	const canConnect = $derived(host.can('account.connect'));

	let connection = $state<Connection | null>(null);
	let status = $state<'loading' | 'ready' | 'unavailable' | 'error'>('loading');
	let busy = $state(false);
	let failure = $state<string | null>(null);
	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();

	const holder = $derived(connection?.state === 'connected' ? connection.holder : undefined);

	function show(read: Connection): void {
		connection = read;
		status = 'ready';
	}

	async function load(): Promise<void> {
		status = 'loading';
		failure = null;

		try {
			const read = await api.connection.read();
			if (read !== null) {
				show(read);
			}
		} catch (error) {
			status = error instanceof ApiError && error.is(SERVICE_UNAVAILABLE) ? 'unavailable' : 'error';
		}
	}

	async function run(action: () => Promise<void>, fallback: string): Promise<void> {
		if (busy) {
			return;
		}

		busy = true;
		failure = null;

		try {
			await action();
		} catch (error) {
			failure = failureMessage(error, fallback);
		} finally {
			busy = false;
		}
	}

	function connect(): Promise<void> {
		return run(async () => {
			const read = await api.connection.connect();
			if (read !== null) {
				show(read);
			}
		}, 'The pension account is not connected. Try again.');
	}

	async function disconnect(): Promise<void> {
		const proceed = await confirmDialog?.ask(
			'Maroid collects nothing until you connect again. The collected data and your username and password in the settings stay.',
			{ title: 'Disconnect the pension account?', confirmLabel: 'Disconnect' }
		);
		if (proceed) {
			await run(async () => {
				await api.connection.disconnect();
				await load();
			}, 'The pension account stays connected. Try again.');
		}
	}

	$effect(() => {
		void load();
	});
</script>

<div class="max-w-xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">Pension account</h1>
			<p class="page-lead">
				Connect your account at the pension service, so Maroid collects your contributions, your
				asset value, and your portfolios on the first day of each month.
			</p>
		</div>
	</header>

	{#if failure}
		<div role="alert" class="alert alert-error alert-soft mb-4">
			<span>{failure}</span>
		</div>
	{/if}

	{#if status === 'loading' && connection === null}
		<span class="skeleton h-32 w-full" aria-hidden="true"></span>
	{:else if status === 'unavailable'}
		<div role="alert" class="alert alert-warning alert-soft">
			<span>The pension service is unavailable. The connection stays.</span>
			<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
		</div>
	{:else if status === 'error'}
		<div role="alert" class="alert alert-error alert-soft">
			<span>Failed to load the pension account.</span>
			<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
		</div>
	{:else if holder}
		<section class="panel mb-6 p-4">
			<div class="mb-4 flex items-center justify-between gap-2">
				<h2 class="font-medium">Connected as {holder.full_name}</h2>
				<span class="badge badge-success badge-soft">Connected</span>
			</div>

			<dl class="mb-4">
				<dt class="meta">Asset value</dt>
				<dd class="text-2xl font-semibold tabular-nums">{formatMoney(holder.asset_value)}</dd>
			</dl>

			{#if canConnect}
				<div class="flex gap-2">
					<button type="button" class="btn btn-sm" disabled={busy} onclick={connect}>
						Connect again
					</button>
					<button
						type="button"
						class="btn btn-ghost btn-sm text-error"
						disabled={busy}
						onclick={disconnect}
					>
						Disconnect
					</button>
				</div>
			{/if}
		</section>
	{:else}
		<section class="panel mb-6 p-4">
			<div class="mb-4 flex items-center justify-between gap-2">
				<h2 class="font-medium">Not connected</h2>
				<span class="badge badge-ghost">Not connected</span>
			</div>
			{#if !canConnect}
				<p class="meta">Your role in this workspace does not allow a connection.</p>
			{:else if connection?.credentials_stored}
				<p class="meta mb-4">
					Maroid signs in to the pension service with the username and the password in your
					settings.
				</p>
				<button type="button" class="btn btn-primary btn-sm" disabled={busy} onclick={connect}>
					Connect
				</button>
			{:else}
				<p class="meta">
					Store the username and the password of your pension account in the
					<a class="link" href={host.href('/settings')}>settings</a> first.
				</p>
			{/if}
		</section>
	{/if}

	<ConfirmDialog bind:this={confirmDialog} />
</div>
