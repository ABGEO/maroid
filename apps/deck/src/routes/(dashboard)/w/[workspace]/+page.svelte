<script lang="ts">
	import { resolve } from '$app/paths';

	import WidgetGrid from '$lib/components/widgets/WidgetGrid.svelte';

	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<div class="max-w-5xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">{data.workspace.name}</h1>
			<p class="page-lead">The records of your plugins in this workspace.</p>
		</div>
	</header>

	<WidgetGrid workspaceId={data.workspace.id} />

	<ul class="list panel max-w-2xl">
		<li>
			<a
				class="list-row hover:bg-base-200"
				href={resolve('/(dashboard)/w/[workspace]/members', { workspace: data.workspace.id })}
			>
				<span class="list-col-grow">
					<span class="block font-medium">Members</span>
					<span class="meta block">The people who share this workspace.</span>
				</span>
				<span class="text-base-content/40" aria-hidden="true">&rarr;</span>
			</a>
		</li>
		<li>
			<a
				class="list-row hover:bg-base-200"
				href={resolve('/(dashboard)/w/[workspace]/plugins', { workspace: data.workspace.id })}
			>
				<span class="list-col-grow">
					<span class="block font-medium">Plugins</span>
					<span class="meta block">The plugins this workspace uses.</span>
				</span>
				<span class="text-base-content/40" aria-hidden="true">&rarr;</span>
			</a>
		</li>
		<li>
			<a
				class="list-row hover:bg-base-200"
				href={resolve('/(dashboard)/w/[workspace]/settings/notifications', {
					workspace: data.workspace.id
				})}
			>
				<span class="list-col-grow">
					<span class="block font-medium">Notifications</span>
					<span class="meta block">Where the notifications of this workspace go.</span>
				</span>
				<span class="text-base-content/40" aria-hidden="true">&rarr;</span>
			</a>
		</li>
		<li>
			<a
				class="list-row hover:bg-base-200"
				href={resolve('/(dashboard)/w/[workspace]/settings', { workspace: data.workspace.id })}
			>
				<span class="list-col-grow">
					<span class="block font-medium">Settings</span>
					<span class="meta block">The name of this workspace.</span>
				</span>
				<span class="text-base-content/40" aria-hidden="true">&rarr;</span>
			</a>
		</li>
	</ul>
</div>
