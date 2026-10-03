<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import {
		ApiError,
		PROBLEM_TYPE,
		api,
		createWriteIntent,
		isProblem,
		type Candidate,
		type Member
	} from '$lib/api';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import { userState } from '$lib/state/user.svelte';
	import { loadWorkspaces } from '$lib/state/workspaces.svelte';

	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const intent = createWriteIntent();
	const workspaceId = $derived(data.workspace.id);
	const selfId = $derived(userState.user?.id ?? '');

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

			return error.body.errors?.[0]?.detail ?? error.body.title;
		}

		return fallback;
	}

	async function load(): Promise<void> {
		status = 'loading';

		try {
			const [memberList, candidateList] = await Promise.all([
				api.workspaces.members(workspaceId),
				api.workspaces.candidates(workspaceId)
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
				intent.keyFor({ user_id: chosen })
			);
			if (added === null) {
				return;
			}

			intent.settle();
			await load();
		} catch (error) {
			banner = failureMessage(error, 'The hub added nobody. Try again.');
		} finally {
			busy = false;
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
	<h1 class="font-display text-[42px] leading-[1.05] tracking-tight">Members</h1>
	<p class="text-base-content/60 mt-3 text-sm">
		The people who share the records of {data.workspace.name}.
	</p>

	{#if banner}
		<div class="alert alert-error mt-6">
			<span>{banner}</span>
		</div>
	{/if}

	{#if status === 'loading'}
		<div class="mt-8 flex flex-col gap-2" aria-hidden="true">
			{#each [0, 1, 2] as i (i)}
				<span class="skeleton h-10 w-full"></span>
			{/each}
		</div>
	{:else if status === 'error'}
		<div class="alert alert-error mt-6">
			<span>The hub answered no member. Reload the page.</span>
		</div>
	{:else}
		<ul class="border-base-300 rounded-box mt-8 divide-y border">
			{#each members as member (member.user_id)}
				<li class="flex items-center justify-between gap-3 px-4 py-3">
					<span class="text-sm">
						{nameOf(member)}
						{#if member.user_id === selfId}
							<span class="badge badge-ghost badge-sm ml-2">You</span>
						{/if}
					</span>
					<button
						type="button"
						class="btn btn-ghost btn-xs text-error"
						disabled={busy}
						onclick={() => remove(member)}
					>
						{member.user_id === selfId ? 'Leave' : 'Remove'}
					</button>
				</li>
			{/each}
		</ul>

		<section class="mt-8">
			<h2 class="text-base-content/70 font-mono text-[11px] tracking-wider uppercase">
				Add a member
			</h2>

			{#if candidates.length === 0}
				<p class="text-base-content/60 mt-3 text-sm">Every person on this hub is a member.</p>
			{:else}
				<form class="mt-3 flex gap-2" onsubmit={add}>
					<select class="select select-bordered select-sm w-full max-w-xs" bind:value={chosen}>
						<option value="" disabled>Choose a person</option>
						{#each candidates as candidate (candidate.user_id)}
							<option value={candidate.user_id}>{nameOf(candidate)}</option>
						{/each}
					</select>
					<button type="submit" class="btn btn-primary btn-sm" disabled={busy || chosen === ''}>
						Add
					</button>
				</form>
			{/if}
		</section>
	{/if}
</div>

<ConfirmDialog bind:this={confirmDialog} />
