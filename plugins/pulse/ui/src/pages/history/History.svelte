<script lang="ts">
	import { formatMoment, type PluginHost, type RouteParams } from '@maroid/plugin-sdk';
	import { createPager, Pagination } from '@maroid/ui';
	import { untrack } from 'svelte';
	import { SvelteSet } from 'svelte/reactivity';

	import '../../app.css';
	import {
		createPulseApi,
		type Checkin,
		type Question,
		type Round,
		type RoundQuestion,
		type RoundState
	} from '../../api';
	import OffPanel from '../../lib/OffPanel.svelte';
	import { readAll } from '../../lib/pages';
	import { createPulseState, scheduleZone } from '../../lib/pulse.svelte';
	import { mapLink, valueText } from '../../lib/text';
	import ZoneHeader from '../../lib/ZoneHeader.svelte';

	const STATES: Record<RoundState, { label: string; badge: string }> = {
		offered: { label: 'Waiting to start', badge: 'badge-info' },
		started: { label: 'In progress', badge: 'badge-info' },
		done: { label: 'Done', badge: 'badge-success' },
		declined: { label: 'Declined', badge: 'badge-ghost' },
		expired: { label: 'Expired', badge: 'badge-warning' }
	};

	const OUTCOMES: Record<Exclude<RoundQuestion['state'], 'answered'>, string> = {
		pending: 'Not asked',
		waiting: 'Waiting for the answer',
		skipped: 'Skipped',
		expired: 'Expired'
	};

	let { host, params = {} }: { host: PluginHost; params?: RouteParams } = $props();

	const api = $derived(createPulseApi(host));
	const zone = $derived(scheduleZone(host));
	const pulse = createPulseState(() => host);

	let checkinId = $state('');
	let roundState = $state<RoundState | ''>('');
	let questions = $state(new Map<string, Question>());
	let checkins = $state<Checkin[]>([]);
	let status = $state<'loading' | 'ready' | 'error'>('loading');
	const opened = new SvelteSet<string>();

	function listRounds() {
		const filter = { checkin_id: checkinId || undefined, state: roundState || undefined };

		return createPager((link?: string) => api.rounds.list(filter, link));
	}

	let pager = $state(listRounds());

	async function loadReferences(): Promise<void> {
		status = 'loading';

		try {
			const [active, archived, listed] = await Promise.all([
				readAll((link) => api.questions.list(false, link)),
				readAll((link) => api.questions.list(true, link)),
				readAll((link) => api.checkins.list(link))
			]);
			questions = new Map([...active, ...archived].map((question) => [question.id, question]));
			checkins = listed;
			status = 'ready';
		} catch {
			status = 'error';
		}
	}

	function narrow(): void {
		opened.clear();
		pager = listRounds();
		void pager.reload();
	}

	function chooseCheckin(id: string): void {
		host.navigate(id === '' ? '/history' : `/history/${id}`);
	}

	function toggle(round: Round): void {
		if (opened.has(round.id)) {
			opened.delete(round.id);
		} else {
			opened.add(round.id);
		}
	}

	function answered(round: Round): number {
		return round.questions.filter((question) => question.state === 'answered').length;
	}

	function outcome(asked: RoundQuestion): string {
		const question = questions.get(asked.question_id);

		if (asked.state !== 'answered') {
			return OUTCOMES[asked.state];
		}

		return question && asked.value ? valueText(question, asked.value) : '';
	}

	function moment(value: string): string {
		return formatMoment(value, zone, { dateStyle: 'medium', timeStyle: 'short' });
	}

	$effect(() => {
		const fromRoute = params.checkin ?? '';

		untrack(() => {
			checkinId = fromRoute;
			narrow();
		});
	});

	$effect(() => {
		untrack(() => {
			void loadReferences();
			void pulse.load();
		});
	});

	$effect(() => {
		if (params.checkin === undefined || status === 'loading') {
			return;
		}

		const named = checkins.find((checkin) => checkin.id === params.checkin);
		host.setTitle(named?.name ?? 'Not found');
	});
