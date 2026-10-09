<script lang="ts">
	import { onMount } from 'svelte';

	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';

	import {
		ApiError,
		LOCAL_PROVIDER,
		PROBLEM_TYPE,
		api,
		isProblem,
		type ProviderIdentity,
		type Tagged,
		type UserRecord
	} from '$lib/api';
	import { startAttach } from '$lib/api/client';
	import { authFailureMessage } from '$lib/auth/messages';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import { fieldErrors, passwordProblem } from '$lib/providers/form';
	import { loadUser, userState } from '$lib/state/user.svelte';

	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let identities = $state<ProviderIdentity[]>([]);
	let banner = $state<string | null>(null);
	let detaching = $state<string | null>(null);
	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();

	let record = $state<UserRecord | null>(null);
	let etag = $state<string | undefined>(undefined);
	let firstName = $state('');
	let lastName = $state('');
	let savingNames = $state(false);
	let namesNotice = $state<string | null>(null);

	let currentPassword = $state('');
	let newPassword = $state('');
	let repeatedPassword = $state('');
	let changingPassword = $state(false);
	let passwordNotice = $state<string | null>(null);
	let passwordFailure = $state<string | null>(null);
	let currentPasswordFailure = $state<string | null>(null);

	const currentProvider = $derived(userState.user?.provider ?? null);
	const localAccount = $derived(
		identities.find((one) => one.provider === LOCAL_PROVIDER && one.attached)
	);
	const namesChanged = $derived(
		record !== null &&
			(firstName.trim() !== (record.first_name ?? '') ||
				lastName.trim() !== (record.last_name ?? ''))
	);
	const newPasswordProblem = $derived(
		newPassword === '' ? null : passwordProblem(newPassword, repeatedPassword)
	);
	const canChangePassword = $derived(
		currentPassword !== '' && newPassword !== '' && newPasswordProblem === null && !changingPassword
	);

	function showRecord(tagged: Tagged<UserRecord>): void {
		record = tagged.value;
		etag = tagged.etag;
		firstName = record.first_name ?? '';
		lastName = record.last_name ?? '';
	}

	async function loadRecord(): Promise<void> {
		try {
			const tagged = await api.users.self();
			if (tagged !== null) {
				showRecord(tagged);
			}
		} catch (error) {
			console.error('Failed to load the user record', error);
			banner = 'Failed to load your name. Reload the page.';
		}
	}

	function nameFailureMessage(error: unknown): string {
		if (
			error instanceof ApiError &&
			isProblem(error.body) &&
			error.body.type === PROBLEM_TYPE.preconditionFailed
		) {
			return 'Your name changed in another tab. Reload the page.';
		}

		return 'Your name stays as it was. Try again.';
	}

	async function saveNames(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		savingNames = true;
		banner = null;
		namesNotice = null;

		try {
			const tagged = await api.users.changeSelf(
				{ first_name: firstName.trim(), last_name: lastName.trim() },
				etag
			);
			if (tagged !== null) {
				showRecord(tagged);
			}

			await loadUser();
			namesNotice = 'Your name is saved.';
		} catch (error) {
			banner = nameFailureMessage(error);
		} finally {
			savingNames = false;
		}
	}

	function passwordFailureMessage(error: unknown): string {
		if (error instanceof ApiError && isProblem(error.body)) {
			if (error.body.type === PROBLEM_TYPE.notReady) {
				return 'The sign-in service does not answer. Try again in a moment.';
			}

			const detail = fieldErrors(error)['/password'];
			if (detail) {
				return detail;
			}
		}

		return 'Your password stays as it was. Try again.';
	}

	async function changePassword(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		changingPassword = true;
		passwordNotice = null;
		passwordFailure = null;
		currentPasswordFailure = null;

		try {
			await api.identities.changePassword(currentPassword, newPassword);

			currentPassword = '';
			newPassword = '';
			repeatedPassword = '';
			passwordNotice = 'The new password works from the next sign in.';
		} catch (error) {
			const current = fieldErrors(error)['/current_password'];
			if (current) {
				currentPasswordFailure = 'This is not your current password.';
			} else {
				passwordFailure = passwordFailureMessage(error);
			}
		} finally {
			changingPassword = false;
		}
	}

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

	function detachFailureMessage(error: unknown): string {
		if (error instanceof ApiError && typeof error.body === 'object' && error.body !== null) {
			const reason = (error.body as { error?: unknown }).error;
			if (typeof reason === 'string') {
				return reason;
			}
		}

		return 'The hub kept the account connected. Try again.';
	}

	async function detach(provider: string): Promise<void> {
		if (provider === currentProvider) {
			const proceed = await confirmDialog?.ask(
				"This is the account you're signed in with. Disconnecting it will sign you out.",
				{ title: 'Disconnect this account?', confirmLabel: 'Disconnect' }
			);

			if (!proceed) {
				return;
			}
		}

		detaching = provider;
		banner = null;

		try {
			await api.identities.detach(provider);
			await load();
		} catch (error) {
			banner = detachFailureMessage(error);
		} finally {
			detaching = null;
		}
	}

	function connect(provider: string): void {
		void startAttach(provider).then((address) => {
			window.location.href = address;
		});
	}

	onMount(() => {
		const reason = page.url.searchParams.get('error');

		if (reason) {
			banner = authFailureMessage(reason);
			void goto(resolve('/profile'), { replaceState: true, keepFocus: true, noScroll: true });
		}

		void load();
		void loadRecord();
	});
</script>

