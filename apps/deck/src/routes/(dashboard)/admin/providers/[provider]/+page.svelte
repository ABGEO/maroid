<script lang="ts">
	import { page } from '$app/state';

	import { ApiError, api, type Provider } from '$lib/api';
	import ProviderForm from '$lib/components/providers/ProviderForm.svelte';
	import RemoveProvider from '$lib/components/providers/RemoveProvider.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import CopyField from '$lib/components/ui/CopyField.svelte';
	import Panel from '$lib/components/ui/Panel.svelte';
	import Section from '$lib/components/ui/Section.svelte';
	import { presetLabel } from '$lib/providers/form';

	const providerId = $derived(page.params.provider ?? '');

	let status = $state<'loading' | 'ready' | 'missing' | 'error'>('loading');
	let provider = $state<Provider | null>(null);
	let etag = $state<string | undefined>(undefined);

	const takesClient = $derived(provider?.preset === 'telegram' || provider?.preset === 'oidc');

	async function read(): Promise<void> {
		const tagged = await api.providers.get(providerId);
		if (tagged === null) {
			return;
		}

		provider = tagged.value;
		etag = tagged.etag;
		status = 'ready';
	}

	async function load(): Promise<void> {
		status = 'loading';

		try {
			await read();
		} catch (error) {
			if (error instanceof ApiError && error.status === 404) {
				status = 'missing';

				return;
			}

			console.error('Failed to load the provider', error);
			status = 'error';
		}
	}

	$effect(() => {
		void providerId;
		void load();
	});
</script>

<div class="max-w-2xl">
	{#if status === 'loading'}
		<div class="flex flex-col gap-2" aria-hidden="true">
			<span class="skeleton h-12 w-64"></span>
			{#each [0, 1, 2] as i (i)}
				<span class="skeleton h-24 w-full"></span>
			{/each}
		</div>
	{:else if status === 'missing'}
		<header class="page-header">
			<div>
				<h1 class="page-title">Not found</h1>
				<p class="page-lead">Dex holds no provider with this identifier.</p>
			</div>
		</header>
	{:else if status === 'error'}
		<Alert kind="error"><span>The hub answered no provider. Reload the page.</span></Alert>
	{:else if provider}
		<header class="page-header">
			<div>
				<h1 class="page-title">{provider.name}</h1>
				<div class="mt-2 flex flex-wrap items-center gap-2">
					<span class="badge badge-outline badge-sm">{presetLabel(provider)}</span>
					<span class="meta font-mono">{provider.id}</span>
				</div>
			</div>
		</header>

		{#if provider.static}
			<Alert kind="info" class="mb-6">
				<span class="text-sm">
					The configuration file of Dex holds this provider, so Maroid changes nothing on it. Edit
					the file and restart Dex to change it.
				</span>
			</Alert>
		{:else}
			<Section title="Provider">
				<ProviderForm {provider} {etag} onsaved={read} />
			</Section>

			{#if takesClient && provider.redirect_uri}
				<Section title="Redirect address">
					<Panel>
						<span class="label whitespace-normal">
							{provider.preset === 'telegram'
								? 'Set the domain of this address in BotFather.'
								: 'Register this address at the provider.'}
						</span>
						<CopyField value={provider.redirect_uri} label="The redirect address of Dex" />
					</Panel>
				</Section>
			{/if}

			<Section title="Remove" danger>
				<RemoveProvider {provider} />
			</Section>
		{/if}
	{/if}
</div>
