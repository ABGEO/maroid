<script lang="ts">
	import { ApiError, type PluginHost } from '@maroid/plugin-sdk';

	import '../../app.css';
	import { createTelecomApi, type SubscriberState } from '../../api';
	import { kindLabel, lari, megabytes, minutesAndSeconds, renewal } from '../../lib/format';

	const SERVICE_UNAVAILABLE = '/problems/dev.maroid.magti/service-unavailable';

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createTelecomApi(host));

	let subscriber = $state<SubscriberState | null>(null);
	let status = $state<'loading' | 'ready' | 'not_set_up' | 'unavailable' | 'error'>('loading');

	const renewsOn = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeZone: 'UTC' });

	async function load(): Promise<void> {
		status = 'loading';

		try {
			const read = await api.state.read();
			if (read !== null) {
				subscriber = read;
				status = 'ready';
			}
		} catch (error) {
			if (error instanceof ApiError && error.status === 409) {
				status = 'not_set_up';
			} else if (error instanceof ApiError && error.is(SERVICE_UNAVAILABLE)) {
				status = 'unavailable';
			} else {
				console.error('Failed to load the state of the subscriber', error);
				status = 'error';
			}
		}
	}

	$effect(() => {
		void load();
	});
</script>

<div class="max-w-3xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">Subscriber</h1>
			<p class="page-lead">
				The balance and the plan of the assigned subscriber, read from Magti at each visit.
			</p>
		</div>
		{#if status === 'ready'}
			<button type="button" class="btn btn-sm" onclick={load}>Read again</button>
		{/if}
	</header>

	{#if status === 'loading'}
		<div class="grid gap-4 sm:grid-cols-2" aria-hidden="true">
			<span class="skeleton h-32 w-full"></span>
			<span class="skeleton h-32 w-full"></span>
		</div>
	{:else if status === 'not_set_up'}
		<div role="alert" class="alert alert-info alert-soft max-w-xl">
			<span>Connect the Magti account and assign a subscriber first.</span>
			<a class="btn btn-sm" href={host.href('/account')}>Open the account</a>
		</div>
	{:else if status === 'unavailable'}
		<div role="alert" class="alert alert-warning alert-soft max-w-xl">
			<span>Magti is unavailable.</span>
			<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
		</div>
	{:else if status === 'error' || subscriber === null}
		<div role="alert" class="alert alert-error alert-soft max-w-xl">
			<span>Failed to load the subscriber.</span>
			<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
		</div>
	{:else}
		<p class="meta mb-4 flex flex-wrap items-center gap-2">
			<span class="font-mono">{subscriber.subscriber_number}</span>
			<span class="badge badge-ghost badge-sm">{kindLabel(subscriber.kind)}</span>
			{#if subscriber.status}
				<span>{subscriber.status}</span>
			{/if}
		</p>

		<div class="grid gap-4 sm:grid-cols-2">
			<section class="panel p-4" aria-labelledby="balance-title">
				<h2 id="balance-title" class="meta mb-1">Balance</h2>
				<p class="text-3xl font-semibold tabular-nums">{lari(subscriber.balance.amount)}</p>
			</section>

			<section class="panel p-4" aria-labelledby="plan-title">
				<h2 id="plan-title" class="meta mb-1">Plan</h2>
				{#if subscriber.plan}
					<p class="text-3xl font-semibold tabular-nums">{lari(subscriber.plan.price.amount)}</p>
					<p class="mt-1">
						{#if subscriber.plan.name}
							<span class="font-medium">{subscriber.plan.name}</span>,
						{/if}
						{renewal(subscriber.plan.days_left)}{#if subscriber.plan.renews_on}
							, on {renewsOn.format(new Date(`${subscriber.plan.renews_on}T00:00:00Z`))}{/if}
					</p>
					{#if subscriber.speed}
						<p class="meta mt-1">{subscriber.speed}</p>
					{/if}
				{:else}
					<p class="text-base-content/70">No active plan</p>
				{/if}
			</section>
		</div>

		{#if subscriber.allowances}
			<section class="panel mt-4 p-4" aria-labelledby="allowances-title">
				<h2 id="allowances-title" class="mb-3 font-medium">What remains of the plan</h2>
				<dl class="grid gap-4 sm:grid-cols-3">
					<div>
						<dt class="meta">Data</dt>
						<dd class="text-xl font-semibold tabular-nums">
							{megabytes(subscriber.allowances.data_mb)} MB
						</dd>
					</div>
					<div>
						<dt class="meta">Calls</dt>
						<dd class="text-xl font-semibold tabular-nums">
							{minutesAndSeconds(subscriber.allowances.call_seconds)}
						</dd>
					</div>
					<div>
						<dt class="meta">SMS</dt>
						<dd class="text-xl font-semibold tabular-nums">{subscriber.allowances.sms}</dd>
					</div>
				</dl>
			</section>
		{/if}
	{/if}
</div>
