<script lang="ts">
	import { resolve } from '$app/paths';

	import { ApiError, PROBLEM_TYPE, api } from '#lib/api/index.ts';
	import Alert from '#lib/components/ui/Alert.svelte';

	interface Target {
		id: string;
		name: string;
	}

	interface Props {
		onremoved: (workspaceId: string) => Promise<void> | void;
	}

	let { onremoved }: Props = $props();

	let dialog: HTMLDialogElement | undefined = $state();
	let target = $state<Target | null>(null);
	let typed = $state('');
	let removing = $state(false);
	let failure = $state<
		{ kind: 'members'; others: number } | { kind: 'other'; text: string } | null
	>(null);

	const matches = $derived(target !== null && typed === target.name);

	/** Opens the dialog for the workspace, with an empty field. */
	export function open(workspace: Target): void {
		target = workspace;
		typed = '';
		failure = null;
		dialog?.showModal();
	}

	function close(): void {
		if (!removing) {
			dialog?.close();
		}
	}

	function failureOf(error: unknown): NonNullable<typeof failure> {
		if (!(error instanceof ApiError) || error.problem === null) {
			return { kind: 'other', text: 'The workspace stays. Try again.' };
		}

		const body = error.body as { other_members?: unknown };
		if (error.is(PROBLEM_TYPE.membersRemain) && typeof body.other_members === 'number') {
			return { kind: 'members', others: body.other_members };
		}

		return { kind: 'other', text: error.problem.detail ?? error.problem.title };
	}

	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		if (removing || !matches || target === null) {
			return;
		}

		removing = true;
		failure = null;

		const workspaceId = target.id;

		try {
			await api.workspaces.remove(workspaceId);
		} catch (error) {
			failure = failureOf(error);

			return;
		} finally {
			removing = false;
		}

		dialog?.close();
		await onremoved(workspaceId);
	}
</script>

<dialog bind:this={dialog} class="modal" onclose={() => (target = null)}>
	<form class="modal-box" onsubmit={submit}>
		<h3 class="text-lg font-semibold">Remove {target?.name}?</h3>
		<p class="text-base-content/70 py-3 text-sm">
			Every record of the workspace goes: its settings, its files, and the records of each plugin.
			This cannot be undone.
		</p>

		{#if failure !== null && target !== null}
			<Alert kind="error" class="mb-3">
				{#if failure.kind === 'members'}
					<span>
						Remove the {failure.others} other {failure.others === 1 ? 'member' : 'members'} first.
						<a
							class="link"
							href={resolve('/(dashboard)/w/[workspace]/members', { workspace: target.id })}
						>
							Open the members
						</a>
					</span>
				{:else}
					<span>{failure.text}</span>
				{/if}
			</Alert>
		{/if}

		<fieldset class="fieldset">
			<legend class="fieldset-legend">Type the name of the workspace</legend>
			<input
				type="text"
				class="input w-full"
				autocomplete="off"
				spellcheck="false"
				aria-label="The name of the workspace"
				placeholder={target?.name}
				bind:value={typed}
			/>
		</fieldset>

		<div class="modal-action">
			<button type="button" class="btn btn-ghost btn-sm" disabled={removing} onclick={close}>
				Cancel
			</button>
			<button type="submit" class="btn btn-error btn-sm" disabled={!matches || removing}>
				Remove
			</button>
		</div>
	</form>
	<form method="dialog" class="modal-backdrop">
		<button disabled={removing}>close</button>
	</form>
</dialog>
