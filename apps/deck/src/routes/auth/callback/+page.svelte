<script lang="ts">
	import { onMount } from 'svelte';

	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';

	import { buildAuthUrl } from '$lib/api/client';
	import { authFailureMessage } from '$lib/auth/messages';

	const reason = page.url.searchParams.get('error');
	const message = reason ? authFailureMessage(reason) : '';

	const canRetry = reason === 'auth_failed';

	onMount(() => {
		if (!reason) {
			void goto(resolve('/'));
		}
	});
</script>

{#if reason}
	<div class="bg-base-200 flex min-h-screen items-center justify-center">
		<div class="card bg-base-100 w-full max-w-sm shadow-md">
			<div class="card-body gap-6">
				<div role="alert" class="alert alert-error alert-soft">
					<span>{message}</span>
				</div>

				{#if canRetry}
					<button
						class="btn btn-primary w-full"
						onclick={() => {
							window.location.href = buildAuthUrl();
						}}
					>
						Try Again
					</button>
				{/if}
			</div>
		</div>
	</div>
{/if}
