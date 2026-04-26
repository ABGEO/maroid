<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { pluginState } from '$lib/state/plugins.svelte';

	function isActive(href: string) {
		return page.url.pathname === href;
	}

	function groupIsOpen(hrefs: string[]) {
		return hrefs.some((h) => isActive(h));
	}

	function letterFromName(name: string) {
		return name.charAt(0).toUpperCase();
	}

	// Generate a hue value from a string using a hash function.
	// https://stackoverflow.com/a/15710692
	function nameToHue(name: string) {
		let hash = 5381;
		for (let i = 0; i < name.length; i++) {
			hash = ((hash << 5) + hash + name.charCodeAt(i)) | 0;
		}

		return Math.abs(hash) % 360;
	}
</script>

<aside
	class="border-base-300 bg-base-200 lg:bg-base-200/40 is-drawer-close:w-14 is-drawer-open:w-68 flex min-h-full shrink-0 flex-col border-r pt-14 pb-10 transition-[width] duration-200 ease-out"
>
	<div class="is-drawer-close:overflow-visible flex-1 overflow-y-auto">
		<ul class="menu w-full">
			<li>
				<a href={resolve('/')} class:menu-active={isActive(resolve('/'))}>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="16"
						height="16"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="1.75"
						class="shrink-0"
					>
						<rect x="3" y="3" width="7" height="9" rx="1" />
						<rect x="14" y="3" width="7" height="5" rx="1" />
						<rect x="14" y="12" width="7" height="9" rx="1" />
						<rect x="3" y="16" width="7" height="5" rx="1" />
					</svg>
					<span class="is-drawer-close:hidden">Overview</span>
				</a>
			</li>
			<li>
				<a href="#">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="16"
						height="16"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="1.75"
						class="shrink-0"
					>
						<path d="M4 4h6v6H4zM14 4h6v6h-6zM4 14h6v6H4zM14 14h6v6h-6z" />
					</svg>
					<span class="is-drawer-close:hidden">Services</span>
				</a>
			</li>
			<li>
				<a href="#">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="16"
						height="16"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="1.75"
						class="shrink-0"
					>
						<path d="M6 2h12l-1 7H7z" />
						<path d="M12 9v8" />
						<circle cx="12" cy="20" r="2" />
					</svg>
					<span class="is-drawer-close:hidden">Alerts</span>
				</a>
			</li>
			<!-- @todo: move to a dedicated component -->
			<li
				class="menu-title is-drawer-close:hidden text-base-content/45 font-mono text-[10px] tracking-widest uppercase"
			>
				Plugins
			</li>
			{#if pluginState.status === 'idle' || pluginState.status === 'loading'}
				{#each [0, 1, 2] as i (i)}
					<li>
						<div class="flex items-center gap-2 px-3 py-1.5">
							<span class="skeleton h-4 w-4 shrink-0 rounded" aria-hidden="true"></span>
							<span class="skeleton h-4 w-32" aria-hidden="true"></span>
						</div>
					</li>
				{/each}
			{:else if pluginState.status === 'ready'}
				{#each pluginState.plugins.filter((p) => p.ui) as plugin (plugin.id)}
					<li>
						<details
							open={groupIsOpen(plugin.ui!.routes.map((r) => `/plugin/${plugin.id}${r.path}`))}
						>
							<summary>
								<span
									class="grid h-4 w-4 shrink-0 place-items-center rounded font-mono text-[10px] font-semibold"
									style="background:oklch(92% 0.04 {nameToHue(
										plugin.ui!.name
									)});color:oklch(40% 0.1 {nameToHue(plugin.ui!.name)})"
								>
									{letterFromName(plugin.ui!.name)}
								</span>
								<span class="is-drawer-close:hidden">{plugin.ui!.name}</span>
							</summary>
							<ul>
								{#each plugin.ui!.routes as route (route.path)}
									<li>
										<a
											href={`/plugin/${plugin.id}${route.path}`}
											class:menu-active={isActive(`/plugin/${plugin.id}${route.path}`)}
										>
											{route.label}
										</a>
									</li>
								{/each}
							</ul>
						</details>
					</li>
				{/each}
			{/if}
			<div
				class="is-drawer-close:hidden border-base-300 mx-2 mt-2 rounded-md border border-dashed px-2 py-2"
			>
				<div class="text-base-content/50 font-mono text-[10px] leading-relaxed">
					Only plugins exposing a UI capability appear here.
				</div>
			</div>
		</ul>
	</div>
</aside>
