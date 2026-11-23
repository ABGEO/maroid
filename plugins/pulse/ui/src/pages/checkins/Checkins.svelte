<script lang="ts">
	import { failureMessage, type PluginHost } from '@maroid/plugin-sdk';
	import { ConfirmDialog, createPager, Pagination } from '@maroid/ui';

	import '../../app.css';
	import { createPulseApi, type Checkin, type Question } from '../../api';
	import OffPanel from '../../lib/OffPanel.svelte';
	import { readAll } from '../../lib/pages';
	import { createPulseState } from '../../lib/pulse.svelte';
	import { weekdaysText } from '../../lib/text';
	import ZoneHeader from '../../lib/ZoneHeader.svelte';
	import CheckinForm from './CheckinForm.svelte';

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createPulseApi(host));
	const pulse = createPulseState(() => host);
	const pager = createPager((link?: string) => api.checkins.list(link));

	let questions = $state<Question[]>([]);
	let form = $state<{ checkin?: Checkin; etag?: string } | null>(null);
	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();
	let busy = $state(false);
	let failure = $state<string | null>(null);

	async function loadQuestions(): Promise<void> {
		try {
			questions = await readAll((link) => api.questions.list(false, link));
		} catch (error) {
			failure = failureMessage(error, 'The questions did not load. Reload the page.');
		}
	}

	async function run(action: () => Promise<unknown>, fallback: string): Promise<void> {
		if (busy) {
			return;
		}

		busy = true;
		failure = null;

		try {
			await action();
			await pager.reload();
		} catch (error) {
			failure = failureMessage(error, fallback);
		} finally {
			busy = false;
		}
	}

	async function edit(checkin: Checkin): Promise<void> {
		failure = null;

		try {
			const read = await api.checkins.get(checkin.id);
			if (read !== null) {
				form = { checkin: read.value, etag: read.etag };
			}
		} catch (error) {
			failure = failureMessage(error, 'The check-in did not load. Try again.');
		}
	}

	function setEnabled(checkin: Checkin): Promise<void> {
		const { name, local_times, weekdays, question_ids } = checkin;

		return run(
			() =>
				api.checkins.update(checkin.id, {
					name,
					local_times,
					weekdays,
					question_ids,
					enabled: !checkin.enabled
				}),
			'The check-in stays as it was. Try again.'
		);
	}

	async function remove(checkin: Checkin): Promise<void> {
		const proceed = await confirmDialog?.ask(`"${checkin.name}" goes. Its values stay.`, {
			title: 'Delete this check-in?',
			confirmLabel: 'Delete'
		});
		if (proceed) {
			await run(() => api.checkins.remove(checkin.id), 'The check-in stays. Try again.');
		}
	}

	function openHistory(event: MouseEvent, checkin: Checkin): void {
		event.preventDefault();
		host.navigate(`/history/${checkin.id}`);
	}

	function saved(): void {
		form = null;
		void pager.reload();
	}

	$effect(() => {
		void pager.reload();
		void pulse.load();
		void loadQuestions();
	});
</script>

<div class="max-w-3xl">
	<ZoneHeader {host} title="Check-ins" lead="When pulse asks, and which questions, on Telegram.">
		{#snippet actions()}
			{#if form === null}
				<button type="button" class="btn btn-primary btn-sm" onclick={() => (form = {})}>
					Add check-in
				</button>
			{/if}
		{/snippet}
	</ZoneHeader>

	{#if pulse.enabled === false}
		<OffPanel {host} />
	{/if}

	{#if failure}
		<div role="alert" class="alert alert-error alert-soft mb-4">
			<span>{failure}</span>
		</div>
	{/if}

	{#if form}
		<CheckinForm
			{api}
			{questions}
			checkin={form.checkin}
			etag={form.etag}
			onsaved={saved}
			oncancel={() => (form = null)}
		/>
	{/if}

	{#if pager.status === 'loading' && pager.page === null}
		<div class="flex flex-col gap-2" aria-hidden="true">
			{#each [0, 1, 2] as i (i)}
				<span class="skeleton h-12 w-full"></span>
			{/each}
		</div>
	{:else if pager.status === 'error'}
		<div role="alert" class="alert alert-error alert-soft">
			<span>Failed to load the check-ins.</span>
			<button type="button" class="btn btn-sm" onclick={() => pager.reload()}>Retry</button>
		</div>
	{:else if pager.page?.items.length === 0}
		<div class="empty-state">You have no check-in.</div>
	{:else}
		<ul class="list panel">
			{#each pager.page?.items ?? [] as checkin (checkin.id)}
				<li class="list-row items-center">
					<input
						type="checkbox"
						class="toggle toggle-primary toggle-sm"
						aria-label="{checkin.name} is on"
						checked={checkin.enabled}
						disabled={busy}
						onchange={() => setEnabled(checkin)}
					/>
					<div class="list-col-grow">
						<div class="font-medium">{checkin.name}</div>
						<div class="meta">
							{checkin.local_times.join(', ')}, {weekdaysText(checkin.weekdays)},
							{checkin.question_ids.length}
							{checkin.question_ids.length === 1 ? 'question' : 'questions'}
						</div>
					</div>
					<button
						type="button"
						class="btn btn-ghost btn-sm"
						disabled={busy}
						onclick={() => edit(checkin)}
					>
						Edit
					</button>
					<a
						class="btn btn-ghost btn-sm"
						href={host.href(`/history/${checkin.id}`)}
						onclick={(event) => openHistory(event, checkin)}
					>
						History
					</a>
					<button
						type="button"
						class="btn btn-ghost btn-sm text-error"
						disabled={busy}
						onclick={() => remove(checkin)}
					>
						Delete
					</button>
				</li>
			{/each}
		</ul>
		<Pagination {pager} class="mt-4 ml-auto w-fit" />
	{/if}
</div>

<ConfirmDialog bind:this={confirmDialog} />
