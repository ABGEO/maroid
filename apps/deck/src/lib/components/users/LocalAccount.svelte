<script lang="ts">
	import { resolve } from '$app/paths';

	import { LOCAL_PROVIDER, api, createWriteIntent, type UserIdentity } from '$lib/api';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import Panel from '$lib/components/ui/Panel.svelte';
	import PasswordFields from '$lib/components/users/PasswordFields.svelte';
	import { failureMessage, passwordProblem } from '$lib/providers/form';

	interface Props {
		userId: string;
		name: string;
		identities: UserIdentity[];
		localProvider: boolean;
	}

	let { userId, name, identities = $bindable(), localProvider }: Props = $props();

	const intent = createWriteIntent();

	let email = $state('');
	let password = $state('');
	let repeated = $state('');
	let busy = $state(false);
	let notice = $state<string | null>(null);
	let failure = $state<string | null>(null);
	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();

	const account = $derived(identities.find((one) => one.provider === LOCAL_PROVIDER));

	/** Runs one write of the local account, and reads the identities after it. */
	async function write(
		send: () => Promise<unknown>,
		done: string,
		fallback: string
	): Promise<void> {
		busy = true;
		failure = null;
		notice = null;

		try {
			await send();

			const held = await api.users.identities(userId);
			if (held !== null) {
				identities = held;
			}

			password = '';
			repeated = '';
			notice = done;
		} catch (error) {
			failure = failureMessage(error, fallback);
		} finally {
			busy = false;
		}
	}

	/** Answers whether the two passwords pass, and shows the problem when they do not. */
	function passwordsPass(): boolean {
		failure = passwordProblem(password, repeated);

		return failure === null;
	}

	async function give(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		if (!passwordsPass()) {
			return;
		}

		const address = email.trim();

		await write(
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

	async function reset(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		if (!passwordsPass()) {
			return;
		}

		await write(
			() => api.users.resetLocalPassword(userId, password),
			'The new password works from the next sign in. The old one no longer does.',
			'The password stays as it was. Try again.'
		);
	}

	async function remove(): Promise<void> {
		const confirmed = await confirmDialog?.ask(
			`${name} will no longer sign in with ${account?.username ?? 'the local account'}.`,
			{ title: 'Remove the local account?', confirmLabel: 'Remove' }
		);
		if (!confirmed) {
			return;
		}

		await write(
			() => api.users.removeLocalAccount(userId),
			'The local account is gone.',
			'The local account stays. Try again.'
		);
	}
</script>

<Panel>
	{#if notice}
		<Alert kind="success"><span class="text-sm">{notice}</span></Alert>
	{/if}
	{#if failure}
		<Alert kind="error"><span class="text-sm">{failure}</span></Alert>
	{/if}

	{#if account}
		<span class="label whitespace-normal">
			{name} signs in with <span class="font-mono">{account.username}</span> and a password.
		</span>

		<form class="mt-1 grid gap-3 sm:grid-cols-2" onsubmit={reset}>
			<PasswordFields bind:password bind:repeated disabled={busy} />
			<div class="flex justify-end gap-2 sm:col-span-2">
				<button
					type="button"
					class="btn btn-ghost btn-sm text-error"
					disabled={busy || identities.length < 2}
					title={identities.length < 2 ? 'It is the only way that this user signs in.' : ''}
					onclick={remove}
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

		<form class="mt-1 grid gap-3 sm:grid-cols-2" onsubmit={give}>
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
			<PasswordFields bind:password bind:repeated label="Password" disabled={busy} />
			<div class="flex justify-end sm:col-span-2">
				<button type="submit" class="btn btn-primary btn-sm" disabled={busy}>
					Give the account
				</button>
			</div>
		</form>
	{/if}
</Panel>

<ConfirmDialog bind:this={confirmDialog} />