</script>

<div class="max-w-3xl">
	<ZoneHeader {host} title="History" lead="Every check-in that pulse sent, newest first." />

	{#if pulse.enabled === false}
		<OffPanel {host} />
	{/if}

	<div class="mb-4 flex flex-wrap gap-2">
		<select
			class="select select-sm w-auto"
			aria-label="Check-in"
			value={checkinId}
			onchange={(event) => chooseCheckin(event.currentTarget.value)}
		>
			<option value="">Every check-in</option>
			{#each checkins as checkin (checkin.id)}
				<option value={checkin.id}>{checkin.name}</option>
			{/each}
		</select>
		<select
			class="select select-sm w-auto"
			aria-label="State"
			bind:value={roundState}
			onchange={narrow}
		>
			<option value="">Every state</option>
			{#each Object.entries(STATES) as [value, state] (value)}
				<option {value}>{state.label}</option>
			{/each}
		</select>
	</div>

	{#if status === 'error'}
		<div role="alert" class="alert alert-error alert-soft">
			<span>Failed to load the questions and the check-ins.</span>
			<button type="button" class="btn btn-sm" onclick={() => loadReferences()}>Retry</button>
		</div>
	{:else if (status === 'loading' || pager.status === 'loading') && pager.page === null}
		<div class="flex flex-col gap-2" aria-hidden="true">
			{#each [0, 1, 2] as i (i)}
				<span class="skeleton h-12 w-full"></span>
			{/each}
		</div>
	{:else if pager.status === 'error'}
		<div role="alert" class="alert alert-error alert-soft">
			<span>Failed to load the history.</span>
			<button type="button" class="btn btn-sm" onclick={() => pager.reload()}>Retry</button>
		</div>
	{:else if pager.page?.items.length === 0}
		<div class="empty-state">
			{checkinId || roundState ? 'No check-in matches.' : 'No check-in has fired yet.'}
		</div>
	{:else}
		<ul class="list panel">
			{#each pager.page?.items ?? [] as round (round.id)}
				{@const open = opened.has(round.id)}
				<li>
					<button
						type="button"
						class="list-row w-full items-center text-left"
						aria-expanded={open}
						aria-controls={open ? `round-${round.id}` : undefined}
						onclick={() => toggle(round)}
					>
						<svg
							class={['size-4 shrink-0 transition-transform', open && 'rotate-90']}
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
							aria-hidden="true"
						>
							<path d="m9 18 6-6-6-6" />
						</svg>
						<span class="meta whitespace-nowrap tabular-nums">{moment(round.occurrence_at)}</span>
						<span class="list-col-grow font-medium">{round.checkin_name}</span>
						<span class={['badge badge-soft badge-sm', STATES[round.state].badge]}>
							{STATES[round.state].label}
						</span>
						{#if round.state !== 'declined'}
							<span class="meta whitespace-nowrap tabular-nums">
								{answered(round)} of {round.questions.length} answered
							</span>
						{/if}
					</button>
					{#if open}
						<ol id="round-{round.id}" class="border-base-300 mx-4 mb-3 border-t pt-2">
							{#each round.questions as asked, index (asked.question_id)}
								{@const question = questions.get(asked.question_id)}
								<li class="flex items-baseline gap-3 py-1.5 text-sm">
									<span class="meta w-4 shrink-0 text-right tabular-nums">{index + 1}</span>
									<span class="grow">{question?.prompt ?? 'A deleted question'}</span>
									<span
										class={['text-right', asked.state === 'answered' ? 'tabular-nums' : 'meta']}
									>
										{outcome(asked)}
										{#if question?.type === 'location' && asked.value}
											<a
												class="link ml-2"
												href={mapLink(asked.value)}
												target="_blank"
												rel="noopener noreferrer"
											>
												Open the map
											</a>
										{/if}
									</span>
								</li>
							{/each}
						</ol>
					{/if}
				</li>
			{/each}
		</ul>
		<Pagination {pager} class="mt-4 ml-auto w-fit" />
	{/if}
</div>
