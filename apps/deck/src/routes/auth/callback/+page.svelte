<script lang="ts">
	import { onMount } from 'svelte';

	import { page } from '$app/state';

	import { startSignIn } from '#lib/api/client.ts';
	import { authFailureMessage } from '#lib/auth/messages.ts';
	import { addressAfterSignIn, RETURN_ADDRESS_PARAMETER } from '#lib/auth/return-address.ts';
	import { documentTitle } from '#lib/navigation/title.ts';

	const reason = page.url.searchParams.get('error');
	const message = reason ? authFailureMessage(reason) : '';

	const canRetry = reason === 'auth_failed';

	const returnAddress = addressAfterSignIn(
		page.url.searchParams.get(RETURN_ADDRESS_PARAMETER),
		page.url.origin
	);

	onMount(() => {
		if (!reason) {
			// A replace keeps the landing out of the history, so Back does not reopen it.
			window.location.replace(returnAddress);
		}
	});
</script>

<svelte:head><title>{documentTitle('Sign in')}</title></svelte:head>

{#if reason}
	<div class="bg-base-200 flex min-h-screen items-center justify-center">
		<div class="card panel w-full max-w-sm">
			<div class="card-body gap-6">
				<div role="alert" class="alert alert-error alert-soft">
					<span>{message}</span>
				</div>

				{#if canRetry}
					<button
						class="btn btn-primary w-full"
						onclick={() => {
							void startSignIn(returnAddress).then((address) => {
								window.location.href = address;
							});
						}}
					>
						Try Again
					</button>
				{/if}
			</div>
		</div>
	</div>
{/if}
