<script lang="ts">
	import { ApiError, type PluginHost } from '@maroid/plugin-sdk';

	import '../../app.css';
	import { createTelecomApi, type TrafficDay } from '../../api';
	import { addDays, tbilisiToday } from '../../lib/dates';
	import { megabytes, minutesAndSeconds } from '../../lib/format';
	import type { Measure } from '../../lib/measure';
	import TrafficChart from './TrafficChart.svelte';

	const NOT_ASSIGNED = '/problems/dev.maroid.magti/not-assigned';
	const HOME_LINE = '/problems/dev.maroid.magti/home-line';
	const PERIODS = [30, 90, 365] as const;

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createTelecomApi(host));

	let period = $state<(typeof PERIODS)[number]>(30);
	let days = $state<TrafficDay[]>([]);
	let dates = $state<string[]>([]);
	let latest = 0;
	let status = $state<'loading' | 'ready' | 'not_assigned' | 'home_line' | 'error'>('loading');

	const DATA: Measure[] = [
		{ key: 'data', name: 'Data, MB', read: (day) => day.data_mb, color: 'primary' }
	];
	const CALLS: Measure[] = [
		{
			key: 'out',
			name: 'Calls out',
			read: (day) => day.call_out_seconds,
			color: 'primary'
		},
		{ key: 'in', name: 'Calls in', read: (day) => day.call_in_seconds, color: 'secondary' }
	];
	const CALLS_IN_MINUTES: Measure[] = CALLS.map((measure) => ({
		...measure,
		read: (day) => {
			const seconds = measure.read(day);

			return seconds === undefined ? undefined : seconds / 60;
		}
	}));
	const SMS: Measure[] = [
		{ key: 'sent', name: 'SMS sent', read: (day) => day.sms_sent, color: 'primary' },
		{ key: 'received', name: 'SMS received', read: (day) => day.sms_received, color: 'secondary' }
	];

	function sum(read: (day: TrafficDay) => number | undefined): number {
		return days.reduce((total, day) => total + (read(day) ?? 0), 0);
	}

	const totals = $derived({
		data: sum((day) => day.data_mb),
		callOut: sum((day) => day.call_out_seconds),
		callIn: sum((day) => day.call_in_seconds),
		smsSent: sum((day) => day.sms_sent),
		smsReceived: sum((day) => day.sms_received)
	});

	function daysFrom(from: string, count: number): string[] {
		return Array.from({ length: count }, (_, index) => addDays(from, index));
	}

	async function load(): Promise<void> {
		const request = ++latest;
		status = 'loading';

		// The collection of a day runs on the next morning, so the period ends yesterday.
		const to = addDays(tbilisiToday(), -1);
		const from = addDays(to, 1 - period);
		const count = period;

		try {
			const read = await api.traffic.between(from, to);
			if (request !== latest) {
				return;
			}

			if (read !== null) {
				days = read;
				dates = daysFrom(from, count);
				status = 'ready';
			}
		} catch (error) {
			if (request !== latest) {
				return;
			}

			if (error instanceof ApiError && error.is(NOT_ASSIGNED)) {
				status = 'not_assigned';
			} else if (error instanceof ApiError && error.is(HOME_LINE)) {
				status = 'home_line';
			} else {
				console.error('Failed to load the traffic', error);
				status = 'error';
			}
		}
	}

	function choose(chosen: (typeof PERIODS)[number]): void {
		period = chosen;
		void load();
	}

	$effect(() => {
		void load();
	});
</script>

<div>
	<header class="page-header">
		<div>
			<h1 class="page-title">Traffic</h1>
			<p class="page-lead">
				The data, the calls, and the SMS of each day of the assigned subscriber.
			</p>
		</div>
		<div class="join" role="group" aria-label="Period">
			{#each PERIODS as option (option)}
				<button
					type="button"
					class={['join-item btn btn-sm', period === option && 'btn-active']}
					aria-pressed={period === option}
					onclick={() => choose(option)}
				>
					{option} days
				</button>
			{/each}
		</div>
	</header>

	{#if status === 'not_assigned'}
		<div role="alert" class="alert alert-info alert-soft max-w-xl">
			<span>Connect the Magti account and assign a mobile subscriber first.</span>
			<a class="btn btn-sm" href={host.href('/account')}>Open the account</a>
		</div>
	{:else if status === 'home_line'}
		<div class="empty-state max-w-xl">
			The assigned subscriber is a home line. Magti answers traffic for a mobile subscriber alone.
		</div>
	{:else if status === 'error'}
		<div role="alert" class="alert alert-error alert-soft max-w-xl">
			<span>Failed to load the traffic.</span>
			<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
		</div>
	{:else if status === 'loading'}
		<div class="flex flex-col gap-6" aria-hidden="true">
			<span class="skeleton h-20 w-full"></span>
			<span class="skeleton h-80 w-full"></span>
		</div>
	{:else if days.length === 0}
		<div class="empty-state">No traffic in the last {period} days.</div>
	{:else}
		<div class="flex flex-col gap-6">
			<dl class="panel grid gap-4 p-4 sm:grid-cols-3">
				<div>
					<dt class="meta">Data</dt>
					<dd class="text-xl font-semibold tabular-nums">{megabytes(totals.data)} MB</dd>
				</div>
				<div>
					<dt class="meta">Calls out and in</dt>
					<dd class="text-xl font-semibold tabular-nums">
						{minutesAndSeconds(totals.callOut)} / {minutesAndSeconds(totals.callIn)}
					</dd>
				</div>
				<div>
					<dt class="meta">SMS sent and received</dt>
					<dd class="text-xl font-semibold tabular-nums">
						{totals.smsSent} / {totals.smsReceived}
					</dd>
				</div>
			</dl>

			<section class="panel p-4" aria-labelledby="data-title">
				<h2 id="data-title" class="mb-2 font-medium">Data</h2>
				<TrafficChart
					{days}
					{dates}
					measures={DATA}
					shown={(value) => `${megabytes(value)} MB`}
					label="The data of each day, in MB."
				/>
			</section>

			<section class="panel p-4" aria-labelledby="calls-title">
				<h2 id="calls-title" class="mb-2 font-medium">Calls, minutes</h2>
				<TrafficChart
					{days}
					{dates}
					measures={CALLS_IN_MINUTES}
					shown={(value) => minutesAndSeconds(value * 60)}
					label="The minutes of the calls out and of the calls in of each day."
				/>
			</section>

			<section class="panel p-4" aria-labelledby="sms-title">
				<h2 id="sms-title" class="mb-2 font-medium">SMS</h2>
				<TrafficChart
					{days}
					{dates}
					measures={SMS}
					shown={(value) => String(value)}
					label="The SMS sent and received of each day."
				/>
			</section>
		</div>
	{/if}
</div>
