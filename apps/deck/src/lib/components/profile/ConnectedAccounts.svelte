<script lang="ts">
	import { ApiError, LOCAL_PROVIDER, api, type ProviderIdentity } from '$lib/api';
	import { startAttach } from '$lib/api/client';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import Panel from '$lib/components/ui/Panel.svelte';
	import SkeletonList from '$lib/components/ui/SkeletonList.svelte';
	import { userState } from '$lib/state/user.svelte';

	interface Props {
		identities: ProviderIdentity[];
		status: 'loading' | 'ready' | 'error';
		onreload: () => Promise<void>;
	}

	let { identities, status, onreload }: Props = $props();

	let detaching = $state<string | null>(null);
	let failure = $state<string | null>(null);
	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();

	const currentProvider = $derived(userState.user?.provider ?? null);

	function failureMessage(error: unknown): string {
		if (error instanceof ApiError && typeof error.body === 'object' && error.body !== null) {
			const reason = (error.body as { error?: unknown }).error;
			if (typeof reason === 'string') {
				return reason;
			}
		}

		return 'The hub kept the account connected. Try again.';
	}

	async function detach(provider: string): Promise<void> {
		if (provider === currentProvider) {
			const proceed = await confirmDialog?.ask(
				"This is the account you're signed in with. Disconnecting it will sign you out.",
				{ title: 'Disconnect this account?', confirmLabel: 'Disconnect' }
			);

			if (!proceed) {
				return;
			}
		}

		detaching = provider;
		failure = null;

		try {
			await api.identities.detach(provider);
			await onreload();
		} catch (error) {
			failure = failureMessage(error);
		} finally {
			detaching = null;
		}
	}

	function connect(provider: string): void {
		void startAttach(provider).then((address) => {
			window.location.href = address;
		});
	}
</script>

<Panel>
	{#if failure}
		<Alert kind="error"><span class="text-sm">{failure}</span></Alert>
	{/if}

	{#if status === 'loading'}
		<SkeletonList count={2} row="h-16" />
	{:else if status === 'error'}
		<Alert kind="error">
			<span class="text-sm">Failed to load the connected accounts.</span>
			<button type="button" class="btn btn-sm" onclick={() => void onreload()}>Retry</button>
		</Alert>
	{:else}
		<ul class="list panel">
			{#each identities as identity (identity.provider)}
				<li class="list-row flex items-center justify-between gap-3">
					<span class="min-w-0">
						<span class="flex items-center gap-2">
							<span class="text-sm font-semibold">{identity.name}</span>
							{#if identity.provider === currentProvider}
								<span class="badge badge-ghost badge-xs">Current session</span>
							{/if}
						</span>
						{#if identity.attached}
							<span class="meta block truncate font-mono">
								{identity.username ?? identity.display_name ?? 'Connected'}
							</span>
						{:else}
							<span class="meta block font-mono">Not connected</span>
						{/if}
					</span>

					{#if identity.attached}
						<button
							type="button"
							class="btn btn-ghost btn-sm text-error"
							disabled={detaching === identity.provider}
							onclick={() => void detach(identity.provider)}
						>
							Disconnect
						</button>
					{:else if identity.provider === LOCAL_PROVIDER}
						<span class="label whitespace-normal">An administrator gives this account</span>
					{:else}
						<button type="button" class="btn btn-sm" onclick={() => connect(identity.provider)}>
							Connect
						</button>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</Panel>

<ConfirmDialog bind:this={confirmDialog} />
