<script lang="ts">
	import { invalidateAll } from '$app/navigation';

	import { ApiError, PROBLEM_TYPE, api, type Workspace } from '$lib/api';
	import Alert from '$lib/components/ui/Alert.svelte';
	import { loadWorkspaces } from '$lib/state/workspaces.svelte';

	interface Props {
		workspace: Workspace;
		etag: string | undefined;
	}

	let { workspace, etag }: Props = $props();

	const MAX_NAME = 64;

	let name = $derived(workspace.name);
	let saving = $state(false);
	let outcome = $state<{ kind: 'success' | 'error'; text: string } | null>(null);

	const length = $derived([...name].length);
	const valid = $derived(length >= 1 && length <= MAX_NAME);

	function failureMessage(error: unknown): string {
		if (error instanceof ApiError && error.problem !== null) {
			if (error.is(PROBLEM_TYPE.preconditionFailed)) {
				return 'Someone renamed the workspace since you opened this page. Reload it and try again.';
			}

			return error.problem.errors?.[0]?.detail ?? error.problem.title;
		}

		return 'The name stays. Try again.';
	}

	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		if (saving || !valid) {
			return;
		}

		saving = true;
		outcome = null;

		try {
			const renamed = await api.workspaces.rename(workspace.id, name, etag);
			if (renamed === null) {
				return;
			}

			await Promise.all([invalidateAll(), loadWorkspaces()]);
			outcome = { kind: 'success', text: 'Saved.' };
		} catch (error) {
			outcome = { kind: 'error', text: failureMessage(error) };
		} finally {
			saving = false;
		}
	}
</script>

{#if outcome}
	<Alert kind={outcome.kind} class="mb-6"><span>{outcome.text}</span></Alert>
{/if}

<form class="panel max-w-md p-5" onsubmit={submit}>
	<fieldset class="fieldset">
		<legend class="fieldset-legend">Name</legend>
		<input type="text" class="input w-full" bind:value={name} required />
		<p class="label">{length} / {MAX_NAME}</p>
	</fieldset>

	<div class="mt-4">
		<button type="submit" class="btn btn-primary btn-sm" disabled={saving || !valid}>Save</button>
	</div>
</form>
