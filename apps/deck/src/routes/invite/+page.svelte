<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { buildInviteUrl } from '$lib/api/client';

	onMount(() => {
		const token = page.url.searchParams.get('token');

		if (!token) {
			void goto(resolve('/auth/callback?error=invitation_invalid'));
			return;
		}

		window.location.href = buildInviteUrl(token);
	});
</script>
