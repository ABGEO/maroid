<script lang="ts">
	import type { PluginHost } from '@maroid/plugin-sdk';

	import '../app.css';
	import { createCarLogApi } from '../api';

	let { host, page, title }: { host: PluginHost; page: string; title: string } = $props();

	const api = $derived(createCarLogApi(host));

	let status = $state<'loading' | 'empty' | 'error'>('loading');

	async function load(): Promise<void> {
		status = 'loading';

		try {
			const read = await api.vehicles();
			if (read === null) {
				return;
			}

			const chosen = read.items.find((vehicle) => vehicle.default);
			if (chosen === undefined) {
				status = 'empty';

				return;
			}

			host.navigate(`/${page}/${encodeURIComponent(chosen.id)}`);
		} catch (error) {
			console.error('Failed to load the vehicles', error);
			status = 'error';
		}
	}

	$effect(() => {
		void load();
	});
</script>

<div>
	<header class="page-header">
		<div>
			<h1 class="page-title">{title}</h1>
		</div>
	</header>

	{#if status === 'loading'}
		<span class="skeleton h-32 w-full" aria-hidden="true"></span>
	{:else if status === 'error'}
		<div role="alert" class="alert alert-error alert-soft max-w-xl">
			<span>Failed to load the vehicles.</span>
			<button type="button" class="btn btn-sm" onclick={load}>Retry</button>
		</div>
	{:else}
		<div role="alert" class="alert alert-info alert-soft max-w-xl">
			<span>Maroid holds no vehicle yet. The collection runs at the start of each hour.</span>
			<a class="btn btn-sm" href={host.href('/account')}>Open the account</a>
		</div>
	{/if}
</div>
