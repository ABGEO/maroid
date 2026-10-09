<script lang="ts">
	import type { Member, Role } from '$lib/api';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import { ROLES, roleLabel } from '$lib/permissions';
	import { personName } from '$lib/users/name';

	interface Props {
		member: Member;
		isSelf: boolean;
		canManage: boolean;
		disabled?: boolean;
		onrolechange: (role: Role) => void;
		onremove: () => void;
	}

	let { member, isSelf, canManage, disabled = false, onrolechange, onremove }: Props = $props();

	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();

	const name = $derived(personName(member));

	async function remove(): Promise<void> {
		const proceed = await confirmDialog?.ask(
			isSelf
				? 'You will no longer reach this workspace or its records.'
				: `${name} will no longer reach this workspace. Its records stay.`,
			{
				title: isSelf ? 'Leave this workspace?' : 'Remove this member?',
				confirmLabel: isSelf ? 'Leave' : 'Remove'
			}
		);

		if (proceed) {
			onremove();
		}
	}
</script>

<li class="list-row items-center">
	<span class="list-col-grow font-medium">
		{name}
		{#if isSelf}
			<span class="badge badge-ghost badge-sm ml-2">You</span>
		{/if}
	</span>
	{#if canManage}
		<select
			class="select select-sm w-32"
			aria-label="The role of {name}"
			value={member.role}
			{disabled}
			onchange={(event) => onrolechange(event.currentTarget.value as Role)}
		>
			{#each ROLES as role (role)}
				<option value={role}>{roleLabel(role)}</option>
			{/each}
		</select>
	{:else}
		<span class="badge badge-outline badge-sm">{roleLabel(member.role)}</span>
	{/if}
	{#if canManage || isSelf}
		<button type="button" class="btn btn-ghost btn-sm text-error" {disabled} onclick={remove}>
			{isSelf ? 'Leave' : 'Remove'}
		</button>
	{/if}

	<ConfirmDialog bind:this={confirmDialog} />
</li>
