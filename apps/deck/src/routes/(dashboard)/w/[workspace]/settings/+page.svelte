<script lang="ts">
	import RenameWorkspaceForm from '$lib/components/workspace/RenameWorkspaceForm.svelte';
	import { PERMISSION, holds } from '$lib/permissions';

	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<div class="max-w-2xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">Settings</h1>
			<p class="page-lead">The name that every member sees.</p>
		</div>
	</header>

	{#if !holds(data.workspace, PERMISSION.workspaceWrite)}
		<div class="panel max-w-md p-5">
			<p class="font-medium">{data.workspace.name}</p>
			<p class="meta mt-1">A manager renames the workspace.</p>
		</div>
	{:else}
		<RenameWorkspaceForm workspace={data.workspace} etag={data.etag} />
	{/if}
</div>
