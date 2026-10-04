<script lang="ts">
	import { resolve } from '$app/paths';

	import {
		ApiError,
		PROBLEM_TYPE,
		api,
		createWriteIntent,
		isProblem,
		type Invitation,
		type UserChange,
		type UserRecord
	} from '$lib/api';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import { displayNameOf } from '$lib/plugins/capabilities';
	import { pluginState } from '$lib/state/plugins.svelte';
	import { loadUser, userState } from '$lib/state/user.svelte';

	const createIntent = createWriteIntent();
	const inviteIntent = createWriteIntent();

	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let users = $state<UserRecord[]>([]);
	let busy = $state(false);
	let banner = $state<string | null>(null);
	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();

	let firstName = $state('');
	let lastName = $state('');
	let administrator = $state(false);
	let allowedPlugins = $state<string[]>([]);

	const allowlistLabel = $derived(
		allowedPlugins.length === 0
			? 'No plugin'
			: pluginState.plugins
					.filter((plugin) => allowedPlugins.includes(plugin.id))
					.map((plugin) => displayNameOf(plugin, plugin.id))
					.join(', ')
	);

	let invitation = $state<{ name: string; invitation: Invitation } | null>(null);
	let copied = $state(false);

	const selfId = $derived(userState.user?.id ?? '');

	function nameOf(user: { first_name?: string; last_name?: string }): string {
		return [user.first_name, user.last_name].filter(Boolean).join(' ') || 'Unnamed person';
	}

	function failureMessage(error: unknown, fallback: string): string {
		if (error instanceof ApiError && isProblem(error.body)) {
			if (error.body.type === PROBLEM_TYPE.administratorLast) {
				return 'The instance keeps one active administrator. Mark another user first.';
			}

			return error.body.errors?.[0]?.detail ?? error.body.title;
		}

		return fallback;
	}

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

	async function create(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		if (busy) {
			return;
		}

		const input = {
			first_name: firstName.trim() || undefined,
			last_name: lastName.trim() || undefined,
			is_administrator: administrator,
			allowed_plugins: administrator ? [] : allowedPlugins
		};

		busy = true;
		banner = null;

		try {
			const created = await api.users.create(input, createIntent.keyFor(input));
			if (created === null) {
				return;
			}

			createIntent.settle();
			invitation = { name: nameOf(created.user), invitation: created.invitation };
			copied = false;
			firstName = '';
			lastName = '';
			administrator = false;
			allowedPlugins = [];
			await load();
		} catch (error) {
			banner = failureMessage(error, 'The hub created nobody. Try again.');
		} finally {
			busy = false;
		}
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
			invitation = { name: nameOf(user), invitation: issued };
			copied = false;
		} catch (error) {
			banner = failureMessage(error, 'The hub issued no invitation. Try again.');
		} finally {
			busy = false;
		}
	}

	async function change(user: UserRecord, update: UserChange, question?: string): Promise<void> {
		if (question !== undefined) {
			const proceed = await confirmDialog?.ask(question, {
				title: update.status === 'blocked' ? 'Block this user?' : 'Unmark this administrator?',
				confirmLabel: update.status === 'blocked' ? 'Block' : 'Unmark'
			});

			if (!proceed) {
				return;
			}
		}

		busy = true;
		banner = null;

		try {
			await api.users.change(user.id, update);

			if (user.id === selfId) {
				await loadUser();
			}
		} catch (error) {
			banner = failureMessage(error, 'The user stays as they were. Try again.');
		} finally {
			busy = false;
			await load();
		}
	}

	async function copy(address: string): Promise<void> {
		try {
			await navigator.clipboard.writeText(address);
			copied = true;
		} catch (error) {
			console.error('Failed to copy the invitation', error);
		}
	}

	$effect(() => {
		void load();
	});
</script>

