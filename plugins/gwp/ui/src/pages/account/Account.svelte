<script lang="ts">
	import { ApiError, failureMessage, type PluginHost } from '@maroid/plugin-sdk';
	import { ConfirmDialog } from '@maroid/ui';

	import '../../app.css';
	import { createGwpApi, type Connection } from '../../api';

	const SERVICE_UNAVAILABLE = '/problems/dev.maroid.gwp/service-unavailable';

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createGwpApi(host));
	const canConnect = $derived(host.can('account.connect'));

	let connection = $state<Connection | null>(null);
	let status = $state<'loading' | 'ready' | 'unavailable' | 'error'>('loading');
	let busy = $state(false);
	let failure = $state<string | null>(null);
	let selected = $state('');
	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();

	const connected = $derived(connection?.state === 'connected');
	const customers = $derived(connection?.customers ?? []);
	const assigned = $derived(customers.find((customer) => customer.assigned)?.customer_number ?? '');

	function show(read: Connection): void {
		connection = read;
		selected = read.customers?.find((customer) => customer.assigned)?.customer_number ?? '';
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
		}, 'The water account is not connected. Try again.');
	}

	function assign(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		return run(async () => {
			const read = await api.connection.assign(selected);
			if (read !== null) {
				show(read);
			}
		}, 'The customer is not assigned. Try again.');
	}

	async function disconnect(): Promise<void> {
		const proceed = await confirmDialog?.ask(
			'The workspace collects no reading until an editor connects again. The readings and the settings stay.',
			{ title: 'Disconnect the water account?', confirmLabel: 'Disconnect' }
		);
		if (proceed) {
			await run(async () => {
				await api.connection.disconnect();
				await load();
			}, 'The water account stays connected. Try again.');
		}
	}

	$effect(() => {
		void load();
	});
</script>

<div class="max-w-xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">Water account</h1>
			<p class="page-lead">
				Connect the GWP account of this workspace, and assign the customer of the house, so Maroid
				collects its readings.
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
			<span>GWP is unavailable. The connection stays.</span>
			<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
		</div>
	{:else if status === 'error'}
		<div role="alert" class="alert alert-error alert-soft">
			<span>Failed to load the water account.</span>
			<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
		</div>
	{:else if connected}
		<section class="panel mb-6 p-4">
			<div class="mb-4 flex items-center justify-between gap-2">
				<h2 class="font-medium">Connected</h2>
				<span class="badge badge-success badge-soft">Connected</span>
			</div>

			{#if customers.length === 0}
				<p class="meta mb-4">The water account holds no customer.</p>
			{:else}
				<form class="mb-4 flex flex-col gap-3" onsubmit={assign}>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">Customer of this workspace</legend>
						{#each customers as customer (customer.customer_number)}
							<label class="label cursor-pointer justify-start gap-3 py-1">
								<input
									type="radio"
									class="radio radio-sm"
									name="customer"
									value={customer.customer_number}
									bind:group={selected}
									disabled={busy || !canConnect}
								/>
								<span class="flex flex-col">
									<span class="text-base-content">
										<span class="font-mono">{customer.customer_number}</span>
										{#if customer.communal_number}
											<span class="text-base-content/60 text-sm">
												· Communal no. <span class="font-mono">{customer.communal_number}</span>
											</span>
										{/if}
									</span>
									<span class="meta">{customer.address}</span>
								</span>
								{#if customer.assigned}
									<span class="badge badge-primary badge-soft badge-sm">Assigned</span>
								{/if}
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
					<button type="button" class="btn btn-sm" disabled={busy} onclick={connect}
						>Connect again</button
					>
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
				<p class="meta">An editor of the workspace connects the water account.</p>
			{:else if connection?.credentials_stored}
				<p class="meta mb-4">
					Maroid signs in to GWP with the username and the password in the settings.
				</p>
				<button type="button" class="btn btn-primary btn-sm" disabled={busy} onclick={connect}
					>Connect</button
				>
			{:else}
				<p class="meta">
					Store the username and the password of the GWP account in the
					<a class="link" href={host.href('/settings')}>settings</a> first.
				</p>
			{/if}
		</section>
	{/if}

	<ConfirmDialog bind:this={confirmDialog} />
</div>
