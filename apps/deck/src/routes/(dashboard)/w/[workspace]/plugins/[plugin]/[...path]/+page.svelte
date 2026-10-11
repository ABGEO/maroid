<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import type { PluginHost } from '@maroid/plugin-sdk';

	import type { PageProps } from './$types';
	import { createPluginClient } from '$lib/api';
	import { pluginCan } from '$lib/permissions';
	import { uiOf } from '$lib/plugins/capabilities';
	import { knownPlugin } from '$lib/state/plugins.svelte';
	import { pageTitle } from '$lib/state/title.svelte';
	import { displayName, userState } from '$lib/state/user.svelte';

	let { data }: PageProps = $props();

	function pluginPath(path: string): string {
		return resolve('/(dashboard)/w/[workspace]/plugins/[plugin]/[...path]', {
			workspace: data.workspace.id,
			plugin: data.pluginId,
			path: path.replace(/^\//, '')
		});
	}
	let target: HTMLElement | undefined = $state();

	const api = $derived(createPluginClient(data.workspace.id, data.pluginId));
	const firstEntry = $derived(uiOf(knownPlugin(data.pluginId))?.routes[0]);

	const host: PluginHost = {
		get user() {
			const user = userState.user;

			return user ? { name: displayName(user), picture: user.picture ?? '' } : null;
		},
		get api() {
			return api;
		},
		can: (permission) => pluginCan(data.workspace, data.pluginId, permission),
		href: (path) => pluginPath(path),
		navigate: (path) => {
			void goto(
				resolve('/(dashboard)/w/[workspace]/plugins/[plugin]/[...path]', {
					workspace: data.workspace.id,
					plugin: data.pluginId,
					path: path.replace(/^\//, '')
				})
			);
		},
		setTitle: (title) => {
			pageTitle.value = title;
		}
	};

	$effect(() => {
		const route = data.route;
		if (!target || route === null) {
			return;
		}

		const unmount = route.mount(target, { host, params: route.params });

		return () => {
			unmount();
			pageTitle.value = null;
		};
	});
</script>

{#if data.route === null}
	<header class="page-header">
		<div>
			<h1 class="page-title">Not found</h1>
			<p class="page-lead">This plugin has no page at this address.</p>
		</div>
	</header>
	{#if firstEntry}
		<a
			class="btn btn-sm"
			href={resolve('/(dashboard)/w/[workspace]/plugins/[plugin]/[...path]', {
				workspace: data.workspace.id,
				plugin: data.pluginId,
				path: firstEntry.path.replace(/^\//, '')
			})}
		>
			{firstEntry.label}
		</a>
	{/if}
{/if}

<div bind:this={target}></div>
