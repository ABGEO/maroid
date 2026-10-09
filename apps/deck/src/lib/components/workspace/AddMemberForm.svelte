<script lang="ts">
	import {
		ApiError,
		PROBLEM_TYPE,
		api,
		createWriteIntent,
		type Candidate,
		type Role
	} from '$lib/api';
	import Alert from '$lib/components/ui/Alert.svelte';
	import { ROLES, roleLabel } from '$lib/permissions';
	import { personName } from '$lib/users/name';

	interface Props {
		workspaceId: string;
		candidates: Candidate[];
		onadded: () => Promise<void>;
	}

	let { workspaceId, candidates, onadded }: Props = $props();

	const intent = createWriteIntent();

	let chosen = $state('');
	let role = $state<Role>('viewer');
	let busy = $state(false);
	let failure = $state<string | null>(null);

	function failureMessage(error: unknown): string {
		if (error instanceof ApiError && error.problem !== null) {
			if (error.is(PROBLEM_TYPE.memberExists)) {
				return 'That person is already a member.';
			}

			return error.problem.errors?.[0]?.detail ?? error.problem.title;
		}

		return 'The hub added nobody. Try again.';
	}

	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		if (busy || chosen === '') {
			return;
		}

		busy = true;
		failure = null;

		try {
			const added = await api.workspaces.addMember(
				workspaceId,
				chosen,
				role,
				intent.keyFor({ user_id: chosen, role })
			);
			if (added === null) {
				return;
			}

			intent.settle();
			chosen = '';
			role = 'viewer';
			await onadded();
		} catch (error) {
			failure = failureMessage(error);
		} finally {
			busy = false;
		}
	}
</script>

{#if failure}
	<Alert kind="error" class="mb-2"><span class="text-sm">{failure}</span></Alert>
{/if}

<form class="panel flex flex-wrap items-end gap-2 p-4" onsubmit={submit}>
	<select class="select select-sm min-w-0 flex-1" aria-label="The new member" bind:value={chosen}>
		<option value="" disabled>Choose a person</option>
		{#each candidates as candidate (candidate.user_id)}
			<option value={candidate.user_id}>{personName(candidate)}</option>
		{/each}
	</select>
	<select class="select select-sm w-32" aria-label="The role of the new member" bind:value={role}>
		{#each ROLES as one (one)}
			<option value={one}>{roleLabel(one)}</option>
		{/each}
	</select>
	<button type="submit" class="btn btn-primary btn-sm" disabled={busy || chosen === ''}>
		Add
	</button>
</form>
