<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { startInvitationRedemption } from '$lib/api/client';

	onMount(() => {
		const token = page.url.searchParams.get('token');

		if (!token) {
			void goto(resolve('/auth/callback?error=invitation_invalid'));
			return;
		}

		void startInvitationRedemption(token)
			.then((address) => {
				window.location.href = address;
			})
			.catch(() => goto(resolve('/auth/callback?error=invitation_invalid')));
	});
</script>
