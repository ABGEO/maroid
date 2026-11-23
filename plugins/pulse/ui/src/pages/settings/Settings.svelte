<script lang="ts">
	import {
		failureMessage,
		formatMoment,
		toMoment,
		today,
		type PluginHost
	} from '@maroid/plugin-sdk';

	import '../../app.css';
	import { createPulseApi, type Settings } from '../../api';
	import { addDays } from '../../lib/days';
	import { scheduleZone } from '../../lib/pulse.svelte';
	import ZoneHeader from '../../lib/ZoneHeader.svelte';

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createPulseApi(host));
	const zone = $derived(scheduleZone(host));

	let settings = $state<Settings | null>(null);
	let etag = $state<string | undefined>(undefined);
	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let busy = $state(false);
	let failure = $state<string | null>(null);
	let pauseDay = $state('');

	const pausedUntil = $derived(
		settings?.paused_until && Date.parse(settings.paused_until) > Date.now()
			? settings.paused_until
			: null
	);

	async function load(): Promise<void> {
		status = 'loading';

		try {
			const read = await api.settings.get();
			settings = read?.value ?? null;
			etag = read?.etag;
			status = 'ready';
		} catch {
			status = 'error';
		}
	}

	async function save(enabled: boolean, paused: string | null, fallback: string): Promise<void> {
		if (busy) {
			return;
		}

		busy = true;
		failure = null;

		try {
			await api.settings.put({ enabled, paused_until: paused }, etag);
			await load();
		} catch (error) {
			failure = failureMessage(error, fallback);
		} finally {
			busy = false;
		}
	}

	function toggle(): void {
		if (settings === null) {
			return;
		}

		// The hub refuses a pause that has passed, so a past one goes as null.
		void save(!settings.enabled, pausedUntil, 'Pulse stays as it was. Try again.');
	}

	function pause(event: SubmitEvent): void {
		event.preventDefault();

		const until = toMoment(`${pauseDay}T00:00`, zone);
		if (until === undefined || settings === null) {
			return;
		}

		void save(settings.enabled, until, 'The pause was not set. Try again.');
		pauseDay = '';
	}

	function resume(): void {
		if (settings !== null) {
			void save(settings.enabled, null, 'The pause stays. Try again.');
		}
	}

	$effect(() => {
		void load();
	});
</script>

<div class="max-w-2xl">
	<ZoneHeader {host} title="Settings" lead="Turn the check-ins on or off, or pause them." />

	{#if failure}
		<div role="alert" class="alert alert-error alert-soft mb-4">
			<span>{failure}</span>
		</div>
	{/if}

	{#if status === 'loading' && settings === null}
		<span class="skeleton block h-32 w-full" aria-hidden="true"></span>
	{:else if status === 'error' || settings === null}
		<div role="alert" class="alert alert-error alert-soft">
			<span>Failed to load the settings.</span>
			<button type="button" class="btn btn-sm" onclick={() => load()}>Retry</button>
		</div>
	{:else}
		<section class="panel mb-6 flex items-center gap-4 p-4">
			<div class="grow">
				<h2 class="font-medium">Check-ins</h2>
				<p class="meta">
					{#if settings.enabled}
						Pulse asks your questions on Telegram at the times of your check-ins.
					{:else}
						Pulse sends nothing. The first turn-on adds the default questions and check-ins.
					{/if}
				</p>
			</div>
			<input
				type="checkbox"
				class="toggle toggle-primary"
				aria-label="Pulse is on"
				checked={settings.enabled}
				disabled={busy}
				onchange={toggle}
			/>
		</section>

		<section class="panel p-4">
			<h2 class="font-medium">Pause</h2>
			{#if pausedUntil}
				<p class="meta mb-3">
					Paused until {formatMoment(pausedUntil, zone, {
						dateStyle: 'medium',
						timeStyle: 'short'
					})}. A check-in of the pause never fires later.
				</p>
				<button type="button" class="btn btn-sm" disabled={busy} onclick={resume}>Resume</button>
			{:else}
				<p class="meta mb-3">
					No check-in fires until 00:00 of the day you choose. On Telegram, send /pause with a date
					or a duration, and /resume.
				</p>
				<form class="flex items-end gap-2" onsubmit={pause}>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">Until</legend>
						<input
							type="date"
							class="input"
							min={addDays(today(zone), 1)}
							bind:value={pauseDay}
							disabled={busy || !settings.enabled}
							required
						/>
					</fieldset>
					<button
						type="submit"
						class="btn btn-primary btn-sm mb-1"
						disabled={busy || !settings.enabled || pauseDay === ''}
					>
						Pause
					</button>
				</form>
			{/if}
		</section>
	{/if}
</div>
