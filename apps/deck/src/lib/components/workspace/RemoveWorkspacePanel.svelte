<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import type { Workspace } from '#lib/api/index.ts';
	import Panel from '#lib/components/ui/Panel.svelte';
	import { loadWorkspaces } from '#lib/state/workspaces.svelte.ts';

	import RemoveWorkspaceDialog from './RemoveWorkspaceDialog.svelte';

	interface Props {
		workspace: Workspace;
	}

	let { workspace }: Props = $props();

	let removal = $state<ReturnType<typeof RemoveWorkspaceDialog>>();

	async function removed(): Promise<void> {
		await loadWorkspaces();
		await goto(resolve(''));
	}
</script>

<Panel class="border-error/40 mt-8 max-w-md">
	<legend class="fieldset-legend">Remove the workspace</legend>
	<span class="label whitespace-normal">
		The removal deletes every record of the workspace, and cannot be undone.
	</span>
	<div class="flex justify-end">
		<button type="button" class="btn btn-error btn-sm" onclick={() => removal?.open(workspace)}>
			Remove the workspace
		</button>
	</div>
</Panel>

<RemoveWorkspaceDialog bind:this={removal} onremoved={removed} />
