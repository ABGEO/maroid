<script lang="ts">
	import { createWriteIntent, failureMessage } from '@maroid/plugin-sdk';
	import { untrack } from 'svelte';

	import type { Choice, PulseApi, Question, QuestionPayload, QuestionType } from '../../api';
	import { choicesOf, numberOf, scaleOf, TYPE_LABELS } from '../../lib/text';

	const MAX_PROMPT = 200;
	const MAX_LABEL = 32;
	const MAX_UNIT = 16;
	const MAX_CHOICES = 12;

	let {
		api,
		question,
		etag,
		onsaved,
		oncancel
	}: {
		api: PulseApi;
		/** The question to change. Absent for a new question. */
		question?: Question;
		etag?: string;
		onsaved: () => void;
		oncancel: () => void;
	} = $props();

	const intent = createWriteIntent();
	const typeLocked = $derived(question !== undefined && question.value_count > 0);

	// The form copies the question once, when it opens.
	const initial = untrack(() => question);
	const scale = initial ? scaleOf(initial) : undefined;
	const number = initial ? numberOf(initial) : undefined;

	let type = $state<QuestionType>(initial?.type ?? 'scale');
	let prompt = $state(initial?.prompt ?? '');
	let scaleMin = $state(scale?.min ?? 1);
	let scaleMax = $state(scale?.max ?? 5);
	let lowLabel = $state(scale?.low_label ?? '');
	let highLabel = $state(scale?.high_label ?? '');
	let numberMin = $state(number?.min ?? 0);
	let numberMax = $state(number?.max ?? 10);
	let unit = $state(number?.unit ?? '');
	let quickValues = $state(number?.quick_values?.join(', ') ?? '');
	let choices = $state<Choice[]>(
		initial ? choicesOf(initial).map((choice) => ({ ...choice })) : [{ label: '' }, { label: '' }]
	);
	let busy = $state(false);
	let failure = $state<string | null>(null);

	function payload(): QuestionPayload {
		const body: QuestionPayload = {
			type,
			prompt: prompt.trim(),
			archived: question?.archived ?? false
		};

		switch (type) {
			case 'scale':
				body.config = {
					min: scaleMin,
					max: scaleMax,
					...(lowLabel.trim() ? { low_label: lowLabel.trim() } : {}),
					...(highLabel.trim() ? { high_label: highLabel.trim() } : {})
				};
				break;
			case 'number': {
				const quick = quickValues
					.split(',')
					.map((part) => part.trim())
					.filter((part) => part !== '')
					.map(Number);
				body.config = {
					min: numberMin,
					max: numberMax,
					...(unit.trim() ? { unit: unit.trim() } : {}),
					...(quick.length > 0 ? { quick_values: quick } : {})
				};
				break;
			}
			case 'choice':
				body.config = {
					choices: choices.map((choice) =>
						choice.id
							? { id: choice.id, label: choice.label.trim() }
							: { label: choice.label.trim() }
					)
				};
				break;
		}

		return body;
	}

	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (busy) {
			return;
		}

		busy = true;
		failure = null;

		try {
			const body = payload();
			if (question) {
				await api.questions.update(question.id, body, etag);
			} else {
				await api.questions.create(body, { idempotencyKey: intent.keyFor(body) });
				intent.settle();
			}

			onsaved();
		} catch (error) {
			failure = failureMessage(error, 'The question was not saved. Try again.');
		} finally {
			busy = false;
		}
	}
</script>

