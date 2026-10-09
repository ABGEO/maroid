<script lang="ts">
	import { onMount } from 'svelte';

	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';

	import { LOCAL_PROVIDER, api, type ProviderIdentity } from '$lib/api';
	import { authFailureMessage } from '$lib/auth/messages';
	import ChangePasswordForm from '$lib/components/profile/ChangePasswordForm.svelte';
	import ConnectedAccounts from '$lib/components/profile/ConnectedAccounts.svelte';
	import ProfileNameForm from '$lib/components/profile/ProfileNameForm.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import Section from '$lib/components/ui/Section.svelte';

	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let identities = $state<ProviderIdentity[]>([]);
	let banner = $state<string | null>(null);

	const localAccount = $derived(
		identities.find((one) => one.provider === LOCAL_PROVIDER && one.attached)
	);

	async function load(): Promise<void> {
		status = 'loading';

		try {
			const result = await api.identities.list();
			if (result === null) {
				return;
			}

			identities = result;
			status = 'ready';
		} catch (error) {
			console.error('Failed to load the connected accounts', error);
			status = 'error';
		}
	}

	onMount(() => {
		const reason = page.url.searchParams.get('error');

		if (reason) {
			banner = authFailureMessage(reason);
			void goto(resolve('/profile'), { replaceState: true, keepFocus: true, noScroll: true });
		}

		void load();
	});
</script>

<div class="max-w-2xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">Profile</h1>
			<p class="page-lead">Your name, your password, and the accounts that reach this record.</p>
		</div>
	</header>

	{#if banner}
		<Alert kind="error" class="mb-6"><span>{banner}</span></Alert>
	{/if}

	<Section title="Name">
		<ProfileNameForm />
	</Section>

	{#if localAccount}
		<Section title="Password">
			<ChangePasswordForm username={localAccount.username} />
		</Section>
	{/if}

	<Section title="Connected accounts">
		<ConnectedAccounts {identities} {status} onreload={load} />
	</Section>
</div>
