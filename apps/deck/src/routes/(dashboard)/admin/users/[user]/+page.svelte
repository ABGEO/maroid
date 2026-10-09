<script lang="ts">
	import { page } from '$app/state';

	import {
		ApiError,
		LOCAL_PROVIDER,
		api,
		type UserChange,
		type UserIdentity,
		type UserRecord
	} from '$lib/api';
	import Alert from '$lib/components/ui/Alert.svelte';
	import Section from '$lib/components/ui/Section.svelte';
	import LocalAccount from '$lib/components/users/LocalAccount.svelte';
	import NameForm from '$lib/components/users/NameForm.svelte';
	import PluginAllowlist from '$lib/components/users/PluginAllowlist.svelte';
	import UserAccess from '$lib/components/users/UserAccess.svelte';
	import UserBadges from '$lib/components/users/UserBadges.svelte';
	import { loadUser, userState } from '$lib/state/user.svelte';
	import { userFailureMessage } from '$lib/users/failure';
	import { personName } from '$lib/users/name';

	const userId = $derived(page.params.user ?? '');
	const isSelf = $derived(userState.user?.id === userId);

	let status = $state<'loading' | 'ready' | 'missing' | 'error'>('loading');
	let user = $state<UserRecord | null>(null);
	let allowed = $state<string[]>([]);
	let identities = $state<UserIdentity[]>([]);
	let localProvider = $state(false);
	let busy = $state(false);
	let banner = $state<string | null>(null);

	const name = $derived(user ? personName(user) : '');

	async function load(): Promise<void> {
		status = 'loading';

		try {
			const [record, list, held, providers] = await Promise.all([
				api.users.get(userId),
				api.users.allowedPlugins(userId),
				api.users.identities(userId),
				api.providers.list()
			]);
			if (record === null || list === null || held === null || providers === null) {
				return;
			}

			user = record;
			allowed = list.map((one) => one.plugin_id);
			identities = held;
			localProvider = providers.some((one) => one.id === LOCAL_PROVIDER);
			status = 'ready';
		} catch (error) {
			if (error instanceof ApiError && error.status === 404) {
				status = 'missing';

				return;
			}

			console.error('Failed to load the user', error);
			status = 'error';
		}
	}

	/** Sends the change, and answers whether the hub accepted it. */
	async function change(update: UserChange, fallback: string): Promise<boolean> {
		busy = true;
		banner = null;

		try {
			const changed = await api.users.change(userId, update);
			if (changed !== null) {
				user = changed;
			}

			if (isSelf) {
				await loadUser();
			}

			return true;
		} catch (error) {
			banner = userFailureMessage(error, fallback);

			return false;
		} finally {
			busy = false;
		}
	}

	async function saveNames(names: { first_name: string; last_name: string }): Promise<void> {
		await change(names, 'The names stay as they were. Try again.');
	}

	$effect(() => {
		void userId;
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
				<p class="page-lead">No user has this address.</p>
			</div>
		</header>
	{:else if status === 'error'}
		<Alert kind="error"><span>The hub answered no user. Reload the page.</span></Alert>
	{:else if user}
		<header class="page-header">
			<div>
				<h1 class="page-title">{name}</h1>
				<div class="mt-2 flex flex-wrap items-center gap-2">
					<UserBadges {user} {isSelf} showActive />
				</div>
			</div>
		</header>

		{#if banner}
			<Alert kind="error" class="mb-6"><span>{banner}</span></Alert>
		{/if}

		<Section title="Name">
			<NameForm record={user} disabled={busy} onsave={saveNames} />
		</Section>

		<Section title="Access">
			<UserAccess {user} {name} disabled={busy} onchange={change} />
		</Section>

		<Section title="Local account">
			<LocalAccount {userId} {name} bind:identities {localProvider} />
		</Section>

		<Section title="Plugins">
			<PluginAllowlist {user} {name} {isSelf} bind:allowed />
		</Section>
	{/if}
</div>
