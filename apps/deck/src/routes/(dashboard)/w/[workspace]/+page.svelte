<script lang="ts">
	import { resolve } from '$app/paths';

	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<div class="max-w-2xl">
	<h1 class="font-display text-[42px] leading-[1.05] tracking-tight">{data.workspace.name}</h1>
	<p class="text-base-content/60 mt-3 text-sm">The records of your plugins in this workspace.</p>

	<div class="mt-8 flex gap-2">
		<a
			class="btn btn-sm"
			href={resolve('/(dashboard)/w/[workspace]/members', { workspace: data.workspace.id })}
		>
			Members
		</a>
		<a
			class="btn btn-sm"
			href={resolve('/(dashboard)/w/[workspace]/plugins', { workspace: data.workspace.id })}
		>
			Plugins
		</a>
		<a
			class="btn btn-sm"
			href={resolve('/(dashboard)/w/[workspace]/settings', { workspace: data.workspace.id })}
		>
			Settings
		</a>
	</div>
</div>
