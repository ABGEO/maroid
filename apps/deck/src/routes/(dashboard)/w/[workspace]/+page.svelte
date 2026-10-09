<script lang="ts">
	import { resolve } from '$app/paths';

	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<div class="max-w-2xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">{data.workspace.name}</h1>
			<p class="page-lead">The records of your plugins in this workspace.</p>
		</div>
	</header>

	<ul class="list panel">
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
