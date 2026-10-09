<script lang="ts">
	import type { PageProps } from './$types';
	import SettingsForm from '$lib/components/settings/SettingsForm.svelte';
	import { knownPlugin } from '$lib/state/plugins.svelte';
	import { PERMISSION, holds } from '$lib/permissions';
	import { displayNameOf } from '$lib/plugins/capabilities';

	let { data }: PageProps = $props();

	const plugin = $derived(knownPlugin(data.pluginId));
	const name = $derived(displayNameOf(plugin, data.pluginId));
</script>

<div class="max-w-4xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">{name}</h1>
			<p class="meta mt-1 font-mono">{data.pluginId}</p>
			<p class="page-lead">
				These values belong to this workspace. A field marked personal belongs to you alone, and its
				value serves every workspace of yours. A secret never leaves the hub once you store it.
			</p>
		</div>
	</header>

	<SettingsForm
		workspaceId={data.workspace.id}
		pluginId={data.pluginId}
		readonly={!holds(data.workspace, PERMISSION.settingsWrite)}
	/>
</div>
