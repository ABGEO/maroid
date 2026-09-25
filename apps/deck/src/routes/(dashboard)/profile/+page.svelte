<script lang="ts">
	import { onMount } from 'svelte';

	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';

	import { ApiError, api, type ProviderIdentity } from '$lib/api';
	import { startAttach } from '$lib/api/client';
	import { authFailureMessage } from '$lib/auth/messages';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import { userState } from '$lib/state/user.svelte';

	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let identities = $state<ProviderIdentity[]>([]);
	let banner = $state<string | null>(null);
	let detaching = $state<string | null>(null);
	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();

	const currentProvider = $derived(userState.user?.provider ?? null);

	async function load(): Promise<void> {
		status = 'loading';

		try {
			const result = await api.identities.list();
			if (result === null) {
				return;
			}

			identities = result;
			status = 'ready';
		} catch (error) {
			console.error('Failed to load the connected accounts', error);
			status = 'error';
		}
	}

	function detachFailureMessage(error: unknown): string {
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
		banner = null;

		try {
			await api.identities.detach(provider);
			await load();
		} catch (error) {
			banner = detachFailureMessage(error);
		} finally {
			detaching = null;
		}
	}

	function connect(provider: string): void {
		void startAttach(provider).then((address) => {
			window.location.href = address;
		});
	}

	onMount(() => {
		const reason = page.url.searchParams.get('error');

		if (reason) {
			banner = authFailureMessage(reason);
			void goto(resolve('/profile'), { replaceState: true, keepFocus: true, noScroll: true });
		}

		void load();
	});
</script>

<div class="max-w-2xl">
	<h1 class="font-display text-[42px] leading-[1.05] tracking-tight">Profile</h1>
	<p class="text-base-content/60 mt-3 text-sm">
		Your name and the accounts that reach this record.
	</p>

	{#if banner}
		<div class="alert alert-error mt-6">
			<span>{banner}</span>
		</div>
	{/if}

	<section class="mt-8">
		<h2 class="text-base-content/70 font-mono text-[11px] tracking-wider uppercase">Name</h2>

		<div class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
			<label class="form-control">
				<span class="label-text mb-1 block text-xs">First name</span>
				<input
					type="text"
					class="input input-bordered input-sm w-full"
					placeholder="First name"
					value={userState.user?.first_name ?? ''}
					disabled
				/>
			</label>
			<label class="form-control">
				<span class="label-text mb-1 block text-xs">Last name</span>
				<input
					type="text"
					class="input input-bordered input-sm w-full"
					placeholder="Last name"
					value={userState.user?.last_name ?? ''}
					disabled
				/>
			</label>
		</div>

		<div class="mt-3 flex items-center gap-3">
			<button type="button" class="btn btn-primary btn-sm" disabled>Save</button>
			<span class="text-base-content/50 font-mono text-[11px]">
				Saving a name isn't available yet.
			</span>
		</div>
	</section>

	<section class="mt-10">
		<h2 class="text-base-content/70 font-mono text-[11px] tracking-wider uppercase">
			Connected accounts
		</h2>

		<div class="mt-3">
			{#if status === 'loading'}
				<div class="flex flex-col gap-2">
					{#each [0, 1] as i (i)}
						<span class="skeleton h-16 w-full" aria-hidden="true"></span>
					{/each}
				</div>
			{:else if status === 'error'}
				<div class="alert alert-error">
					<span>Failed to load the connected accounts.</span>
					<button type="button" class="btn btn-sm" onclick={() => void load()}>Retry</button>
				</div>
			{:else}
				<ul class="border-base-300 divide-base-300 divide-y rounded-md border">
					{#each identities as identity (identity.provider)}
						<li class="flex items-center gap-4 px-4 py-3">
							<div class="min-w-0 flex-1">
								<div class="flex items-center gap-2">
									<span class="text-[15px] font-semibold">{identity.name}</span>
									{#if identity.provider === currentProvider}
										<span class="badge badge-ghost badge-xs">Current session</span>
									{/if}
								</div>
								{#if identity.attached}
									<div class="text-base-content/50 truncate font-mono text-[11px]">
										{identity.username ?? identity.display_name ?? 'Connected'}
									</div>
								{:else}
									<div class="text-base-content/35 font-mono text-[11px]">Not connected</div>
								{/if}
							</div>

							{#if identity.attached}
								<button
									type="button"
									class="btn btn-error btn-soft btn-sm"
									disabled={detaching === identity.provider}
									onclick={() => void detach(identity.provider)}
								>
									{#if detaching === identity.provider}
										<span class="loading loading-spinner loading-xs"></span>
									{/if}
									Disconnect
								</button>
							{:else}
								<button type="button" class="btn btn-sm" onclick={() => connect(identity.provider)}>
									Connect
								</button>
							{/if}
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	</section>
</div>

<ConfirmDialog bind:this={confirmDialog} />
