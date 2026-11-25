<script lang="ts">
	import { failureMessage, type PluginHost } from '@maroid/plugin-sdk';
	import { ConfirmDialog } from '@maroid/ui';

	import '../app.css';
	import { createCarLogApi, type Connection } from '../api';
	import { isProblem, SERVICE_UNAVAILABLE } from '../lib/problems';

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createCarLogApi(host));
	const canConnect = $derived(host.can('account.connect'));

	let connection = $state<Connection | null>(null);
	let status = $state<'loading' | 'ready' | 'unavailable' | 'error'>('loading');
	let busy = $state(false);
	let failure = $state<string | null>(null);
	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();

	const connected = $derived(connection?.state === 'connected');

	async function load(): Promise<void> {
		status = 'loading';
		failure = null;

		try {
			const read = await api.connection.read();
			if (read !== null) {
				connection = read.value;
				status = 'ready';
			}
		} catch (error) {
			status = isProblem(error, SERVICE_UNAVAILABLE) ? 'unavailable' : 'error';
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
		}, 'The Drivvo account is not connected. Try again.');
	}

	async function disconnect(): Promise<void> {
		const proceed = await confirmDialog?.ask(
			'The workspace collects no log entry until an editor connects again. The vehicles, the log entries, and the settings stay.',
			{ title: 'Disconnect the Drivvo account?', confirmLabel: 'Disconnect' }
		);
		if (proceed) {
			await run(async () => {
				await api.connection.disconnect();
				await load();
			}, 'The Drivvo account stays connected. Try again.');
		}
	}

	$effect(() => {
		void load();
	});
</script>

<div class="max-w-xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">Car log account</h1>
			<p class="page-lead">
				Connect the Drivvo account of this workspace, so Maroid collects its vehicles, its log
				entries, and its reminders at the start of each hour.
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
			<span>Drivvo is unavailable. The connection stays.</span>
			<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
		</div>
	{:else if status === 'error'}
		<div role="alert" class="alert alert-error alert-soft">
			<span>Failed to load the Drivvo account.</span>
			<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
		</div>
	{:else if connected}
		<section class="panel mb-6 p-4">
			<div class="mb-4 flex items-center justify-between gap-2">
				<h2 class="font-medium">Connected</h2>
				<span class="badge badge-success badge-soft">Connected</span>
			</div>

			{#if (connection?.vehicles ?? []).length === 0}
				<p class="meta mb-4">The Drivvo account holds no vehicle.</p>
			{:else}
				<ul class="mb-4 flex flex-col gap-2">
					{#each connection?.vehicles ?? [] as vehicle, index (index)}
						<li class="flex min-w-0 flex-col">
							<span class="flex min-w-0 items-center gap-2">
								<span class="truncate font-medium">{vehicle.name}</span>
								{#if !vehicle.active}
									<span class="badge badge-ghost badge-sm shrink-0">Inactive</span>
								{/if}
							</span>
							<span class="meta truncate">
								{[vehicle.make, vehicle.model].filter(Boolean).join(' ')}
							</span>
						</li>
					{/each}
				</ul>
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
				<p class="meta">An editor of the workspace connects the Drivvo account.</p>
			{:else if connection?.credentials_stored}
				<p class="meta mb-4">
					Maroid signs in to Drivvo with the email address and the password in the settings.
				</p>
				<button type="button" class="btn btn-primary btn-sm" disabled={busy} onclick={connect}>
					Connect
				</button>
			{:else}
				<p class="meta">
					Store the email address and the password of the Drivvo account in the
					<a class="link" href={host.href('/settings')}>settings</a> first.
				</p>
			{/if}
		</section>
	{/if}

	<ConfirmDialog bind:this={confirmDialog} />
</div>
