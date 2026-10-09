<script lang="ts">
	import {
		api,
		createWriteIntent,
		type Invitation,
		type InvitedUser,
		type UserChange,
		type UserRecord
	} from '$lib/api';
	import Alert from '$lib/components/ui/Alert.svelte';
	import Section from '$lib/components/ui/Section.svelte';
	import SkeletonList from '$lib/components/ui/SkeletonList.svelte';
	import CreateUserForm from '$lib/components/users/CreateUserForm.svelte';
	import InvitationNotice from '$lib/components/users/InvitationNotice.svelte';
	import UserRow from '$lib/components/users/UserRow.svelte';
	import { loadUser, userState } from '$lib/state/user.svelte';
	import { userFailureMessage } from '$lib/users/failure';
	import { personName } from '$lib/users/name';

	const inviteIntent = createWriteIntent();

	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let users = $state<UserRecord[]>([]);
	let busy = $state(false);
	let banner = $state<string | null>(null);
	let invitation = $state<{ name: string; invitation: Invitation } | null>(null);

	const selfId = $derived(userState.user?.id ?? '');

	async function load(): Promise<void> {
		status = 'loading';

		try {
			const list = await api.users.list();
			if (list === null) {
				return;
			}

			users = list;
			status = 'ready';
		} catch (error) {
			console.error('Failed to load the users', error);
			status = 'error';
		}
	}

	async function created(result: InvitedUser): Promise<void> {
		invitation = { name: personName(result.user), invitation: result.invitation };
		await load();
	}

	async function invite(user: UserRecord): Promise<void> {
		busy = true;
		banner = null;

		try {
			const issued = await api.users.invite(user.id, inviteIntent.keyFor({ user: user.id }));
			if (issued === null) {
				return;
			}

			inviteIntent.settle();
			invitation = { name: personName(user), invitation: issued };
		} catch (error) {
			banner = userFailureMessage(error, 'The hub issued no invitation. Try again.');
		} finally {
			busy = false;
		}
	}

	async function change(user: UserRecord, update: UserChange): Promise<void> {
		busy = true;
		banner = null;

		try {
			await api.users.change(user.id, update);

			if (user.id === selfId) {
				await loadUser();
			}
		} catch (error) {
			banner = userFailureMessage(error, 'The user stays as they were. Try again.');
		} finally {
			busy = false;
			await load();
		}
	}

	$effect(() => {
		void load();
	});
</script>

<div class="max-w-3xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">Users</h1>
			<p class="page-lead">
				Every person of this instance. A block keeps the records of the person.
			</p>
		</div>
	</header>

	{#if banner}
		<Alert kind="error" class="mb-6"><span>{banner}</span></Alert>
	{/if}

	{#if invitation}
		<InvitationNotice
			name={invitation.name}
			invitation={invitation.invitation}
			onclose={() => (invitation = null)}
		/>
	{/if}

	<Section title="Create a user">
		<CreateUserForm oncreated={created} />
	</Section>

	<Section title="All users">
		{#if status === 'loading'}
			<SkeletonList />
		{:else if status === 'error'}
			<Alert kind="error" class="mb-6">
				<span>The hub answered no user. Reload the page.</span>
			</Alert>
		{:else}
			<ul class="list panel">
				{#each users as user (user.id)}
					<UserRow
						{user}
						isSelf={user.id === selfId}
						disabled={busy}
						oninvite={() => invite(user)}
						onchange={(update) => change(user, update)}
					/>
				{/each}
			</ul>
		{/if}
	</Section>
</div>