<div class="max-w-2xl">
	<h1 class="font-display text-[42px] leading-[1.05] tracking-tight">Profile</h1>
	<p class="text-base-content/60 mt-3 text-sm">
		Your name, your password, and the accounts that reach this record.
	</p>

	{#if banner}
		<div class="alert alert-error mt-6">
			<span>{banner}</span>
		</div>
	{/if}

	<form class="mt-8" onsubmit={saveNames}>
		<fieldset class="fieldset bg-base-200/40 border-base-300 rounded-box w-full min-w-0 border p-4">
			<legend class="fieldset-legend">Profile</legend>

			{#if namesNotice}
				<div role="status" class="alert alert-success alert-soft">
					<span class="text-sm">{namesNotice}</span>
				</div>
			{/if}

			<div class="grid gap-3 sm:grid-cols-2">
				<label class="flex flex-col gap-1">
					<span class="label">First name</span>
					<input
						class="input input-sm w-full"
						autocomplete="given-name"
						bind:value={firstName}
						disabled={record === null || savingNames}
					/>
				</label>
				<label class="flex flex-col gap-1">
					<span class="label">Last name</span>
					<input
						class="input input-sm w-full"
						autocomplete="family-name"
						bind:value={lastName}
						disabled={record === null || savingNames}
					/>
				</label>
			</div>

			<div class="mt-3 flex justify-end">
				<button
					type="submit"
					class="btn btn-primary btn-sm"
					disabled={!namesChanged || savingNames}
				>
					Save
				</button>
			</div>
		</fieldset>
	</form>

	{#if localAccount}
		<form class="mt-6" onsubmit={changePassword}>
			<fieldset
				class="fieldset bg-base-200/40 border-base-300 rounded-box w-full min-w-0 border p-4"
			>
				<legend class="fieldset-legend">Password</legend>

				{#if passwordNotice}
					<div role="status" class="alert alert-success alert-soft">
						<span class="text-sm">{passwordNotice}</span>
					</div>
				{/if}
				{#if passwordFailure}
					<div role="alert" class="alert alert-error alert-soft">
						<span class="text-sm">{passwordFailure}</span>
					</div>
				{/if}

				<span class="label whitespace-normal">
					You sign in with <span class="font-mono">{localAccount.username}</span> and a password.
				</span>

				<div class="mt-1 grid gap-3 sm:grid-cols-2">
					<label class="flex flex-col gap-1 sm:col-span-2">
						<span class="label">Current password</span>
						<input
							class="input input-sm w-full"
							class:input-error={currentPasswordFailure !== null}
							type="password"
							autocomplete="current-password"
							bind:value={currentPassword}
							disabled={changingPassword}
							required
						/>
						{#if currentPasswordFailure}
							<span class="label text-error">{currentPasswordFailure}</span>
						{/if}
					</label>
					<label class="flex flex-col gap-1">
						<span class="label">New password</span>
						<input
							class="input input-sm w-full"
							type="password"
							autocomplete="new-password"
							bind:value={newPassword}
							disabled={changingPassword}
							required
						/>
					</label>
					<label class="flex flex-col gap-1">
						<span class="label">Repeat the password</span>
						<input
							class="input input-sm w-full"
							type="password"
							autocomplete="new-password"
							bind:value={repeatedPassword}
							disabled={changingPassword}
							required
						/>
					</label>
					{#if newPasswordProblem}
						<span class="label whitespace-normal sm:col-span-2">{newPasswordProblem}</span>
					{/if}
					<div class="flex justify-end sm:col-span-2">
						<button type="submit" class="btn btn-primary btn-sm" disabled={!canChangePassword}>
							Set the password
						</button>
					</div>
				</div>
			</fieldset>
		</form>
	{/if}

	<fieldset
		class="fieldset bg-base-200/40 border-base-300 rounded-box mt-6 w-full min-w-0 border p-4"
	>
		<legend class="fieldset-legend">Connected accounts</legend>

		{#if status === 'loading'}
			<div class="flex flex-col gap-2">
				{#each [0, 1] as i (i)}
					<span class="skeleton h-16 w-full" aria-hidden="true"></span>
				{/each}
			</div>
		{:else if status === 'error'}
			<div role="alert" class="alert alert-error alert-soft">
				<span class="text-sm">Failed to load the connected accounts.</span>
				<button type="button" class="btn btn-sm" onclick={() => void load()}>Retry</button>
			</div>
		{:else}
			<ul class="border-base-300 bg-base-100 rounded-box mt-1 divide-y border">
				{#each identities as identity (identity.provider)}
					<li class="flex items-center justify-between gap-3 px-4 py-3">
						<span class="min-w-0">
							<span class="flex items-center gap-2">
								<span class="text-sm font-semibold">{identity.name}</span>
								{#if identity.provider === currentProvider}
									<span class="badge badge-ghost badge-xs">Current session</span>
								{/if}
							</span>
							{#if identity.attached}
								<span class="text-base-content/50 block truncate font-mono text-[11px]">
									{identity.username ?? identity.display_name ?? 'Connected'}
								</span>
							{:else}
								<span class="text-base-content/35 block font-mono text-[11px]">Not connected</span>
							{/if}
						</span>

						{#if identity.attached}
							<button
								type="button"
								class="btn btn-ghost btn-sm text-error"
								disabled={detaching === identity.provider}
								onclick={() => void detach(identity.provider)}
							>
								Disconnect
							</button>
						{:else if identity.provider === LOCAL_PROVIDER}
							<span class="label whitespace-normal">An administrator gives this account</span>
						{:else}
							<button type="button" class="btn btn-sm" onclick={() => connect(identity.provider)}>
								Connect
							</button>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}
	</fieldset>
</div>

<ConfirmDialog bind:this={confirmDialog} />