<form class="panel mb-6 flex flex-col gap-3 p-4" onsubmit={submit}>
	<h2 class="font-medium">{question ? 'Change the question' : 'New question'}</h2>

	{#if failure}
		<div role="alert" class="alert alert-error alert-soft">
			<span>{failure}</span>
		</div>
	{/if}

	<div class="flex flex-wrap gap-3">
		<fieldset class="fieldset">
			<legend class="fieldset-legend">Type</legend>
			<select class="select" bind:value={type} disabled={busy || typeLocked}>
				{#each Object.entries(TYPE_LABELS) as [value, label] (value)}
					<option {value}>{label}</option>
				{/each}
			</select>
			{#if typeLocked}
				<p class="label">A question with a value keeps its type.</p>
			{/if}
		</fieldset>
		<fieldset class="fieldset grow">
			<legend class="fieldset-legend">Prompt</legend>
			<input
				class="input w-full"
				placeholder="How is your mood?"
				maxlength={MAX_PROMPT}
				bind:value={prompt}
				disabled={busy}
				required
			/>
		</fieldset>
	</div>

	{#if type === 'scale'}
		<div class="flex flex-wrap gap-3">
			<fieldset class="fieldset w-24">
				<legend class="fieldset-legend">From</legend>
				<input
					class="input"
					type="number"
					step="1"
					bind:value={scaleMin}
					disabled={busy}
					required
				/>
			</fieldset>
			<fieldset class="fieldset w-24">
				<legend class="fieldset-legend">To</legend>
				<input
					class="input"
					type="number"
					step="1"
					bind:value={scaleMax}
					disabled={busy}
					required
				/>
			</fieldset>
			<fieldset class="fieldset grow">
				<legend class="fieldset-legend">Label of the lowest</legend>
				<input
					class="input w-full"
					placeholder="bad"
					maxlength={MAX_LABEL}
					bind:value={lowLabel}
					disabled={busy}
				/>
			</fieldset>
			<fieldset class="fieldset grow">
				<legend class="fieldset-legend">Label of the highest</legend>
				<input
					class="input w-full"
					placeholder="great"
					maxlength={MAX_LABEL}
					bind:value={highLabel}
					disabled={busy}
				/>
			</fieldset>
		</div>
	{:else if type === 'number'}
		<div class="flex flex-wrap gap-3">
			<fieldset class="fieldset w-28">
				<legend class="fieldset-legend">Lowest</legend>
				<input
					class="input"
					type="number"
					step="any"
					bind:value={numberMin}
					disabled={busy}
					required
				/>
			</fieldset>
			<fieldset class="fieldset w-28">
				<legend class="fieldset-legend">Highest</legend>
				<input
					class="input"
					type="number"
					step="any"
					bind:value={numberMax}
					disabled={busy}
					required
				/>
			</fieldset>
			<fieldset class="fieldset w-28">
				<legend class="fieldset-legend">Unit</legend>
				<input
					class="input"
					placeholder="kg"
					maxlength={MAX_UNIT}
					bind:value={unit}
					disabled={busy}
				/>
			</fieldset>
			<fieldset class="fieldset grow">
				<legend class="fieldset-legend">Buttons, up to 8</legend>
				<input
					class="input w-full"
					placeholder="0, 1, 2, 3"
					bind:value={quickValues}
					disabled={busy}
				/>
			</fieldset>
		</div>
	{:else if type === 'choice'}
		<fieldset class="fieldset">
			<legend class="fieldset-legend">Choices, 2 to 12</legend>
			{#each choices as choice, index (index)}
				<div class="flex gap-2">
					<input
						class="input grow"
						aria-label="Choice {index + 1}"
						maxlength={MAX_LABEL}
						bind:value={choice.label}
						disabled={busy}
						required
					/>
					<button
						type="button"
						class="btn btn-ghost btn-sm"
						disabled={busy || choices.length <= 2}
						onclick={() => choices.splice(index, 1)}
					>
						Remove
					</button>
				</div>
			{/each}
			<button
				type="button"
				class="btn btn-sm mt-1 w-fit"
				disabled={busy || choices.length >= MAX_CHOICES}
				onclick={() => choices.push({ label: '' })}
			>
				Add a choice
			</button>
			<p class="label">A choice that a value records stays until its values go.</p>
		</fieldset>
	{/if}

	<div class="flex gap-2">
		<button type="submit" class="btn btn-primary btn-sm" disabled={busy || prompt.trim() === ''}>
			Save
		</button>
		<button type="button" class="btn btn-ghost btn-sm" onclick={oncancel}>Cancel</button>
	</div>
</form>
