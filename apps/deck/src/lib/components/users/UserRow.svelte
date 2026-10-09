<script lang="ts">
	import { resolve } from '$app/paths';

	import type { UserChange, UserRecord } from '$lib/api';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import UserBadges from '$lib/components/users/UserBadges.svelte';
	import { personName } from '$lib/users/name';

	interface Props {
		user: UserRecord;
		isSelf: boolean;
		disabled?: boolean;
		oninvite: () => void;
		onchange: (update: UserChange) => void;
	}

	let { user, isSelf, disabled = false, oninvite, onchange }: Props = $props();

	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();

	const name = $derived(personName(user));

	async function unmark(): Promise<void> {
		const proceed = await confirmDialog?.ask(`${name} will no longer administer this instance.`, {
			title: 'Unmark this administrator?',
			confirmLabel: 'Unmark'
		});

		if (proceed) {
			onchange({ is_administrator: false });
		}
	}

	async function block(): Promise<void> {
		const proceed = await confirmDialog?.ask(
			`${name} will reach nothing until you unblock them. Their records stay.`,
			{ title: 'Block this user?', confirmLabel: 'Block' }
		);

		if (proceed) {
			onchange({ status: 'blocked' });
		}
	}
</script>

<li class="list-row flex flex-wrap items-center justify-between gap-3">
	<span class="flex items-center gap-2 text-sm">
		<a
			class="link link-hover font-semibold"
			href={resolve('/(dashboard)/admin/users/[user]', { user: user.id })}
		>
			{name}
		</a>
		<UserBadges {user} {isSelf} />
	</span>
	<span class="flex items-center gap-1">
		<button type="button" class="btn btn-ghost btn-sm" {disabled} onclick={oninvite}>
			Invite
		</button>
		{#if user.is_administrator}
			<button type="button" class="btn btn-ghost btn-sm" {disabled} onclick={unmark}>
				Unmark
			</button>
		{:else}
			<button
				type="button"
				class="btn btn-ghost btn-sm"
				{disabled}
				onclick={() => onchange({ is_administrator: true })}
			>
				Mark administrator
			</button>
		{/if}
		{#if user.status === 'blocked'}
			<button
				type="button"
				class="btn btn-ghost btn-sm"
				{disabled}
				onclick={() => onchange({ status: 'active' })}
			>
				Unblock
			</button>
		{:else}
			<button type="button" class="btn btn-ghost btn-sm text-error" {disabled} onclick={block}>
				Block
			</button>
		{/if}
	</span>

	<ConfirmDialog bind:this={confirmDialog} />
</li>
