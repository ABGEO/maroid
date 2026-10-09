<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';

	import {
		ApiError,
		LOCAL_PROVIDER,
		PROBLEM_TYPE,
		api,
		createWriteIntent,
		isProblem,
		type UserChange,
		type UserIdentity,
		type UserRecord
	} from '$lib/api';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import { failureMessage as accountFailure, passwordProblem } from '$lib/providers/form';
	import { displayNameOf } from '$lib/plugins/capabilities';
	import { pluginState } from '$lib/state/plugins.svelte';
	import { loadUser, userState } from '$lib/state/user.svelte';

	const intent = createWriteIntent();
	const userId = $derived(page.params.user ?? '');
	const isSelf = $derived(userState.user?.id === userId);

	let status = $state<'loading' | 'ready' | 'missing' | 'error'>('loading');
	let user = $state<UserRecord | null>(null);
	let allowed = $state<string[]>([]);
	let busy = $state(false);
	let banner = $state<string | null>(null);
	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();

	let firstName = $state('');
	let lastName = $state('');

	let identities = $state<UserIdentity[]>([]);
	let localProvider = $state(false);
	let email = $state('');
	let password = $state('');
	let repeated = $state('');
	let accountNotice = $state<string | null>(null);

	const localAccount = $derived(identities.find((one) => one.provider === LOCAL_PROVIDER));

	const name = $derived(
		[user?.first_name, user?.last_name].filter(Boolean).join(' ') || 'Unnamed person'
	);
	const namesChanged = $derived(
		firstName.trim() !== (user?.first_name ?? '') || lastName.trim() !== (user?.last_name ?? '')
	);

	function failureMessage(error: unknown, fallback: string): string {
		if (error instanceof ApiError && isProblem(error.body)) {
			if (error.body.type === PROBLEM_TYPE.administratorLast) {
				return 'The instance keeps one active administrator. Mark another user first.';
			}

			if (error.body.type === PROBLEM_TYPE.administratorAllowlist) {
				return 'An administrator reaches every plugin and holds no allowlist.';
			}

			return error.body.errors?.[0]?.detail ?? error.body.title;
		}

		return fallback;
	}

	function show(record: UserRecord): void {
		user = record;
		firstName = record.first_name ?? '';
		lastName = record.last_name ?? '';
	}

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

			show(record);
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
				show(changed);
			}

			if (isSelf) {
				await loadUser();
			}

			return true;
		} catch (error) {
			banner = failureMessage(error, fallback);

			return false;
		} finally {
			busy = false;
		}
	}

	async function saveNames(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		await change(
			{ first_name: firstName.trim(), last_name: lastName.trim() },
			'The names stay as they were. Try again.'
		);
	}

	// A switch flips before the person confirms, so a cancel or a refusal flips it back.
	async function setAdministrator(input: HTMLInputElement): Promise<void> {
		const on = input.checked;
		const confirmed =
			on ||
			(await confirmDialog?.ask(`${name} will no longer administer this instance.`, {
				title: 'Unmark this administrator?',
				confirmLabel: 'Unmark'
			}));

		if (!confirmed || !(await change({ is_administrator: on }, 'The mark stays as it was.'))) {
			input.checked = !on;
		}
	}

	async function setBlocked(input: HTMLInputElement): Promise<void> {
		const blocked = input.checked;
		const confirmed =
			!blocked ||
			(await confirmDialog?.ask(
				`${name} will reach nothing until you unblock them. Their records stay.`,
				{ title: 'Block this user?', confirmLabel: 'Block' }
			));
		const accepted =
			confirmed &&
			(await change(
				{ status: blocked ? 'blocked' : 'active' },
				'The user stays as they were. Try again.'
			));

		if (!accepted) {
			input.checked = !blocked;
		}
	}

	async function togglePlugin(pluginId: string, input: HTMLInputElement): Promise<void> {
		const on = input.checked;

		busy = true;
		banner = null;

		try {
			if (on) {
				await api.users.allowPlugin(userId, pluginId, intent.keyFor({ plugin_id: pluginId }));
				intent.settle();
				allowed = [...allowed, pluginId];
			} else {
				await api.users.disallowPlugin(userId, pluginId);
				allowed = allowed.filter((id) => id !== pluginId);
			}
		} catch (error) {
			banner = failureMessage(error, 'The allowlist stays as it was. Try again.');
			input.checked = !on;
		} finally {
			busy = false;
		}
	}

	/** Runs one write of the local account, and reads the identities after it. */
	async function account(
		write: () => Promise<unknown>,
		done: string,
		fallback: string
	): Promise<void> {
		busy = true;
		banner = null;
		accountNotice = null;

		try {
			await write();

			const held = await api.users.identities(userId);
			if (held !== null) {
				identities = held;
			}

			password = '';
			repeated = '';
			accountNotice = done;
		} catch (error) {
			banner = accountFailure(error, fallback);
		} finally {
			busy = false;
		}
	}

	async function giveAccount(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		const problem = passwordProblem(password, repeated);
		if (problem) {
			banner = problem;

			return;
		}

		const address = email.trim();

		await account(
			async () => {
				await api.users.giveLocalAccount(
					userId,
					address,
					password,
					intent.keyFor({ provider: LOCAL_PROVIDER, email: address })
				);
				intent.settle();
				email = '';
			},
			`${name} signs in with ${address} and this password. Hand both over in person.`,
			'The user holds no local account. Try again.'
		);
	}

	async function resetPassword(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		const problem = passwordProblem(password, repeated);
		if (problem) {
			banner = problem;

			return;
		}

		await account(
			() => api.users.resetLocalPassword(userId, password),
			'The new password works from the next sign in. The old one no longer does.',
			'The password stays as it was. Try again.'
		);
	}

	async function removeAccount(): Promise<void> {
		const confirmed = await confirmDialog?.ask(
			`${name} will no longer sign in with ${localAccount?.username ?? 'the local account'}.`,
			{ title: 'Remove the local account?', confirmLabel: 'Remove' }
		);
		if (!confirmed) {
			return;
		}

		await account(
			() => api.users.removeLocalAccount(userId),
			'The local account is gone.',
			'The local account stays. Try again.'
		);
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
		<div role="alert" class="alert alert-error alert-soft">
			<span>The hub answered no user. Reload the page.</span>
		</div>
	{:else if user}
		<header class="page-header">
			<div>
				<h1 class="page-title">{name}</h1>
				<div class="mt-2 flex flex-wrap items-center gap-2">
					{#if isSelf}
						<span class="badge badge-ghost badge-sm">You</span>
					{/if}
					{#if user.is_administrator}
						<span class="badge badge-outline badge-sm">Administrator</span>
					{/if}
					{#if user.status === 'blocked'}
						<span class="badge badge-error badge-sm">Blocked</span>
					{:else}
						<span class="badge badge-success badge-soft badge-sm">Active</span>
					{/if}
				</div>
			</div>
		</header>

		{#if banner}
			<div role="alert" class="alert alert-error alert-soft mb-6">
				<span>{banner}</span>
			</div>
		{/if}

		<section class="section">
			<h2 class="section-title">Name</h2>
			<form onsubmit={saveNames}>
				<fieldset class="fieldset panel w-full min-w-0 p-5">
					<div class="grid gap-3 sm:grid-cols-2">
						<label class="flex flex-col gap-1">
							<span class="label">First name</span>
							<input class="input input-sm w-full" bind:value={firstName} disabled={busy} />
						</label>
						<label class="flex flex-col gap-1">
							<span class="label">Last name</span>
							<input class="input input-sm w-full" bind:value={lastName} disabled={busy} />
						</label>
					</div>

					<div class="mt-3 flex justify-end">
						<button type="submit" class="btn btn-primary btn-sm" disabled={busy || !namesChanged}>
							Save
						</button>
					</div>
				</fieldset>
			</form>
		</section>

		<section class="section">
			<h2 class="section-title">Access</h2>
			<fieldset class="fieldset panel w-full min-w-0 p-5">
				<label class="flex cursor-pointer flex-wrap items-center gap-2">
					<input
						type="checkbox"
						class="toggle toggle-sm"
						checked={user.is_administrator}
						disabled={busy}
						onchange={(event) => setAdministrator(event.currentTarget)}
					/>
					<span class="text-sm">Administrator</span>
					<span class="label whitespace-normal"
						>Manages the users and the plugins of every workspace.</span
					>
				</label>

				<label class="mt-2 flex cursor-pointer flex-wrap items-center gap-2">
					<input
						type="checkbox"
						class="toggle toggle-sm toggle-error"
						checked={user.status === 'blocked'}
						disabled={busy}
						onchange={(event) => setBlocked(event.currentTarget)}
					/>
					<span class="text-sm">Blocked</span>
					<span class="label whitespace-normal"
						>Reaches nothing until you unblock them. Their records stay.</span
					>
				</label>
			</fieldset>
		</section>

		<section class="section">
			<h2 class="section-title">Local account</h2>
			<fieldset class="fieldset panel w-full min-w-0 p-5">
				{#if accountNotice}
					<div role="status" class="alert alert-success alert-soft">
						<span class="text-sm">{accountNotice}</span>
					</div>
				{/if}

				{#if localAccount}
					<span class="label whitespace-normal">
						{name} signs in with <span class="font-mono">{localAccount.username}</span> and a password.
					</span>

					<form class="mt-1 grid gap-3 sm:grid-cols-2" onsubmit={resetPassword}>
						<label class="flex flex-col gap-1">
							<span class="label">New password</span>
							<input
								class="input input-sm w-full"
								type="password"
								autocomplete="new-password"
								bind:value={password}
								disabled={busy}
								required
							/>
						</label>
						<label class="flex flex-col gap-1">
							<span class="label">Repeat the password</span>
							<input
								class="input input-sm w-full"
								type="password"
								autocomplete="new-password"
								bind:value={repeated}
								disabled={busy}
								required
							/>
						</label>
						<div class="flex justify-end gap-2 sm:col-span-2">
							<button
								type="button"
								class="btn btn-ghost btn-sm text-error"
								disabled={busy || identities.length < 2}
								title={identities.length < 2 ? 'It is the only way that this user signs in.' : ''}
								onclick={removeAccount}
							>
								Remove the account
							</button>
							<button type="submit" class="btn btn-primary btn-sm" disabled={busy}>
								Set the password
							</button>
						</div>
					</form>
				{:else if !localProvider}
					<span class="label whitespace-normal">
						Add the Email provider on the
						<a class="link" href={resolve('/admin/providers')}>sign-in providers</a> page first.
					</span>
				{:else}
					<span class="label whitespace-normal">
						{name} holds no local account. Give one, then hand the address and the password over in person.
					</span>

					<form class="mt-1 grid gap-3 sm:grid-cols-2" onsubmit={giveAccount}>
						<label class="flex flex-col gap-1 sm:col-span-2">
							<span class="label">Email address</span>
							<input
								class="input input-sm w-full"
								type="email"
								autocomplete="off"
								bind:value={email}
								disabled={busy}
								required
							/>
						</label>
						<label class="flex flex-col gap-1">
							<span class="label">Password</span>
							<input
								class="input input-sm w-full"
								type="password"
								autocomplete="new-password"
								bind:value={password}
								disabled={busy}
								required
							/>
						</label>
						<label class="flex flex-col gap-1">
							<span class="label">Repeat the password</span>
							<input
								class="input input-sm w-full"
								type="password"
								autocomplete="new-password"
								bind:value={repeated}
								disabled={busy}
								required
							/>
						</label>
						<div class="flex justify-end sm:col-span-2">
							<button type="submit" class="btn btn-primary btn-sm" disabled={busy}>
								Give the account
							</button>
						</div>
					</form>
				{/if}
			</fieldset>
		</section>

		<section class="section">
			<h2 class="section-title">Plugins</h2>
			<fieldset class="fieldset panel w-full min-w-0 p-5">
				{#if pluginState.plugins.length === 0}
					<span class="label whitespace-normal">The hub loaded no plugin.</span>
				{:else}
					{#if user.is_administrator}
						<div role="alert" class="alert alert-info alert-soft">
							<span class="text-sm">
								{isSelf ? 'You are' : `${name} is`} an administrator, so
								{isSelf ? 'you' : 'they'} already reach every plugin. The allowlist applies again when
								the mark goes.
							</span>
						</div>
					{:else}
						<span class="label whitespace-normal">
							The plugins that {name} can turn on in a workspace. A removal keeps every plugin that a
							workspace already uses.
						</span>
					{/if}
					<ul class="list panel">
						{#each pluginState.plugins as plugin (plugin.id)}
							<li class="list-row flex items-center justify-between gap-3">
								<span class="min-w-0">
									<span class="block font-medium">{displayNameOf(plugin, plugin.id)}</span>
									<span class="meta block truncate font-mono">
										{plugin.id}
									</span>
									{#if plugin.description}
										<span class="text-base-content/70 mt-1 block text-sm">{plugin.description}</span
										>
									{/if}
								</span>
								<input
									type="checkbox"
									class="toggle toggle-primary toggle-sm"
									aria-label="Allow {displayNameOf(plugin, plugin.id)} for {name}"
									checked={allowed.includes(plugin.id)}
									disabled={busy || user.is_administrator}
									onchange={(event) => togglePlugin(plugin.id, event.currentTarget)}
								/>
							</li>
						{/each}
					</ul>
				{/if}
			</fieldset>
		</section>
	{/if}
</div>

<ConfirmDialog bind:this={confirmDialog} />
