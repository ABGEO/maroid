<script lang="ts">
	import { startSignIn } from '$lib/api/client';
</script>

<div class="bg-base-200 flex min-h-screen items-center justify-center p-4">
	<div class="card bg-base-100 w-full max-w-sm shadow-md">
		<div class="card-body gap-6 text-center">
			<div class="flex flex-col gap-2">
				<h1 class="text-lg font-semibold">You signed out</h1>
				<p class="text-base-content/60 text-sm">
					Maroid ended this session. Sign in again to reach your records.
				</p>
			</div>

			<button
				class="btn btn-primary w-full"
				onclick={() => {
					void startSignIn().then((address) => {
						window.location.href = address;
					});
				}}
			>
				Sign in
			</button>
		</div>
	</div>
</div>
