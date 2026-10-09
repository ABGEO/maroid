<script lang="ts">
	import type { UserRecord } from '$lib/api';

	interface Props {
		user: UserRecord;
		isSelf: boolean;
		showActive?: boolean;
	}

	let { user, isSelf, showActive = false }: Props = $props();
</script>

{#if isSelf}
	<span class="badge badge-ghost badge-sm">You</span>
{/if}
{#if user.is_administrator}
	<span class="badge badge-outline badge-sm">Administrator</span>
{/if}
{#if user.status === 'blocked'}
	<span class="badge badge-error badge-sm">Blocked</span>
{:else if showActive}
	<span class="badge badge-success badge-soft badge-sm">Active</span>
{/if}
