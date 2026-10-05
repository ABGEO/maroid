<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { enabledState, loadEnabled } from '$lib/state/plugins.svelte';
	import { userState } from '$lib/state/user.svelte';
	import { actingWorkspaceId, isMember, workspaceState } from '$lib/state/workspaces.svelte';
	import { uiOf } from '$lib/plugins/capabilities';

	function isActive(href: string) {
		return page.url.pathname === href;
	}

	function groupIsOpen(hrefs: string[]) {
		return hrefs.some((h) => isActive(h));
	}

	const workspaceId = $derived(actingWorkspaceId(page.params.workspace));

	function routeHref(workspace: string, pluginId: string, path: string) {
		return resolve('/(dashboard)/w/[workspace]/plugins/[plugin]/[...path]', {
			workspace,
			plugin: pluginId,
			path: path.replace(/^\//, '')
		});
	}

	$effect(() => {
		if (workspaceId !== null) {
			void loadEnabled(workspaceId);
		}
	});

	// An administrator manages a workspace of another person and reads none of its
	// records, so the navigation offers no page of a plugin there.
	const uiPlugins = $derived(
		workspaceId === null || !isMember(workspaceId) || enabledState.workspaceId !== workspaceId
			? []
			: enabledState.plugins
					.map((plugin) => ({ id: plugin.id, manifest: uiOf(plugin) }))
					.filter((entry) => entry.manifest !== undefined)
	);

	const pluginsLoading = $derived(
		enabledState.status === 'idle' || enabledState.status === 'loading'
	);

	const isAdministrator = $derived(userState.user?.is_administrator === true);

	const hasNoWorkspace = $derived(
		workspaceState.status === 'ready' && workspaceState.workspaces.length === 0
	);

	const pluginsGroupIsOpen = $derived(
		page.route.id?.startsWith('/(dashboard)/w/[workspace]/plugins') === true
	);

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
						<path d="M6 2h12l-1 7H7z" />
						<path d="M12 9v8" />
						<circle cx="12" cy="20" r="2" />
					</svg>
					<span class="is-drawer-close:hidden">Alerts</span>
				</a>
			</li>
			<!-- @todo: move to a dedicated component -->
			{#if !hasNoWorkspace}
				<li>
					<details open={pluginsGroupIsOpen}>
						<summary>
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
							<span class="is-drawer-close:hidden">Plugins</span>
						</summary>
						<ul>
							{#if workspaceId !== null}
								<li>
									<a
										href={resolve('/(dashboard)/w/[workspace]/plugins', { workspace: workspaceId })}
										class:menu-active={page.route.id === '/(dashboard)/w/[workspace]/plugins'}
									>
										All plugins
									</a>
								</li>
							{/if}

							{#if pluginsLoading}
								{#each [0, 1, 2] as i (i)}
									<li>
										<div class="flex items-center gap-2 px-3 py-1.5">
											<span class="skeleton h-4 w-4 shrink-0 rounded" aria-hidden="true"></span>
											<span class="skeleton h-4 w-24" aria-hidden="true"></span>
										</div>
									</li>
								{/each}
							{:else if enabledState.status === 'ready' && workspaceId !== null}
								{#each uiPlugins as plugin (plugin.id)}
									<li>
										<details
											open={groupIsOpen(
												plugin.manifest!.routes.map((r) =>
													routeHref(workspaceId, plugin.id, r.path)
												)
											)}
										>
											<summary>
												<span
													class="grid h-4 w-4 shrink-0 place-items-center rounded font-mono text-[10px] font-semibold"
													style="background:oklch(92% 0.04 {nameToHue(
														plugin.manifest!.name
													)});color:oklch(40% 0.1 {nameToHue(plugin.manifest!.name)})"
												>
													{letterFromName(plugin.manifest!.name)}
												</span>
												<span class="is-drawer-close:hidden">{plugin.manifest!.name}</span>
											</summary>
											<ul>
												{#each plugin.manifest!.routes as route (route.path)}
													<li>
														<a
															href={resolve(
																'/(dashboard)/w/[workspace]/plugins/[plugin]/[...path]',
																{
																	workspace: workspaceId,
																	plugin: plugin.id,
																	path: route.path.replace(/^\//, '')
																}
															)}
															class:menu-active={isActive(
																routeHref(workspaceId, plugin.id, route.path)
															)}
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
						</ul>
					</details>
				</li>
			{/if}
			{#if isAdministrator}
				<li>
					<details open={page.route.id?.startsWith('/(dashboard)/admin') === true}>
						<summary>
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
								<path d="M12 3l8 3v6c0 4.5-3.4 8.3-8 9-4.6-.7-8-4.5-8-9V6z" />
							</svg>
							<span class="is-drawer-close:hidden">Administration</span>
						</summary>
						<ul>
							<li>
								<a
									href={resolve('/admin/users')}
									class:menu-active={page.route.id?.startsWith('/(dashboard)/admin/users') === true}
								>
									Users
								</a>
							</li>
							<li>
								<a
									href={resolve('/admin/workspaces')}
									class:menu-active={isActive(resolve('/admin/workspaces'))}
								>
									Workspaces
								</a>
							</li>
							<li>
								<a
									href={resolve('/admin/plugins')}
									class:menu-active={isActive(resolve('/admin/plugins'))}
								>
									Plugins
								</a>
							</li>
						</ul>
					</details>
				</li>
			{/if}
		</ul>
	</div>
</aside>
