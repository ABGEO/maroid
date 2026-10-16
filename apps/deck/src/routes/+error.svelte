<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';

	import { documentTitle } from '$lib/navigation/title';

	const notFound = $derived(page.status === 404);
	const title = $derived(notFound ? 'Not found' : 'Something went wrong');
	// The router answers an address that matches no route with the bare title, which
	// tells the person nothing the heading does not.
	const generic = $derived(
		!page.error?.message || page.error.message.toLowerCase() === title.toLowerCase()
	);
	const message = $derived(
		generic
			? notFound
				? 'Nothing lives at this address.'
				: 'Reload the page.'
			: (page.error?.message ?? '')
	);
</script>

<svelte:head><title>{documentTitle(title)}</title></svelte:head>

<div class="bg-base-200 flex min-h-screen items-center justify-center p-4">
	<div class="card panel w-full max-w-sm">
		<div class="card-body gap-6 text-center">
			<div class="flex flex-col gap-2">
				<p class="meta font-mono">{page.status}</p>
				<h1 class="page-title">{title}</h1>
				<p class="page-lead mx-auto">{message}</p>
			</div>

			<div class="flex flex-col gap-2">
				<a class="btn btn-primary w-full" href={resolve('/')}>Go to the home page</a>
				{#if !notFound}
					<button class="btn btn-ghost w-full" onclick={() => window.location.reload()}>
						Reload
					</button>
				{/if}
			</div>
		</div>
	</div>
</div>
