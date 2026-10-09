<script lang="ts">
	import Alert from '$lib/components/ui/Alert.svelte';
	import Panel from '$lib/components/ui/Panel.svelte';

	interface Names {
		first_name: string;
		last_name: string;
	}

	interface Props {
		record: { first_name?: string; last_name?: string } | null;
		disabled?: boolean;
		own?: boolean;
		notice?: string | null;
		failure?: string | null;
		onsave: (names: Names) => Promise<void>;
	}

	let {
		record,
		disabled = false,
		own = false,
		notice = null,
		failure = null,
		onsave
	}: Props = $props();

	let firstName = $derived(record?.first_name ?? '');
	let lastName = $derived(record?.last_name ?? '');
	let saving = $state(false);

	const changed = $derived(
		record !== null &&
			(firstName.trim() !== (record.first_name ?? '') ||
				lastName.trim() !== (record.last_name ?? ''))
	);

	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		saving = true;

		try {
			await onsave({ first_name: firstName.trim(), last_name: lastName.trim() });
		} finally {
			saving = false;
		}
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

		<div class="grid gap-3 sm:grid-cols-2">
			<label class="flex flex-col gap-1">
				<span class="label">First name</span>
				<input
					class="input input-sm w-full"
					autocomplete={own ? 'given-name' : 'off'}
					bind:value={firstName}
					disabled={disabled || record === null || saving}
				/>
			</label>
			<label class="flex flex-col gap-1">
				<span class="label">Last name</span>
				<input
					class="input input-sm w-full"
					autocomplete={own ? 'family-name' : 'off'}
					bind:value={lastName}
					disabled={disabled || record === null || saving}
				/>
			</label>
		</div>

		<div class="mt-3 flex justify-end">
			<button
				type="submit"
				class="btn btn-primary btn-sm"
				disabled={disabled || saving || !changed}
			>
				Save
			</button>
		</div>
	</Panel>
</form>
