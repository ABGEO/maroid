<script lang="ts">
	import { onMount } from 'svelte';

	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';

	import { buildAuthUrl } from '$lib/api/client';

	const hasError = page.url.searchParams.get('error') === 'auth_failed';

	onMount(() => {
		if (!hasError) {
			void goto(resolve('/'));
		}
	});
</script>

{#if hasError}
	<div class="bg-base-200 flex min-h-screen items-center justify-center">
		<div class="card bg-base-100 w-full max-w-sm shadow-md">
			<div class="card-body gap-6">
				<div role="alert" class="alert alert-error alert-soft">
					<span>Authentication failed. Unable to sign in. Please try again.</span>
				</div>

				<button
					class="btn btn-primary w-full"
					onclick={() => {
						window.location.href = buildAuthUrl();
					}}
				>
					Try Again
				</button>
			</div>
		</div>
	</div>
{/if}
