<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import { ApiError, api, createWriteIntent, isProblem } from '$lib/api';
	import { loadWorkspaces } from '$lib/state/workspaces.svelte';

	const MAX_NAME = 64;

	const intent = createWriteIntent();

	let name = $state('');
	let submitting = $state(false);
	let failure = $state<string | null>(null);

	const length = $derived([...name].length);
	const valid = $derived(length >= 1 && length <= MAX_NAME);

	function failureMessage(error: unknown): string {
		if (error instanceof ApiError && isProblem(error.body)) {
			return error.body.errors?.[0]?.detail ?? error.body.title;
		}

		return 'The hub created no workspace. Try again.';
	}

	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		if (submitting || !valid) {
			return;
		}

		submitting = true;
		failure = null;

		try {
			const created = await api.workspaces.create(name, intent.keyFor({ name }));
			if (created === null) {
				return;
			}

			intent.settle();
			await loadWorkspaces();
			await goto(resolve('/(dashboard)/w/[workspace]', { workspace: created.id }));
		} catch (error) {
			failure = failureMessage(error);
		} finally {
			submitting = false;
		}
	}
</script>

<form class="panel max-w-md p-5" onsubmit={submit}>
	<fieldset class="fieldset">
		<legend class="fieldset-legend">Name</legend>
		<input type="text" class="input w-full" placeholder="Home" bind:value={name} required />
		<p class="label">{length} / {MAX_NAME}</p>
	</fieldset>

	{#if failure}
		<div role="alert" class="alert alert-error alert-soft mt-4">
			<span>{failure}</span>
		</div>
	{/if}

	<div class="mt-4">
		<button type="submit" class="btn btn-primary btn-sm" disabled={submitting || !valid}>
			Create workspace
		</button>
	</div>
</form>
