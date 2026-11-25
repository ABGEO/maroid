<script lang="ts">
	import { ApiError, failureMessage, type PluginHost } from '@maroid/plugin-sdk';
	import { ConfirmDialog } from '@maroid/ui';

	import '../../app.css';
	import { createTelecomApi, type Connection } from '../../api';
	import { kindLabel } from '../../lib/format';

	const SERVICE_UNAVAILABLE = '/problems/dev.maroid.magti/service-unavailable';

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createTelecomApi(host));
	const canConnect = $derived(host.can('account.connect'));

	let connection = $state<Connection | null>(null);
	let etag = $state<string | undefined>(undefined);
	let status = $state<'loading' | 'ready' | 'unavailable' | 'error'>('loading');
	let busy = $state(false);
	let failure = $state<string | null>(null);
	let selected = $state('');
	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();

	const connected = $derived(connection?.state === 'connected');
	const subscribers = $derived(
		(connection?.subscribers ?? []).filter((subscriber) => subscriber.kind !== 'other')
	);
	const assigned = $derived(
		subscribers.find((subscriber) => subscriber.assigned)?.subscriber_number ?? ''
	);

	function show(read: Connection): void {
		connection = read;
		selected = read.subscribers?.find((subscriber) => subscriber.assigned)?.subscriber_number ?? '';
		status = 'ready';
	}

	async function load(): Promise<void> {
		status = 'loading';
		failure = null;

		try {
			const read = await api.connection.read();
			if (read !== null) {
				etag = read.etag;
				show(read.value);
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
			if ((await api.connection.connect()) !== null) {
				await load();
			}
		}, 'The Magti account is not connected. Try again.');
	}

	function assign(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		return run(async () => {
			if ((await api.connection.assign(selected, etag)) !== null) {
				await load();
			}
		}, 'The subscriber is not assigned. Try again.');
	}

	async function disconnect(): Promise<void> {
		const proceed = await confirmDialog?.ask(
			'The workspace collects no traffic and no money entry until an editor connects again. The traffic, the money entries, and the settings stay.',
			{ title: 'Disconnect the Magti account?', confirmLabel: 'Disconnect' }
		);
		if (proceed) {
			await run(async () => {
				await api.connection.disconnect();
				await load();
			}, 'The Magti account stays connected. Try again.');
		}
	}

	$effect(() => {
		void load();
	});
</script>

<div class="max-w-xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">Telecom account</h1>
			<p class="page-lead">
				Connect the Magti account of this workspace, and assign one subscriber, so Maroid shows its
				balance and collects its traffic and its money entries.
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
			<span>Magti is unavailable. The connection stays.</span>
			<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
		</div>
	{:else if status === 'error'}
		<div role="alert" class="alert alert-error alert-soft">
			<span>Failed to load the Magti account.</span>
			<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
		</div>
	{:else if connected}
		<section class="panel mb-6 p-4">
			<div class="mb-4 flex items-center justify-between gap-2">
				<h2 class="font-medium">Connected</h2>
				<span class="badge badge-success badge-soft">Connected</span>
			</div>

			{#if subscribers.length === 0}
				<p class="meta mb-4">The Magti account holds no subscriber.</p>
			{:else}
				<form class="mb-4 flex flex-col gap-3" onsubmit={assign}>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">Subscriber of this workspace</legend>
						{#each subscribers as subscriber (subscriber.subscriber_number)}
							<label class="label w-full cursor-pointer justify-start gap-3 py-1">
								<input
									type="radio"
									class="radio radio-sm shrink-0"
									name="subscriber"
									value={subscriber.subscriber_number}
									bind:group={selected}
									disabled={busy || !canConnect}
								/>
								<span class="flex min-w-0 flex-1 flex-col">
									<span class="text-base-content flex min-w-0 items-center gap-2">
										<span class="shrink-0 font-mono">{subscriber.subscriber_number}</span>
										<span class="badge badge-ghost badge-sm shrink-0">
											{kindLabel(subscriber.kind)}
										</span>
										{#if subscriber.assigned}
											<span class="badge badge-primary badge-soft badge-sm shrink-0">Assigned</span>
										{/if}
									</span>
									{#if subscriber.label}
										<span class="meta truncate" title={subscriber.label}>{subscriber.label}</span>
									{/if}
								</span>
							</label>
						{/each}
					</fieldset>
					{#if canConnect}
						<div>
							<button
								type="submit"
								class="btn btn-primary btn-sm"
								disabled={busy || selected === '' || selected === assigned}
							>
								Assign
							</button>
						</div>
					{/if}
				</form>
			{/if}

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
				<p class="meta">An editor of the workspace connects the Magti account.</p>
			{:else if connection?.credentials_stored}
				<p class="meta mb-4">
					Maroid signs in to Magti with the phone number and the password in the settings.
				</p>
				<button type="button" class="btn btn-primary btn-sm" disabled={busy} onclick={connect}>
					Connect
				</button>
			{:else}
				<p class="meta">
					Store the phone number and the password of the Magti account in the
					<a class="link" href={host.href('/settings')}>settings</a> first.
				</p>
			{/if}
		</section>
	{/if}

	<ConfirmDialog bind:this={confirmDialog} />
</div>
