<script lang="ts">
	import { goto, invalidateAll } from '$app/navigation';
	import { resolve } from '$app/paths';

	import {
		ApiError,
		PROBLEM_TYPE,
		api,
		createWriteIntent,
		isProblem,
		type Candidate,
		type Member,
		type Role
	} from '$lib/api';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import { PERMISSION, ROLES, holds, roleLabel } from '$lib/permissions';
	import { userState } from '$lib/state/user.svelte';
	import { loadWorkspaces } from '$lib/state/workspaces.svelte';

	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	let newMemberRole = $state<Role>('viewer');

	const intent = createWriteIntent();
	const workspaceId = $derived(data.workspace.id);
	const selfId = $derived(userState.user?.id ?? '');
	const canManage = $derived(holds(data.workspace, PERMISSION.membersWrite));

	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let members = $state<Member[]>([]);
	let candidates = $state<Candidate[]>([]);
	let chosen = $state('');
	let busy = $state(false);
	let banner = $state<string | null>(null);
	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();

	function nameOf(person: { first_name?: string; last_name?: string }): string {
		return [person.first_name, person.last_name].filter(Boolean).join(' ') || 'Unnamed person';
	}

	function failureMessage(error: unknown, fallback: string): string {
		if (error instanceof ApiError && isProblem(error.body)) {
			if (error.body.type === PROBLEM_TYPE.memberExists) {
				return 'That person is already a member.';
			}

			if (error.body.type === PROBLEM_TYPE.managerLast) {
				return 'A workspace keeps one manager. Make another member a manager first.';
			}

			return error.body.errors?.[0]?.detail ?? error.body.title;
		}

		return fallback;
	}

	async function load(): Promise<void> {
		status = 'loading';

		try {
			const [memberList, candidateList] = await Promise.all([
				api.workspaces.members(workspaceId),
				canManage ? api.workspaces.candidates(workspaceId) : Promise.resolve([])
			]);
			if (memberList === null || candidateList === null) {
				return;
			}

			members = memberList;
			candidates = candidateList;
			chosen = '';
			status = 'ready';
		} catch (error) {
			console.error('Failed to load the members', error);
			status = 'error';
		}
	}

	async function add(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		if (busy || chosen === '') {
			return;
		}

		busy = true;
		banner = null;

		try {
			const added = await api.workspaces.addMember(
				workspaceId,
				chosen,
				newMemberRole,
				intent.keyFor({ user_id: chosen, role: newMemberRole })
			);
			if (added === null) {
				return;
			}

			intent.settle();
			newMemberRole = 'viewer';
			await load();
		} catch (error) {
			banner = failureMessage(error, 'The hub added nobody. Try again.');
		} finally {
			busy = false;
		}
	}

	async function changeRole(member: Member, role: Role): Promise<void> {
		if (role === member.role) {
			return;
		}

		busy = true;
		banner = null;

		try {
			await api.workspaces.changeRole(workspaceId, member.user_id, role);

			if (member.user_id === selfId) {
				await Promise.all([invalidateAll(), loadWorkspaces()]);
			}
		} catch (error) {
			banner = failureMessage(error, 'The role stays. Try again.');
		} finally {
			busy = false;
			await load();
		}
	}

	async function remove(member: Member): Promise<void> {
		const leaving = member.user_id === selfId;
		const proceed = await confirmDialog?.ask(
			leaving
				? 'You will no longer reach this workspace or its records.'
				: `${nameOf(member)} will no longer reach this workspace. Its records stay.`,
			{
				title: leaving ? 'Leave this workspace?' : 'Remove this member?',
				confirmLabel: leaving ? 'Leave' : 'Remove'
			}
		);

		if (!proceed) {
			return;
		}

		busy = true;
		banner = null;

		try {
			await api.workspaces.removeMember(workspaceId, member.user_id);

			if (leaving) {
				await loadWorkspaces();
				await goto(resolve('/'));

				return;
			}

			await load();
		} catch (error) {
			banner = failureMessage(error, 'The membership stays. Try again.');
		} finally {
			busy = false;
		}
	}

	$effect(() => {
		void workspaceId;
		void load();
	});
</script>

<div class="max-w-2xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">Members</h1>
			<p class="page-lead">The people who share the records of {data.workspace.name}.</p>
		</div>
	</header>

	{#if banner}
		<div role="alert" class="alert alert-error alert-soft mb-6">
			<span>{banner}</span>
		</div>
	{/if}

	{#if status === 'loading'}
		<div class="flex flex-col gap-2" aria-hidden="true">
			{#each [0, 1, 2] as i (i)}
				<span class="skeleton h-14 w-full"></span>
			{/each}
		</div>
	{:else if status === 'error'}
		<div role="alert" class="alert alert-error alert-soft">
			<span>The hub answered no member. Reload the page.</span>
		</div>
	{:else}
		<ul class="list panel">
			{#each members as member (member.user_id)}
				<li class="list-row items-center">
					<span class="list-col-grow font-medium">
						{nameOf(member)}
						{#if member.user_id === selfId}
							<span class="badge badge-ghost badge-sm ml-2">You</span>
						{/if}
					</span>
					{#if canManage}
						<select
							class="select select-sm w-32"
							aria-label="The role of {nameOf(member)}"
							value={member.role}
							disabled={busy}
							onchange={(event) => changeRole(member, event.currentTarget.value as Role)}
						>
							{#each ROLES as role (role)}
								<option value={role}>{roleLabel(role)}</option>
							{/each}
						</select>
					{:else}
						<span class="badge badge-outline badge-sm">{roleLabel(member.role)}</span>
					{/if}
					{#if canManage || member.user_id === selfId}
						<button
							type="button"
							class="btn btn-ghost btn-sm text-error"
							disabled={busy}
							onclick={() => remove(member)}
						>
							{member.user_id === selfId ? 'Leave' : 'Remove'}
						</button>
					{/if}
				</li>
			{/each}
		</ul>

		{#if canManage}
			<section class="section">
				<h2 class="section-title">Add a member</h2>

				{#if candidates.length === 0}
					<div class="empty-state">Every person on this hub is a member.</div>
				{:else}
					<form class="panel flex flex-wrap items-end gap-2 p-4" onsubmit={add}>
						<select
							class="select select-sm min-w-0 flex-1"
							aria-label="The new member"
							bind:value={chosen}
						>
							<option value="" disabled>Choose a person</option>
							{#each candidates as candidate (candidate.user_id)}
								<option value={candidate.user_id}>{nameOf(candidate)}</option>
							{/each}
						</select>
						<select
							class="select select-sm w-32"
							aria-label="The role of the new member"
							bind:value={newMemberRole}
						>
							{#each ROLES as role (role)}
								<option value={role}>{roleLabel(role)}</option>
							{/each}
						</select>
						<button type="submit" class="btn btn-primary btn-sm" disabled={busy || chosen === ''}>
							Add
						</button>
					</form>
				{/if}
			</section>
		{/if}
	{/if}
</div>

<ConfirmDialog bind:this={confirmDialog} />
