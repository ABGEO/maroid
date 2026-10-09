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
	<header class="page-header">
		<div>
			<h1 class="page-title">Settings</h1>
			<p class="page-lead">The name that every member sees.</p>
		</div>
	</header>

	{#if banner}
		<div
			role="alert"
			class="alert alert-soft mb-6"
			class:alert-success={banner.kind === 'success'}
			class:alert-error={banner.kind === 'error'}
		>
			<span>{banner.text}</span>
		</div>
	{/if}

	{#if !holds(data.workspace, PERMISSION.workspaceWrite)}
		<div class="panel max-w-md p-5">
			<p class="font-medium">{data.workspace.name}</p>
			<p class="meta mt-1">A manager renames the workspace.</p>
		</div>
	{:else}
		<form class="panel max-w-md p-5" onsubmit={save}>
			<fieldset class="fieldset">
				<legend class="fieldset-legend">Name</legend>
				<input type="text" class="input w-full" bind:value={name} required />
				<p class="label">{length} / {MAX_NAME}</p>
			</fieldset>

			<div class="mt-4">
				<button type="submit" class="btn btn-primary btn-sm" disabled={saving || !valid}>
					Save
				</button>
			</div>
		</form>
	{/if}
</div>
