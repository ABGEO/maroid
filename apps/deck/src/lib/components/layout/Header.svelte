<script>
	import { resolve } from '$app/paths';

	import UserDropdown from '$lib/components/UserDropdown.svelte';
</script>

<header
	class="bg-base-100/90 border-base-300 fixed inset-x-0 top-0 z-50 h-14 border-b backdrop-blur"
>
	<div class="flex h-14 items-center gap-2 px-3">
		<label for="app-drawer" aria-label="toggle sidebar" class="btn btn-ghost btn-sm btn-square">
			<svg
				xmlns="http://www.w3.org/2000/svg"
				width="18"
				height="18"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
			>
				<path d="M3 6h18M3 12h18M3 18h18" />
			</svg>
		</label>

		<a href={resolve('/')} class="flex items-center gap-2 pr-3 pl-1">
			<div class="bg-base-300 relative h-9 w-9 overflow-hidden rounded-full">
				<img src="/maroid.png" alt="Maroid" class="h-full w-full object-cover" />
			</div>
			<div class="leading-tight">
				<div class="text-[15px] font-semibold tracking-tight">Maroid</div>
				<div class="text-base-content/50 -mt-0.5 font-mono text-[10px]">deck · v0.1.0</div>
			</div>
		</a>

		<div class="flex-1"></div>

		<UserDropdown />
	</div>
</header>
