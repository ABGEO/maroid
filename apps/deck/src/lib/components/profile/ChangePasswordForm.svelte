<script lang="ts">
	import { ApiError, PROBLEM_TYPE, api } from '$lib/api';
	import Alert from '$lib/components/ui/Alert.svelte';
	import Panel from '$lib/components/ui/Panel.svelte';
	import PasswordFields from '$lib/components/users/PasswordFields.svelte';
	import { fieldErrors, passwordProblem } from '$lib/providers/form';

	interface Props {
		username?: string;
	}

	let { username }: Props = $props();

	let currentPassword = $state('');
	let newPassword = $state('');
	let repeatedPassword = $state('');
	let changing = $state(false);
	let notice = $state<string | null>(null);
	let failure = $state<string | null>(null);
	let currentFailure = $state<string | null>(null);

	const problem = $derived(
		newPassword === '' ? null : passwordProblem(newPassword, repeatedPassword)
	);
	const canChange = $derived(
		currentPassword !== '' && newPassword !== '' && problem === null && !changing
	);

	function failureMessage(error: unknown): string {
		if (error instanceof ApiError && error.is(PROBLEM_TYPE.notReady)) {
			return 'The sign-in service does not answer. Try again in a moment.';
		}

		return fieldErrors(error)['/password'] ?? 'Your password stays as it was. Try again.';
	}

	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		changing = true;
		notice = null;
		failure = null;
		currentFailure = null;

		try {
			await api.identities.changePassword(currentPassword, newPassword);

			currentPassword = '';
			newPassword = '';
			repeatedPassword = '';
			notice = 'The new password works from the next sign in.';
		} catch (error) {
			if (fieldErrors(error)['/current_password']) {
				currentFailure = 'This is not your current password.';
			} else {
				failure = failureMessage(error);
			}
		} finally {
			changing = false;
		}
	}
</script>

<form onsubmit={submit}>
	<Panel>
		{#if notice}
			<Alert kind="success"><span class="text-sm">{notice}</span></Alert>
		{/if}
		{#if failure}
			<Alert kind="error"><span class="text-sm">{failure}</span></Alert>
		{/if}

		<span class="label whitespace-normal">
			You sign in with <span class="font-mono">{username}</span> and a password.
		</span>

		<div class="mt-1 grid gap-3 sm:grid-cols-2">
			<label class="flex flex-col gap-1 sm:col-span-2">
				<span class="label">Current password</span>
				<input
					class="input input-sm w-full"
					class:input-error={currentFailure !== null}
					type="password"
					autocomplete="current-password"
					bind:value={currentPassword}
					disabled={changing}
					required
				/>
				{#if currentFailure}
					<span class="label text-error">{currentFailure}</span>
				{/if}
			</label>
			<PasswordFields
				bind:password={newPassword}
				bind:repeated={repeatedPassword}
				disabled={changing}
			/>
			{#if problem}
				<span class="label whitespace-normal sm:col-span-2">{problem}</span>
			{/if}
			<div class="flex justify-end sm:col-span-2">
				<button type="submit" class="btn btn-primary btn-sm" disabled={!canChange}>
					Set the password
				</button>
			</div>
		</div>
	</Panel>
</form>
