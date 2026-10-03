<script lang="ts">
	import { invalidateAll } from '$app/navigation';

	import { ApiError, PROBLEM_TYPE, api, isProblem } from '$lib/api';
	import { PERMISSION, holds } from '$lib/permissions';
	import { loadWorkspaces } from '$lib/state/workspaces.svelte';

	import type { PageProps } from './$types';

	const MAX_NAME = 64;

	let { data }: PageProps = $props();

	let name = $state('');
	let saving = $state(false);
	let banner = $state<{ kind: 'success' | 'error'; text: string } | null>(null);

	const length = $derived([...name].length);
	const valid = $derived(length >= 1 && length <= MAX_NAME);

	$effect(() => {
		name = data.workspace.name;
	});

	function failureMessage(error: unknown): string {
		if (error instanceof ApiError && isProblem(error.body)) {
			if (error.body.type === PROBLEM_TYPE.preconditionFailed) {
				return 'Someone renamed the workspace since you opened this page. Reload it and try again.';
			}

			return error.body.errors?.[0]?.detail ?? error.body.title;
		}

		return 'The name stays. Try again.';
	}

	async function save(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		if (saving || !valid) {
			return;
		}

		saving = true;
		banner = null;

		try {
			const renamed = await api.workspaces.rename(data.workspace.id, name, data.etag);
			if (renamed === null) {
				return;
			}

			await Promise.all([invalidateAll(), loadWorkspaces()]);
			banner = { kind: 'success', text: 'Saved.' };
		} catch (error) {
			banner = { kind: 'error', text: failureMessage(error) };
		} finally {
			saving = false;
		}
	}
</script>

<div class="max-w-2xl">
	<h1 class="font-display text-[42px] leading-[1.05] tracking-tight">Settings</h1>
	<p class="text-base-content/60 mt-3 text-sm">The name that every member sees.</p>

	{#if banner}
		<div
			class="alert mt-6"
			class:alert-success={banner.kind === 'success'}
			class:alert-error={banner.kind === 'error'}
		>
			<span>{banner.text}</span>
		</div>
	{/if}

	{#if !holds(data.workspace, PERMISSION.workspaceWrite)}
		<p class="mt-6 text-sm">{data.workspace.name}</p>
		<p class="text-base-content/60 mt-1 text-sm">A manager renames the workspace.</p>
	{:else}
		<form class="mt-6 flex max-w-md flex-col gap-3" onsubmit={save}>
			<label class="form-control">
				<span class="label-text mb-1 block text-xs">Name</span>
				<input type="text" class="input input-bordered w-full" bind:value={name} required />
				<span class="text-base-content/50 mt-1 text-[11px]">{length} / {MAX_NAME}</span>
			</label>

			<div>
				<button type="submit" class="btn btn-primary btn-sm" disabled={saving || !valid}>
					Save
				</button>
			</div>
		</form>
	{/if}
</div>
