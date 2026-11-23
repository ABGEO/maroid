<script lang="ts">
	import { failureMessage, formatMoment, today, type PluginHost } from '@maroid/plugin-sdk';
	import { createPager, Pagination } from '@maroid/ui';

	import '../../app.css';
	import { createPulseApi, type Day, type DayRange, type Question } from '../../api';
	import { addDays } from '../../lib/days';
	import OffPanel from '../../lib/OffPanel.svelte';
	import { readAll } from '../../lib/pages';
	import { createPulseState, scheduleZone } from '../../lib/pulse.svelte';
	import { mapLink, TYPE_LABELS, valueText } from '../../lib/text';
	import ZoneHeader from '../../lib/ZoneHeader.svelte';
	import DayChart from './DayChart.svelte';

	const RANGES = [7, 30, 365];
	const CHARTED = new Set(['scale', 'number', 'boolean']);

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createPulseApi(host));
	const zone = $derived(scheduleZone(host));
	const pulse = createPulseState(() => host);

	let questions = $state<Question[]>([]);
	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let selected = $state<Question | null>(null);
	let span = $state(30);
	let days = $state<Day[]>([]);
	let failure = $state<string | null>(null);

	const range = $derived<DayRange>({ from: addDays(today(zone), 1 - span), to: today(zone) });
	const values = createPager((link?: string) =>
		selected === null ? Promise.resolve(null) : api.questions.values(selected.id, range, link)
	);

	async function loadQuestions(): Promise<void> {
		status = 'loading';

		try {
			const [active, archived] = await Promise.all([
				readAll((link) => api.questions.list(false, link)),
				readAll((link) => api.questions.list(true, link))
			]);
			questions = [...active, ...archived];
			status = 'ready';
		} catch {
			status = 'error';
		}
	}

	async function loadSelected(): Promise<void> {
		if (selected === null) {
			return;
		}

		failure = null;
		days = [];
		void values.reload();

		if (!CHARTED.has(selected.type)) {
			return;
		}

		try {
			days = (await api.questions.days(selected.id, range))?.items ?? [];
		} catch (error) {
			failure = failureMessage(error, 'The chart did not load. Try again.');
		}
	}

	function choose(question: Question): void {
		selected = question;
		void loadSelected();
	}

	function setSpan(next: number): void {
		span = next;
		void loadSelected();
	}

	function moment(value: string): string {
		return formatMoment(value, zone, { dateStyle: 'medium', timeStyle: 'short' });
	}

	$effect(() => {
		void loadQuestions();
		void pulse.load();
	});
</script>

<div class="max-w-4xl">
	<ZoneHeader {host} title="Data" lead="Your answers, one question at a time." />

	{#if pulse.enabled === false}
		<OffPanel {host} />
	{/if}

	{#if status === 'loading'}
		<div class="flex flex-col gap-2" aria-hidden="true">
			{#each [0, 1, 2] as i (i)}
				<span class="skeleton h-12 w-full"></span>
			{/each}
		</div>
	{:else if status === 'error'}
		<div role="alert" class="alert alert-error alert-soft">
			<span>Failed to load the questions.</span>
			<button type="button" class="btn btn-sm" onclick={() => loadQuestions()}>Retry</button>
		</div>
	{:else if questions.length === 0}
		<div class="empty-state">You have no question yet.</div>
	{:else}
		<ul class="list panel mb-6">
			{#each questions as question (question.id)}
				<li>
					<button
						type="button"
						class={[
							'list-row w-full items-center text-left',
							selected?.id === question.id && 'bg-base-200'
						]}
						aria-pressed={selected?.id === question.id}
						onclick={() => choose(question)}
					>
						<div class="list-col-grow">
							<div class="font-medium">
								{question.prompt}
								{#if question.archived}<span class="badge badge-ghost badge-sm ml-1">Archived</span
									>{/if}
							</div>
							<div class="meta">
								{TYPE_LABELS[question.type]},
								{question.value_count}
								{question.value_count === 1 ? 'value' : 'values'}
							</div>
						</div>
						{#if question.latest}
							<div class="text-right">
								<div class="tabular-nums">{valueText(question, question.latest)}</div>
								<div class="meta">{moment(question.latest.occurred_at)}</div>
							</div>
						{/if}
					</button>
				</li>
			{/each}
		</ul>

		{#if selected}
			{@const question = selected}
			<section class="panel p-4">
				<div class="mb-4 flex flex-wrap items-center gap-2">
					<h2 class="grow font-medium">{question.prompt}</h2>
					<div class="join">
						{#each RANGES as count (count)}
							<button
								type="button"
								class={['btn btn-sm join-item', span === count && 'btn-active']}
								aria-pressed={span === count}
								onclick={() => setSpan(count)}
							>
								{count} days
							</button>
						{/each}
					</div>
				</div>

				{#if failure}
					<div role="alert" class="alert alert-error alert-soft mb-4">
						<span>{failure}</span>
					</div>
				{/if}

				{#if CHARTED.has(question.type) && days.length > 0}
					<div class="mb-6">
						<DayChart {question} {days} />
					</div>
				{/if}

				{#if values.status === 'loading' && values.page === null}
					<span class="skeleton block h-24 w-full" aria-hidden="true"></span>
				{:else if values.status === 'error'}
					<div role="alert" class="alert alert-error alert-soft">
						<span>Failed to load the values.</span>
						<button type="button" class="btn btn-sm" onclick={() => values.reload()}>Retry</button>
					</div>
				{:else if values.page?.items.length === 0}
					<div class="empty-state">No value in these {span} days.</div>
				{:else}
					<ul class="list">
						{#each values.page?.items ?? [] as value (value.id)}
							<li class="list-row items-center">
								<span class="meta tabular-nums">{moment(value.occurred_at)}</span>
								<span class="list-col-grow tabular-nums">{valueText(question, value)}</span>
								{#if question.type === 'location'}
									<a class="link" href={mapLink(value)} target="_blank" rel="noopener noreferrer">
										Open the map
									</a>
								{/if}
							</li>
						{/each}
					</ul>
					<Pagination pager={values} class="mt-4 ml-auto w-fit" />
				{/if}
			</section>
		{/if}
	{/if}
</div>
