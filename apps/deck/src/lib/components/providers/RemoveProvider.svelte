<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import { api, type Provider, type UserRef } from '$lib/api';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import Panel from '$lib/components/ui/Panel.svelte';
	import { failureMessage } from '$lib/providers/form';

	interface Props {
		provider: Provider;
	}

	let { provider }: Props = $props();

	let busy = $state(false);
	let failure = $state<string | null>(null);
	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();

	function names(people: UserRef[]): string {
		return people
			.map((one) => [one.first_name, one.last_name].filter(Boolean).join(' ') || one.id)
			.join(', ');
	}

	function question(current: Provider): string {
		const count = current.identity_count;
		const stranded = current.administrators_without_sign_in;
		let message = `Removing ${current.name} deletes ${count} ${count === 1 ? 'identity' : 'identities'}.`;

		if (current.preset === 'local') {
			message += ' Every local account goes with it.';
		}

		if (stranded.length > 0) {
			message += ` ${names(stranded)} then ${stranded.length === 1 ? 'holds' : 'hold'} no way to sign in, and only maroid user password brings ${stranded.length === 1 ? 'that administrator' : 'them'} back.`;
		}

		return message;
	}

	/** Reads the provider again, so the report of the removal is the one of this moment. */
	async function remove(): Promise<void> {
		failure = null;

		let current: Provider;

		try {
			const tagged = await api.providers.get(provider.id);
			if (tagged === null) {
				return;
			}

			current = tagged.value;
		} catch (error) {
			failure = failureMessage(error, 'The hub answered no report. Try again.');

			return;
		}

		const confirmed = await confirmDialog?.ask(question(current), {
			title: 'Remove this provider?',
			confirmLabel: 'Remove'
		});
		if (!confirmed) {
			return;
		}

		busy = true;

		try {
			await api.providers.remove(provider.id);
			await goto(resolve('/admin/providers'));
		} catch (error) {
			failure = failureMessage(error, 'The provider stays. Try again.');
		} finally {
			busy = false;
		}
	}
</script>

<Panel class="border-error/40">
	{#if failure}
		<Alert kind="error"><span class="text-sm">{failure}</span></Alert>
	{/if}
	<span class="label whitespace-normal">
		{provider.identity_count}
		{provider.identity_count === 1 ? 'identity' : 'identities'} of this provider go with it.
	</span>
	<div class="flex justify-end">
		<button type="button" class="btn btn-error btn-sm" disabled={busy} onclick={remove}>
			Remove the provider
		</button>
	</div>
</Panel>

<ConfirmDialog bind:this={confirmDialog} />
