<script lang="ts">
	import { createWriteIntent, failureMessage } from '@maroid/plugin-sdk';
	import { untrack } from 'svelte';

	import type { Checkin, CheckinPayload, PulseApi, Question } from '../../api';
	import { weekdayLabel } from '../../lib/text';

	const MAX_NAME = 40;
	const MAX_TIMES = 6;
	const MAX_QUESTIONS = 20;
	const WEEKDAYS = [1, 2, 3, 4, 5, 6, 7];

	let {
		api,
		checkin,
		etag,
		questions,
		onsaved,
		oncancel
	}: {
		api: PulseApi;
		/** The check-in to change. Absent for a new check-in. */
		checkin?: Checkin;
		etag?: string;
		/** The questions that a check-in can ask: the ones that are not archived. */
		questions: Question[];
		onsaved: () => void;
		oncancel: () => void;
	} = $props();

	const intent = createWriteIntent();

	// The form copies the check-in once, when it opens.
	const initial = untrack(() => checkin);

	let name = $state(initial?.name ?? '');
	let times = $state<string[]>(initial ? [...initial.local_times] : ['09:00']);
	let weekdays = $state<number[]>(initial ? [...initial.weekdays] : []);
	let enabled = $state(initial?.enabled ?? true);
	let chosen = $state<string[]>(initial ? [...initial.question_ids] : []);
	let adding = $state('');
	let busy = $state(false);
	let failure = $state<string | null>(null);

	const byID = $derived(new Map(questions.map((question) => [question.id, question])));
	const addable = $derived(questions.filter((question) => !chosen.includes(question.id)));

	function toggleDay(day: number): void {
		weekdays = weekdays.includes(day)
			? weekdays.filter((one) => one !== day)
			: [...weekdays, day].sort((a, b) => a - b);
	}

	function move(index: number, by: number): void {
		const target = index + by;
		[chosen[index], chosen[target]] = [chosen[target], chosen[index]];
	}

	function add(): void {
		if (adding !== '') {
			chosen.push(adding);
			adding = '';
		}
	}

	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (busy) {
			return;
		}

		busy = true;
		failure = null;

		const body: CheckinPayload = {
			name: name.trim(),
			local_times: [...new Set(times)].sort(),
			weekdays,
			enabled,
			question_ids: chosen
		};

		try {
			if (checkin) {
				await api.checkins.update(checkin.id, body, etag);
			} else {
				await api.checkins.create(body, { idempotencyKey: intent.keyFor(body) });
				intent.settle();
			}

			onsaved();
		} catch (error) {
			failure = failureMessage(error, 'The check-in was not saved. Try again.');
		} finally {
			busy = false;
		}
	}
</script>

<form class="panel mb-6 flex flex-col gap-3 p-4" onsubmit={submit}>
	<h2 class="font-medium">{checkin ? 'Edit the check-in' : 'New check-in'}</h2>

	{#if failure}
		<div role="alert" class="alert alert-error alert-soft">
			<span>{failure}</span>
		</div>
	{/if}

	<div class="flex flex-wrap items-end gap-3">
		<fieldset class="fieldset grow">
			<legend class="fieldset-legend">Name</legend>
			<input
				class="input w-full"
				placeholder="Morning"
				maxlength={MAX_NAME}
				bind:value={name}
				disabled={busy}
				required
			/>
		</fieldset>
		<fieldset class="fieldset">
			<legend class="fieldset-legend">State</legend>
			<label class="label h-10 gap-2">
				<input
					type="checkbox"
					class="toggle toggle-primary"
					bind:checked={enabled}
					disabled={busy}
				/>
				On
			</label>
		</fieldset>
	</div>

	<fieldset class="fieldset">
		<legend class="fieldset-legend">Times, 1 to 6</legend>
		<div class="flex flex-wrap gap-2">
			{#each times as _, index (index)}
				<div class="join">
					<input
						type="time"
						class="input join-item w-32"
						aria-label="Time {index + 1}"
						bind:value={times[index]}
						disabled={busy}
						required
					/>
					<button
						type="button"
						class="btn join-item"
						aria-label="Remove time {index + 1}"
						disabled={busy || times.length <= 1}
						onclick={() => times.splice(index, 1)}
					>
						✕
					</button>
				</div>
			{/each}
			<button
				type="button"
				class="btn"
				disabled={busy || times.length >= MAX_TIMES}
				onclick={() => times.push('12:00')}
			>
				Add a time
			</button>
		</div>
	</fieldset>

	<fieldset class="fieldset">
		<legend class="fieldset-legend">Weekdays. None is every day.</legend>
		<div class="flex flex-wrap gap-3">
			{#each WEEKDAYS as day (day)}
				<label class="label gap-1">
					<input
						type="checkbox"
						class="checkbox checkbox-sm"
						checked={weekdays.includes(day)}
						disabled={busy}
						onchange={() => toggleDay(day)}
					/>
					{weekdayLabel(day)}
				</label>
			{/each}
		</div>
	</fieldset>

	<fieldset class="fieldset">
		<legend class="fieldset-legend">Questions, in the order of the round, 1 to 20</legend>
		{#if chosen.length > 0}
			<ol class="list panel">
				{#each chosen as id, index (id)}
					<li class="list-row items-center py-2">
						<span class="meta tabular-nums">{index + 1}</span>
						<span class="list-col-grow">{byID.get(id)?.prompt ?? 'An archived question'}</span>
						<button
							type="button"
							class="btn btn-ghost btn-xs"
							aria-label="Move up"
							disabled={busy || index === 0}
							onclick={() => move(index, -1)}
						>
							↑
						</button>
						<button
							type="button"
							class="btn btn-ghost btn-xs"
							aria-label="Move down"
							disabled={busy || index === chosen.length - 1}
							onclick={() => move(index, 1)}
						>
							↓
						</button>
						<button
							type="button"
							class="btn btn-ghost btn-xs"
							disabled={busy}
							onclick={() => chosen.splice(index, 1)}
						>
							Remove
						</button>
					</li>
				{/each}
			</ol>
		{/if}
		<div class="mt-2 flex gap-2">
			<select
				class="select grow"
				aria-label="A question to add"
				bind:value={adding}
				disabled={busy || chosen.length >= MAX_QUESTIONS}
			>
				<option value="">Choose a question</option>
				{#each addable as question (question.id)}
					<option value={question.id}>{question.prompt}</option>
				{/each}
			</select>
			<button type="button" class="btn" disabled={busy || adding === ''} onclick={add}>
				Add
			</button>
		</div>
	</fieldset>

	<div class="mt-1 flex items-center gap-2">
		<button
			type="submit"
			class="btn btn-primary btn-sm"
			disabled={busy || name.trim() === '' || chosen.length === 0}
		>
			{busy ? 'Saving…' : 'Save'}
		</button>
		<button type="button" class="btn btn-ghost btn-sm" onclick={oncancel}>Cancel</button>
	</div>
</form>
