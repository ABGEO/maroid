<script lang="ts">
	import { ApiError, failureMessage, type PluginHost } from '@maroid/plugin-sdk';
	import { ConfirmDialog } from '@maroid/ui';

	import '../../app.css';
	import { createParkingApi, type Connection, type PendingConnection } from '../../api';

	const SERVICE_UNAVAILABLE = '/problems/dev.maroid.parking/service-unavailable';
	const CODE_PATTERN = '[0-9]{4,8}';
	const TICK_MS = 1000;

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createParkingApi(host));

	let connection = $state<Connection | null>(null);
	let pending = $state<PendingConnection | null>(null);
	let status = $state<'loading' | 'ready' | 'unavailable' | 'error'>('loading');
	let code = $state('');
	let busy = $state(false);
	let failure = $state<string | null>(null);
	let now = $state(Date.now());
	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();

	const connected = $derived(connection?.state === 'connected');
	const credentialsStored = $derived(connection?.credentials_stored ?? false);
	const resendIn = $derived(pending ? secondsUntil(pending.resend_at, now) : 0);
	const expired = $derived(pending !== null && secondsUntil(pending.expires_at, now) === 0);

	function secondsUntil(moment: string, from: number): number {
		return Math.max(0, Math.ceil((Date.parse(moment) - from) / TICK_MS));
	}

	async function load(): Promise<void> {
		status = 'loading';
		failure = null;

		try {
			const read = await api.connection.read();
			if (read === null) {
				return;
			}

			connection = read;
			pending = read.pending ?? null;
			status = 'ready';
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
			pending = await api.connection.start();
			code = '';
		}, 'The parking service sent no code. Try again.');
	}

	function requestCode(): Promise<void> {
		return run(async () => {
			pending = await api.connection.requestCode();
		}, 'The parking service sent no new code. Try again.');
	}

	function verify(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		return run(async () => {
			const verified = await api.connection.verify(code.trim());
			if (verified !== null) {
				connection = verified;
				pending = null;
				code = '';
			}
		}, 'The code was not checked. Try again.');
	}

	async function disconnect(): Promise<void> {
		const proceed = await confirmDialog?.ask(
			'The bot stops parking for you until you connect again. Your username and password stay in the settings.',
			{ title: 'Disconnect the parking account?', confirmLabel: 'Disconnect' }
		);
		if (proceed) {
			await run(async () => {
				await api.connection.disconnect();
				await load();
			}, 'The parking account stays connected. Try again.');
		}
	}

	$effect(() => {
		void load();
	});

	$effect(() => {
		if (pending === null) {
			return;
		}

		const timer = setInterval(() => (now = Date.now()), TICK_MS);

		return () => clearInterval(timer);
	});
</script>

<div class="max-w-xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">Parking account</h1>
			<p class="page-lead">
				Connect your account at the parking service, so the bot starts and stops a session for you.
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
			<span>The parking service is unavailable. Your connection stays.</span>
			<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
			<button type="button" class="btn btn-ghost btn-sm" disabled={busy} onclick={disconnect}
				>Disconnect</button
			>
		</div>
	{:else if status === 'error'}
		<div role="alert" class="alert alert-error alert-soft">
			<span>Failed to load the parking account.</span>
			<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
		</div>
	{:else}
		<section class="panel mb-6 p-4">
			{#if connected}
				<div class="mb-4 flex items-center justify-between gap-2">
					<h2 class="font-medium">
						{#if connection?.account}
							Connected as {connection.account.first_name}
							{connection.account.last_name}
						{:else}
							Connected
						{/if}
					</h2>
					<span class="badge badge-success badge-soft">Connected</span>
				</div>
				{#if connection?.account}
					<div class="stats stats-vertical sm:stats-horizontal mb-4 w-full">
						<div class="stat">
							<div class="stat-title">Balance</div>
							<div class="stat-value text-2xl">
								{connection.account.balance.amount.toFixed(2)}
								{connection.account.balance.currency}
							</div>
						</div>
						<div class="stat">
							<div class="stat-title">Free parkings left today</div>
							<div class="stat-value text-2xl">{connection.account.free_parkings_left}</div>
						</div>
					</div>
				{:else}
					<p class="meta mb-4">
						The parking service did not report the account. Reload the page to read it.
					</p>
				{/if}
				<div class="flex gap-2">
					<button
						type="button"
						class="btn btn-sm"
						disabled={busy || pending !== null}
						onclick={connect}
					>
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
			{:else}
				<div class="mb-4 flex items-center justify-between gap-2">
					<h2 class="font-medium">Not connected</h2>
					<span class="badge badge-ghost">Not connected</span>
				</div>
				{#if credentialsStored}
					<p class="meta mb-4">
						Maroid signs in with the username and the password in the settings. The parking service
						then sends a code to your phone.
					</p>
					<button
						type="button"
						class="btn btn-primary btn-sm"
						disabled={busy || pending !== null}
						onclick={connect}
					>
						Connect
					</button>
				{:else}
					<p class="meta">
						Store the username and the password of your parking account in the
						<a class="link" href={host.href('/settings')}>settings</a> first.
					</p>
				{/if}
			{/if}
		</section>

		{#if pending}
			<section class="panel p-4">
				{#if expired}
					<p class="mb-4">The code expired.</p>
					<button type="button" class="btn btn-primary btn-sm" disabled={busy} onclick={connect}>
						Connect again
					</button>
				{:else}
					<form class="flex flex-col gap-3" onsubmit={verify}>
						<fieldset class="fieldset">
							<legend class="fieldset-legend">Code</legend>
							<input
								class="input w-40 tracking-widest"
								inputmode="numeric"
								autocomplete="one-time-code"
								pattern={CODE_PATTERN}
								maxlength={8}
								placeholder="0000"
								bind:value={code}
								disabled={busy}
								required
							/>
							<p class="label">The parking service sent a code to {pending.phone}.</p>
						</fieldset>
						<div class="flex gap-2">
							<button
								type="submit"
								class="btn btn-primary btn-sm"
								disabled={busy || code.trim() === ''}
							>
								Verify
							</button>
							<button
								type="button"
								class="btn btn-ghost btn-sm"
								disabled={busy || resendIn > 0}
								onclick={requestCode}
							>
								{resendIn > 0 ? `Send a new code in ${resendIn} s` : 'Send a new code'}
							</button>
						</div>
					</form>
				{/if}
			</section>
		{/if}
	{/if}

	<ConfirmDialog bind:this={confirmDialog} />
</div>
