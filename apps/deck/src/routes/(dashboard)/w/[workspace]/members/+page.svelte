<script lang="ts">
	import { goto, invalidateAll } from '$app/navigation';
	import { resolve } from '$app/paths';

	import { ApiError, PROBLEM_TYPE, api, type Candidate, type Member, type Role } from '$lib/api';
	import Alert from '$lib/components/ui/Alert.svelte';
	import Section from '$lib/components/ui/Section.svelte';
	import SkeletonList from '$lib/components/ui/SkeletonList.svelte';
	import AddMemberForm from '$lib/components/workspace/AddMemberForm.svelte';
	import MemberRow from '$lib/components/workspace/MemberRow.svelte';
	import { PERMISSION, holds } from '$lib/permissions';
	import { userState } from '$lib/state/user.svelte';
	import { loadWorkspaces } from '$lib/state/workspaces.svelte';

	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const workspaceId = $derived(data.workspace.id);
	const selfId = $derived(userState.user?.id ?? '');
	const canManage = $derived(holds(data.workspace, PERMISSION.membersWrite));

	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let members = $state<Member[]>([]);
	let candidates = $state<Candidate[]>([]);
	let busy = $state(false);
	let banner = $state<string | null>(null);

	function failureMessage(error: unknown, fallback: string): string {
		if (error instanceof ApiError && error.problem !== null) {
			if (error.is(PROBLEM_TYPE.managerLast)) {
				return 'A workspace keeps one manager. Make another member a manager first.';
			}

			return error.problem.errors?.[0]?.detail ?? error.problem.title;
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
			status = 'ready';
		} catch (error) {
			console.error('Failed to load the members', error);
			status = 'error';
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
		busy = true;
		banner = null;

		try {
			await api.workspaces.removeMember(workspaceId, member.user_id);

			if (member.user_id === selfId) {
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
		<Alert kind="error" class="mb-6"><span>{banner}</span></Alert>
	{/if}

	{#if status === 'loading'}
		<SkeletonList row="h-14" />
	{:else if status === 'error'}
		<Alert kind="error"><span>The hub answered no member. Reload the page.</span></Alert>
	{:else}
		<ul class="list panel">
			{#each members as member (member.user_id)}
				<MemberRow
					{member}
					isSelf={member.user_id === selfId}
					{canManage}
					disabled={busy}
					onrolechange={(role) => changeRole(member, role)}
					onremove={() => remove(member)}
				/>
			{/each}
		</ul>

		{#if canManage}
			<Section title="Add a member">
				{#if candidates.length === 0}
					<div class="empty-state">Every person on this hub is a member.</div>
				{:else}
					<AddMemberForm {workspaceId} {candidates} onadded={load} />
				{/if}
			</Section>
		{/if}
	{/if}
</div>
