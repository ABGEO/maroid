<script lang="ts">
	import { api, createWriteIntent, type InvitedUser } from '$lib/api';
	import Alert from '$lib/components/ui/Alert.svelte';
	import Panel from '$lib/components/ui/Panel.svelte';
	import PluginPicker from '$lib/components/users/PluginPicker.svelte';
	import { pluginState } from '$lib/state/plugins.svelte';
	import { userFailureMessage } from '$lib/users/failure';

	interface Props {
		oncreated: (created: InvitedUser) => Promise<void>;
	}

	let { oncreated }: Props = $props();

	const intent = createWriteIntent();

	let firstName = $state('');
	let lastName = $state('');
	let administrator = $state(false);
	let allowedPlugins = $state<string[]>([]);
	let busy = $state(false);
	let failure = $state<string | null>(null);

	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		if (busy) {
			return;
		}

		const input = {
			first_name: firstName.trim() || undefined,
			last_name: lastName.trim() || undefined,
			is_administrator: administrator,
			allowed_plugins: administrator ? [] : allowedPlugins
		};

		busy = true;
		failure = null;

		try {
			const created = await api.users.create(input, intent.keyFor(input));
			if (created === null) {
				return;
			}

			intent.settle();
			firstName = '';
			lastName = '';
			administrator = false;
			allowedPlugins = [];
			await oncreated(created);
		} catch (error) {
			failure = userFailureMessage(error, 'The hub created nobody. Try again.');
		} finally {
			busy = false;
		}
	}
</script>

<form onsubmit={submit}>
	<Panel>
		{#if failure}
			<Alert kind="error"><span class="text-sm">{failure}</span></Alert>
		{/if}

		<div class="grid gap-3 sm:grid-cols-2">
			<label class="flex flex-col gap-1">
				<span class="label">First name</span>
				<input class="input input-sm w-full" bind:value={firstName} />
			</label>
			<label class="flex flex-col gap-1">
				<span class="label">Last name</span>
				<input class="input input-sm w-full" bind:value={lastName} />
			</label>
		</div>

		<div class="mt-2 grid gap-3 sm:grid-cols-2">
			{#if pluginState.plugins.length > 0}
				<PluginPicker bind:selected={allowedPlugins} {administrator} />
			{/if}

			<label class="flex cursor-pointer flex-wrap items-center gap-2 sm:pt-6">
				<input type="checkbox" class="toggle toggle-sm" bind:checked={administrator} />
				<span class="text-sm">Administrator</span>
				<span class="label whitespace-normal">
					Manages the users and the plugins of every workspace.
				</span>
			</label>
		</div>

		<div class="mt-3 flex justify-end">
			<button type="submit" class="btn btn-primary btn-sm" disabled={busy}>Create</button>
		</div>
	</Panel>
</form>
