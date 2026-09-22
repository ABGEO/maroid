<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import type { PluginHost } from '@maroid/plugin-sdk';

	import type { PageProps } from './$types';
	import { createPluginClient } from '$lib/api';
	import { displayName, userState } from '$lib/state/user.svelte';

	let { data }: PageProps = $props();

	function pluginPath(path: string): string {
		return resolve('/(dashboard)/plugins/[plugin]/[...path]', {
			plugin: data.pluginId,
			path: path.replace(/^\//, '')
		});
	}
	let target: HTMLElement | undefined = $state();

	const api = $derived(createPluginClient(data.pluginId));

	const host: PluginHost = {
		get user() {
			const user = userState.user;

			return user ? { name: displayName(user), picture: user.picture } : null;
		},
		get api() {
			return api;
		},
		href: (path) => pluginPath(path),
		navigate: (path) => {
			void goto(
				resolve('/(dashboard)/plugins/[plugin]/[...path]', {
					plugin: data.pluginId,
					path: path.replace(/^\//, '')
				})
			);
		}
	};

	$effect(() => {
		if (!target) {
			return;
		}

		return data.mount(target, { host });
	});
</script>

<div bind:this={target}></div>
