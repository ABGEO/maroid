<script lang="ts">
	import { ApiError, PROBLEM_TYPE, api, type Tagged, type UserRecord } from '$lib/api';
	import NameForm from '$lib/components/users/NameForm.svelte';
	import { loadUser } from '$lib/state/user.svelte';

	interface Props {
		self: Tagged<UserRecord> | null;
		onchanged: (tagged: Tagged<UserRecord>) => void;
	}

	let { self, onchanged }: Props = $props();

	let notice = $state<string | null>(null);
	let failure = $state<string | null>(null);

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
			const tagged = await api.users.changeSelf(names, self?.etag);
			if (tagged !== null) {
				onchanged(tagged);
			}

			await loadUser();
			notice = 'Your name is saved.';
		} catch (error) {
			failure = failureMessage(error);
		}
	}
</script>

<NameForm record={self?.value ?? null} own {notice} {failure} onsave={save} />
