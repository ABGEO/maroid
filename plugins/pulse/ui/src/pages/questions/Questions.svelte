<script lang="ts">
	import { failureMessage, type PluginHost } from '@maroid/plugin-sdk';
	import { ConfirmDialog, createPager, Pagination } from '@maroid/ui';

	import '../../app.css';
	import { createPulseApi, type Question } from '../../api';
	import OffPanel from '../../lib/OffPanel.svelte';
	import { payloadOf } from '../../lib/question';
	import { createPulseState } from '../../lib/pulse.svelte';
	import { TYPE_LABELS } from '../../lib/text';
	import ZoneHeader from '../../lib/ZoneHeader.svelte';
	import QuestionForm from './QuestionForm.svelte';

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createPulseApi(host));
	const pulse = createPulseState(() => host);

	let archived = $state(false);
	let form = $state<{ question?: Question; etag?: string } | null>(null);
	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();
	let busy = $state(false);
	let failure = $state<string | null>(null);

	const pager = createPager((link?: string) => api.questions.list(archived, link));

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

	async function edit(question: Question): Promise<void> {
		failure = null;

		try {
			const read = await api.questions.get(question.id);
			if (read !== null) {
				form = { question: read.value, etag: read.etag };
			}
		} catch (error) {
			failure = failureMessage(error, 'The question did not load. Try again.');
		}
	}

	function setArchived(question: Question, value: boolean): Promise<void> {
		return run(
			() => api.questions.update(question.id, payloadOf(question, value)),
			value ? 'The question stays active. Try again.' : 'The question stays archived. Try again.'
		);
	}

	async function remove(question: Question): Promise<void> {
		const proceed = await confirmDialog?.ask(`"${question.prompt}" goes for good.`, {
			title: 'Delete this question?',
			confirmLabel: 'Delete'
		});
		if (proceed) {
			await run(() => api.questions.remove(question.id), 'The question stays. Try again.');
		}
	}

	function show(next: boolean): void {
		archived = next;
		void pager.reload();
	}

	function saved(): void {
		form = null;
		void pager.reload();
	}

	$effect(() => {
		void pager.reload();
		void pulse.load();
	});
</script>

<div class="max-w-3xl">
	<ZoneHeader
		{host}
		title="Questions"
		lead="What pulse asks. One question can sit in many check-ins."
	/>

	{#if pulse.enabled === false}
		<OffPanel {host} />
	{/if}

	{#if form}
		<QuestionForm
			{api}
			question={form.question}
			etag={form.etag}
			onsaved={saved}
			oncancel={() => (form = null)}
		/>
	{:else}
		<button type="button" class="btn btn-primary btn-sm mb-4" onclick={() => (form = {})}>
			New question
		</button>
	{/if}

	<div role="tablist" class="tabs tabs-border mb-4">
		<button
			type="button"
			role="tab"
			class={['tab', !archived && 'tab-active']}
			aria-selected={!archived}
			onclick={() => show(false)}
		>
			Active
		</button>
		<button
			type="button"
			role="tab"
			class={['tab', archived && 'tab-active']}
			aria-selected={archived}
			onclick={() => show(true)}
		>
			Archive
		</button>
	</div>

	{#if failure}
		<div role="alert" class="alert alert-error alert-soft mb-4">
			<span>{failure}</span>
		</div>
	{/if}

	{#if pager.status === 'loading' && pager.page === null}
		<div class="flex flex-col gap-2" aria-hidden="true">
			{#each [0, 1, 2] as i (i)}
				<span class="skeleton h-12 w-full"></span>
			{/each}
		</div>
	{:else if pager.status === 'error'}
		<div role="alert" class="alert alert-error alert-soft">
			<span>Failed to load the questions.</span>
			<button type="button" class="btn btn-sm" onclick={() => pager.reload()}>Retry</button>
		</div>
	{:else if pager.page?.items.length === 0}
		<div class="empty-state">
			{archived ? 'The archive holds no question.' : 'You have no active question.'}
		</div>
	{:else}
		<ul class="list panel">
			{#each pager.page?.items ?? [] as question (question.id)}
				<li class="list-row items-center">
					<div class="list-col-grow">
						<div class="font-medium">{question.prompt}</div>
						<div class="meta">
							{TYPE_LABELS[question.type]},
							{question.value_count}
							{question.value_count === 1 ? 'value' : 'values'}
						</div>
					</div>
					<button
						type="button"
						class="btn btn-ghost btn-sm"
						disabled={busy}
						onclick={() => edit(question)}
					>
						Change
					</button>
					<button
						type="button"
						class="btn btn-ghost btn-sm"
						disabled={busy}
						onclick={() => setArchived(question, !question.archived)}
					>
						{question.archived ? 'Restore' : 'Archive'}
					</button>
					{#if question.value_count === 0}
						<button
							type="button"
							class="btn btn-ghost btn-sm text-error"
							disabled={busy}
							onclick={() => remove(question)}
						>
							Delete
						</button>
					{/if}
				</li>
			{/each}
		</ul>
		<Pagination {pager} class="mt-4 ml-auto w-fit" />
	{/if}
</div>

<ConfirmDialog bind:this={confirmDialog} />
