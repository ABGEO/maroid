<script lang="ts">
	import AuthGate from '$lib/components/AuthGate.svelte';
	import AppShell from '$lib/components/layout/AppShell.svelte';

	let { children } = $props();
</script>

<AuthGate>
	<AppShell>
		{@render children()}
	</AppShell>
</AuthGate>
