<script lang="ts">
	import { onMount } from 'svelte';

	import { ApiError, PROBLEM_TYPE, api, type Tagged, type UserRecord } from '$lib/api';
	import NameForm from '$lib/components/users/NameForm.svelte';
	import { loadUser } from '$lib/state/user.svelte';

	let record = $state<UserRecord | null>(null);
	let etag = $state<string | undefined>(undefined);
	let notice = $state<string | null>(null);
	let failure = $state<string | null>(null);

	function show(tagged: Tagged<UserRecord>): void {
		record = tagged.value;
		etag = tagged.etag;
	}

	function failureMessage(error: unknown): string {
		if (error instanceof ApiError && error.is(PROBLEM_TYPE.preconditionFailed)) {
			return 'Your name changed in another tab. Reload the page.';
		}

		return 'Your name stays as it was. Try again.';
	}

	async function save(names: { first_name: string; last_name: string }): Promise<void> {
		notice = null;
		failure = null;

		try {
			const tagged = await api.users.changeSelf(names, etag);
			if (tagged !== null) {
				show(tagged);
			}

			await loadUser();
			notice = 'Your name is saved.';
		} catch (error) {
			failure = failureMessage(error);
		}
	}

	onMount(async () => {
		try {
			const tagged = await api.users.self();
			if (tagged !== null) {
				show(tagged);
			}
		} catch (error) {
			console.error('Failed to load the user record', error);
			failure = 'Failed to load your name. Reload the page.';
		}
	});
</script>

<NameForm {record} own {notice} {failure} onsave={save} />
