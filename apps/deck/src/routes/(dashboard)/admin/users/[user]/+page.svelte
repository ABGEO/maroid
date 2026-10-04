<script lang="ts">
	import { page } from '$app/state';

	import {
		ApiError,
		PROBLEM_TYPE,
		api,
		createWriteIntent,
		isProblem,
		type UserChange,
		type UserRecord
	} from '$lib/api';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
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
			const [record, list] = await Promise.all([
				api.users.get(userId),
				api.users.allowedPlugins(userId)
			]);
			if (record === null || list === null) {
				return;
			}

			show(record);
			allowed = list.map((one) => one.plugin_id);
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
		<h1 class="font-display text-[42px] leading-[1.05] tracking-tight">Not found</h1>
		<p class="text-base-content/60 mt-3 text-sm">No user has this address.</p>
	{:else if status === 'error'}
		<div class="alert alert-error">
			<span>The hub answered no user. Reload the page.</span>
		</div>
	{:else if user}
		<h1 class="font-display text-[42px] leading-[1.05] tracking-tight">{name}</h1>
		<div class="mt-3 flex flex-wrap items-center gap-2">
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

		{#if banner}
			<div class="alert alert-error mt-6">
				<span>{banner}</span>
			</div>
		{/if}

		<form class="mt-8" onsubmit={saveNames}>
			<fieldset
				class="fieldset bg-base-200/40 border-base-300 rounded-box w-full min-w-0 border p-4"
			>
				<legend class="fieldset-legend">Profile</legend>

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

		<fieldset
			class="fieldset bg-base-200/40 border-base-300 rounded-box mt-6 w-full min-w-0 border p-4"
		>
			<legend class="fieldset-legend">Access</legend>

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

		<fieldset
			class="fieldset bg-base-200/40 border-base-300 rounded-box mt-6 w-full min-w-0 border p-4"
		>
			<legend class="fieldset-legend">Plugins</legend>

			{#if pluginState.plugins.length === 0}
				<span class="label whitespace-normal">The hub loaded no plugin.</span>
			{:else}
				{#if user.is_administrator}
					<div role="alert" class="alert alert-info alert-soft">
						<span class="text-sm">
							{isSelf ? 'You are' : `${name} is`} an administrator, so
							{isSelf ? 'you' : 'they'} already reach every plugin. The allowlist applies again when the
							mark goes.
						</span>
					</div>
				{:else}
					<span class="label whitespace-normal">
						The plugins that {name} can turn on in a workspace. A removal keeps every plugin that a workspace
						already uses.
					</span>
				{/if}
				<ul class="border-base-300 bg-base-100 rounded-box mt-1 divide-y border">
					{#each pluginState.plugins as plugin (plugin.id)}
						<li class="flex items-center justify-between gap-3 px-4 py-3">
							<span class="min-w-0">
								<span class="block text-sm font-semibold">{displayNameOf(plugin, plugin.id)}</span>
								<span class="text-base-content/50 block truncate font-mono text-[11px]">
									{plugin.id}
								</span>
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
	{/if}
</div>

<ConfirmDialog bind:this={confirmDialog} />
