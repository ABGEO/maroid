<script lang="ts">
	import { api, type Provider } from '$lib/api';
	import AddProviderForm from '$lib/components/providers/AddProviderForm.svelte';
	import ProviderList from '$lib/components/providers/ProviderList.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import CopyField from '$lib/components/ui/CopyField.svelte';
	import Section from '$lib/components/ui/Section.svelte';
	import SkeletonList from '$lib/components/ui/SkeletonList.svelte';

	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let providers = $state<Provider[]>([]);

	const redirectUri = $derived(providers.find((one) => one.redirect_uri)?.redirect_uri ?? '');
	const taken = $derived(new Set(providers.map((one) => one.id)));

	async function load(): Promise<void> {
		status = 'loading';

		try {
			const list = await api.providers.list();
			if (list === null) {
				return;
			}

			providers = list;
			status = 'ready';
		} catch (error) {
			console.error('Failed to load the providers', error);
			status = 'error';
		}
	}

	$effect(() => {
		void load();
	});
</script>

<div class="max-w-3xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">Sign-in providers</h1>
			<p class="page-lead">
				The accounts that a person signs in with. A change reaches the sign in page of Dex at the
				next sign in.
			</p>
		</div>
	</header>

	{#if redirectUri}
		<Alert kind="info" class="mb-6 flex flex-col items-start gap-2">
			<span class="text-sm">
				Register this redirect address at an OpenID Connect provider, or set its domain in BotFather
				for Telegram, before you add the provider here.
			</span>
			<CopyField value={redirectUri} label="The redirect address of Dex" />
		</Alert>
	{/if}

	<Section title="Add a provider">
		<AddProviderForm {taken} />
	</Section>

	<Section title="All providers">
		{#if status === 'loading'}
			<SkeletonList />
		{:else if status === 'error'}
			<Alert kind="error" class="mb-6">
				<span>The hub answered no provider. Reload the page.</span>
			</Alert>
		{:else}
			<ProviderList {providers} />
		{/if}
	</Section>
</div>
