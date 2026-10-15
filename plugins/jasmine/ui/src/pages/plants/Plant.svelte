<script lang="ts">
	import type { PluginHost, RouteParams } from '@maroid/plugin-sdk';

	import '../../app.css';
	import { createJasmineApi, type EnvironmentReference } from '../../api';
	import { readAll } from '../../lib/pages';
	import PlantView from './PlantView.svelte';

	let { host, params }: { host: PluginHost; params: RouteParams } = $props();

	const api = $derived(createJasmineApi(host));

	let environments = $state<EnvironmentReference[]>([]);

	$effect(() => {
		readAll((link?: string) => api.environments.list(link))
			.then((items) => (environments = items))
			.catch((error) => console.error('Failed to load the environments', error));
	});
</script>

<div class="max-w-5xl">
	<PlantView {host} {api} plantId={params.id} {environments} />
</div>
