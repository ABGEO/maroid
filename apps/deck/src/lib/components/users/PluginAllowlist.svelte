<script lang="ts">
	import { api, createWriteIntent, type UserRecord } from '$lib/api';
	import PluginSummary from '$lib/components/plugins/PluginSummary.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import Panel from '$lib/components/ui/Panel.svelte';
	import { displayNameOf } from '$lib/plugins/capabilities';
	import { pluginState } from '$lib/state/plugins.svelte';
	import { userFailureMessage } from '$lib/users/failure';

	interface Props {
		user: UserRecord;
		name: string;
		isSelf: boolean;
		allowed: string[];
	}

	let { user, name, isSelf, allowed = $bindable() }: Props = $props();

	const intent = createWriteIntent();

	let busy = $state(false);
	let failure = $state<string | null>(null);

	async function toggle(pluginId: string, input: HTMLInputElement): Promise<void> {
		const on = input.checked;

		busy = true;
		failure = null;

		try {
			if (on) {
				await api.users.allowPlugin(user.id, pluginId, intent.keyFor({ plugin_id: pluginId }));
				intent.settle();
				allowed = [...allowed, pluginId];
			} else {
				await api.users.disallowPlugin(user.id, pluginId);
				allowed = allowed.filter((id) => id !== pluginId);
			}
		} catch (error) {
			failure = userFailureMessage(error, 'The allowlist stays as it was. Try again.');
			input.checked = !on;
		} finally {
			busy = false;
		}
	}
</script>

<Panel>
	{#if pluginState.plugins.length === 0}
		<span class="label whitespace-normal">The hub loaded no plugin.</span>
	{:else}
		{#if failure}
			<Alert kind="error"><span class="text-sm">{failure}</span></Alert>
		{/if}
		{#if user.is_administrator}
			<Alert kind="info">
				<span class="text-sm">
					{isSelf ? 'You are' : `${name} is`} an administrator, so
					{isSelf ? 'you' : 'they'} already reach every plugin. The allowlist applies again when the mark
					goes.
				</span>
			</Alert>
		{:else}
			<span class="label whitespace-normal">
				The plugins that {name} can turn on in a workspace. A removal keeps every plugin that a workspace
				already uses.
			</span>
		{/if}
		<ul class="list panel">
			{#each pluginState.plugins as plugin (plugin.id)}
				<li class="list-row flex items-center justify-between gap-3">
					<PluginSummary id={plugin.id} {plugin} />
					<input
						type="checkbox"
						class="toggle toggle-primary toggle-sm"
						aria-label="Allow {displayNameOf(plugin, plugin.id)} for {name}"
						checked={allowed.includes(plugin.id)}
						disabled={busy || user.is_administrator}
						onchange={(event) => toggle(plugin.id, event.currentTarget)}
					/>
				</li>
			{/each}
		</ul>
	{/if}
</Panel>