<div class="max-w-3xl">
	<h1 class="font-display text-[42px] leading-[1.05] tracking-tight">Users</h1>
	<p class="text-base-content/60 mt-3 text-sm">
		Every person of this instance. A block keeps the records of the person.
	</p>

	{#if banner}
		<div class="alert alert-error mt-6">
			<span>{banner}</span>
		</div>
	{/if}

	{#if invitation}
		<div class="alert alert-info mt-6 flex flex-col items-start gap-2">
			<span class="text-sm">
				The invitation of {invitation.name}. It shows this one time, and it expires on
				{new Date(invitation.invitation.expires_at).toLocaleString()}.
			</span>
			<div class="flex w-full gap-2">
				<input
					class="input input-bordered input-sm w-full font-mono text-[11px]"
					readonly
					value={invitation.invitation.address}
					aria-label="The address of the invitation"
				/>
				<button
					type="button"
					class="btn btn-sm"
					onclick={() => invitation && copy(invitation.invitation.address)}
				>
					{copied ? 'Copied' : 'Copy'}
				</button>
				<button type="button" class="btn btn-ghost btn-sm" onclick={() => (invitation = null)}>
					Close
				</button>
			</div>
		</div>
	{/if}

	<form class="mt-8" onsubmit={create}>
		<fieldset class="fieldset bg-base-200/40 border-base-300 rounded-box w-full min-w-0 border p-4">
			<legend class="fieldset-legend">Create a user</legend>

			<div class="grid gap-3 sm:grid-cols-2">
				<label class="flex flex-col gap-1">
					<span class="label">First name</span>
					<input class="input input-sm w-full" bind:value={firstName} />
				</label>
				<label class="flex flex-col gap-1">
					<span class="label">Last name</span>
					<input class="input input-sm w-full" bind:value={lastName} />
				</label>
			</div>

			<div class="mt-2 grid gap-3 sm:grid-cols-2">
				{#if pluginState.plugins.length > 0}
					<div class="flex flex-col gap-1">
						<span class="label">Plugins</span>
						{#if administrator}
							<button
								type="button"
								class="btn btn-sm btn-outline w-full justify-start font-normal"
								disabled
							>
								Every plugin
							</button>
						{:else}
							<div class="dropdown w-full">
								<div
									tabindex="0"
									role="button"
									class="btn btn-sm btn-outline w-full justify-between font-normal"
									aria-label="The plugins that this person can turn on"
								>
									<span class="truncate">{allowlistLabel}</span>
									<svg
										xmlns="http://www.w3.org/2000/svg"
										width="12"
										height="12"
										viewBox="0 0 24 24"
										fill="none"
										stroke="currentColor"
										stroke-width="2"
										opacity="0.5"
									>
										<path d="m6 9 6 6 6-6" />
									</svg>
								</div>
								<ul
									tabindex="-1"
									class="dropdown-content menu menu-sm bg-base-100 border-base-300 rounded-box z-50 mt-2 w-full border p-2 shadow-lg"
								>
									{#each pluginState.plugins as plugin (plugin.id)}
										<li>
											<label class="flex items-center gap-2">
												<input
													type="checkbox"
													class="checkbox checkbox-sm"
													value={plugin.id}
													bind:group={allowedPlugins}
												/>
												<span class="truncate">{displayNameOf(plugin, plugin.id)}</span>
											</label>
										</li>
									{/each}
								</ul>
							</div>
						{/if}
						<span class="label whitespace-normal">
							{administrator
								? 'An administrator already reaches every plugin.'
								: 'The plugins that this person can turn on in a workspace.'}
						</span>
					</div>
				{/if}

				<label class="flex cursor-pointer flex-wrap items-center gap-2 sm:pt-6">
					<input type="checkbox" class="toggle toggle-sm" bind:checked={administrator} />
					<span class="text-sm">Administrator</span>
					<span class="label whitespace-normal"
						>Manages the users and the plugins of every workspace.</span
					>
				</label>
			</div>

			<div class="mt-3 flex justify-end">
				<button type="submit" class="btn btn-primary btn-sm" disabled={busy}>Create</button>
			</div>
		</fieldset>
	</form>

	{#if status === 'loading'}
		<div class="mt-8 flex flex-col gap-2" aria-hidden="true">
			{#each [0, 1, 2] as i (i)}
				<span class="skeleton h-12 w-full"></span>
			{/each}
		</div>
	{:else if status === 'error'}
		<div class="alert alert-error mt-6">
			<span>The hub answered no user. Reload the page.</span>
		</div>
	{:else}
		<ul class="border-base-300 rounded-box mt-8 divide-y border">
			{#each users as user (user.id)}
				<li class="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
					<span class="flex items-center gap-2 text-sm">
						<a
							class="link link-hover font-semibold"
							href={resolve('/(dashboard)/admin/users/[user]', { user: user.id })}
						>
							{nameOf(user)}
						</a>
						{#if user.id === selfId}
							<span class="badge badge-ghost badge-sm">You</span>
						{/if}
						{#if user.is_administrator}
							<span class="badge badge-outline badge-sm">Administrator</span>
						{/if}
						{#if user.status === 'blocked'}
							<span class="badge badge-error badge-sm">Blocked</span>
						{/if}
					</span>
					<span class="flex items-center gap-1">
						<button
							type="button"
							class="btn btn-ghost btn-xs"
							disabled={busy}
							onclick={() => invite(user)}
						>
							Invite
						</button>
						{#if user.is_administrator}
							<button
								type="button"
								class="btn btn-ghost btn-xs"
								disabled={busy}
								onclick={() =>
									change(
										user,
										{ is_administrator: false },
										`${nameOf(user)} will no longer administer this instance.`
									)}
							>
								Unmark
							</button>
						{:else}
							<button
								type="button"
								class="btn btn-ghost btn-xs"
								disabled={busy}
								onclick={() => change(user, { is_administrator: true })}
							>
								Mark administrator
							</button>
						{/if}
						{#if user.status === 'blocked'}
							<button
								type="button"
								class="btn btn-ghost btn-xs"
								disabled={busy}
								onclick={() => change(user, { status: 'active' })}
							>
								Unblock
							</button>
						{:else}
							<button
								type="button"
								class="btn btn-ghost btn-xs text-error"
								disabled={busy}
								onclick={() =>
									change(
										user,
										{ status: 'blocked' },
										`${nameOf(user)} will reach nothing until you unblock them. Their records stay.`
									)}
							>
								Block
							</button>
						{/if}
					</span>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<ConfirmDialog bind:this={confirmDialog} />
