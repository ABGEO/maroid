<script lang="ts">
	import type { UserChange, UserRecord } from '$lib/api';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import Panel from '$lib/components/ui/Panel.svelte';

	interface Props {
		user: UserRecord;
		name: string;
		disabled?: boolean;
		onchange: (update: UserChange, fallback: string) => Promise<boolean>;
	}

	let { user, name, disabled = false, onchange }: Props = $props();

	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();

	// A switch flips before the person confirms, so a cancel or a refusal flips it back.
	async function setAdministrator(input: HTMLInputElement): Promise<void> {
		const on = input.checked;
		const confirmed =
			on ||
			(await confirmDialog?.ask(`${name} will no longer administer this instance.`, {
				title: 'Unmark this administrator?',
				confirmLabel: 'Unmark'
			}));

		if (!confirmed || !(await onchange({ is_administrator: on }, 'The mark stays as it was.'))) {
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
			(await onchange(
				{ status: blocked ? 'blocked' : 'active' },
				'The user stays as they were. Try again.'
			));

		if (!accepted) {
			input.checked = !blocked;
		}
	}
</script>

<Panel>
	<label class="flex cursor-pointer flex-wrap items-center gap-2">
		<input
			type="checkbox"
			class="toggle toggle-sm"
			checked={user.is_administrator}
			{disabled}
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
			{disabled}
			onchange={(event) => setBlocked(event.currentTarget)}
		/>
		<span class="text-sm">Blocked</span>
		<span class="label whitespace-normal">
			Reaches nothing until you unblock them. Their records stay.
		</span>
	</label>
</Panel>

<ConfirmDialog bind:this={confirmDialog} />
