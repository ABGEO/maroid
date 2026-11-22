<script lang="ts">
	import { ApiError, PROBLEM_TYPE, api, type Tagged, type UserRecord } from '$lib/api';
	import Alert from '$lib/components/ui/Alert.svelte';
	import Panel from '$lib/components/ui/Panel.svelte';
	import { loadUser } from '$lib/state/user.svelte';

	interface Props {
		self: Tagged<UserRecord> | null;
		onchanged: (tagged: Tagged<UserRecord>) => void;
	}

	let { self, onchanged }: Props = $props();

	const ZONE_POINTER = '/time_zone';
	const zones = ['UTC', ...Intl.supportedValuesOf('timeZone').filter((name) => name !== 'UTC')];
	const browserZone = Intl.DateTimeFormat().resolvedOptions().timeZone;

	const saved = $derived(self?.value.time_zone ?? '');
	let zone = $derived(saved);
	let saving = $state(false);
	let notice = $state<string | null>(null);
	let failure = $state<string | null>(null);
	let fieldFailure = $state<string | null>(null);

	const changed = $derived(self !== null && zone.trim() !== '' && zone.trim() !== saved);

	function showFailure(error: unknown): void {
		if (error instanceof ApiError && error.fields.some((one) => one.pointer === ZONE_POINTER)) {
			fieldFailure = 'Pick a zone from the list, such as Asia/Tbilisi.';
		} else if (error instanceof ApiError && error.is(PROBLEM_TYPE.preconditionFailed)) {
			failure = 'Your profile changed in another tab. Reload the page.';
		} else {
			failure = 'Your time zone stays as it was. Try again.';
		}
	}

	async function write(value: string, done: string): Promise<void> {
		notice = null;
		failure = null;
		fieldFailure = null;
		saving = true;

		try {
			const tagged = await api.users.changeSelf({ time_zone: value }, self?.etag);
			if (tagged !== null) {
				onchanged(tagged);
			}

			await loadUser();
			notice = done;
		} catch (error) {
			showFailure(error);
		} finally {
			saving = false;
		}
	}

	function submit(event: SubmitEvent): void {
		event.preventDefault();
		void write(zone.trim(), 'Your time zone is saved.');
	}
</script>

<form onsubmit={submit}>
	<Panel>
		{#if notice}
			<Alert kind="success"><span class="text-sm">{notice}</span></Alert>
		{/if}
		{#if failure}
			<Alert kind="error"><span class="text-sm">{failure}</span></Alert>
		{/if}

		<label class="flex flex-col gap-1">
			<span class="label">Time zone</span>
			<input
				class="input input-sm w-full"
				class:input-error={fieldFailure !== null}
				list="time-zones"
				autocomplete="off"
				spellcheck="false"
				placeholder={browserZone}
				bind:value={zone}
				disabled={self === null || saving}
				aria-invalid={fieldFailure !== null}
			/>
			<datalist id="time-zones">
				{#each zones as name (name)}
					<option value={name}></option>
				{/each}
			</datalist>
			{#if fieldFailure}
				<span class="text-error text-xs">{fieldFailure}</span>
			{:else if saved === ''}
				<span class="meta text-xs">
					No zone is set, so the shell shows each time in the zone of this browser,
					{browserZone}. A job of a plugin computes in UTC.
				</span>
			{/if}
		</label>

		<div class="mt-3 flex justify-end gap-2">
			<button
				type="button"
				class="btn btn-ghost btn-sm"
				disabled={self === null || saving || saved === ''}
				onclick={() => void write('', 'Your time zone is cleared.')}
			>
				Clear
			</button>
			<button type="submit" class="btn btn-primary btn-sm" disabled={saving || !changed}>
				Save
			</button>
		</div>
	</Panel>
</form>
