<script lang="ts">
	import { api, type Provider, type ProviderChange } from '$lib/api';
	import FieldError from '$lib/components/providers/FieldError.svelte';
	import OptionsField from '$lib/components/providers/OptionsField.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import Panel from '$lib/components/ui/Panel.svelte';
	import {
		failureMessage,
		fieldErrors,
		optionsPatch,
		optionsText,
		parseOptions,
		parseScopes
	} from '$lib/providers/form';

	interface Props {
		provider: Provider;
		etag: string | undefined;
		onsaved: () => Promise<void>;
	}

	let { provider, etag, onsaved }: Props = $props();

	let busy = $state(false);
	let saved = $state(false);
	let failure = $state<string | null>(null);
	let errors = $state<Record<string, string>>({});

	let name = $derived(provider.name);
	let clientId = $derived(provider.client_id ?? '');
	let clientSecret = $state('');
	let scopes = $derived((provider.scopes ?? []).join(' '));
	let options = $derived(optionsText(provider.options));

	const takesClient = $derived(provider.preset === 'telegram' || provider.preset === 'oidc');

	/** Answers the merge patch of the fields that the administrator changed. */
	function changes(): ProviderChange | null {
		const change: ProviderChange = {};

		if (name.trim() !== provider.name) {
			change.name = name.trim();
		}

		if (takesClient) {
			if (clientId.trim() !== (provider.client_id ?? '')) {
				change.client_id = clientId.trim();
			}

			if (clientSecret !== '') {
				change.client_secret = clientSecret;
			}

			const parsed = parseOptions(options);
			if (parsed.error) {
				errors = { '/options': parsed.error };

				return null;
			}

			change.options = optionsPatch(provider.options, parsed.options ?? {});
		}

		if (provider.preset === 'oidc' && scopes.trim() !== (provider.scopes ?? []).join(' ')) {
			change.scopes = parseScopes(scopes);
		}

		return change;
	}

	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		errors = {};
		failure = null;
		saved = false;

		const change = changes();
		if (change === null) {
			return;
		}

		busy = true;

		try {
			await api.providers.change(provider.id, change, etag);
			clientSecret = '';
			await onsaved();
			saved = true;
		} catch (error) {
			errors = fieldErrors(error);
			failure = failureMessage(error, 'The provider stays as it was. Try again.');
		} finally {
			busy = false;
		}
	}
</script>

{#snippet fixed(label: string, value: string)}
	<div class="flex flex-col gap-1">
		<span class="label">{label}</span>
		<input class="input input-sm w-full font-mono" readonly {value} aria-label={label} />
	</div>
{/snippet}

<form onsubmit={submit}>
	<Panel>
		{#if failure}
			<Alert kind="error"><span class="text-sm">{failure}</span></Alert>
		{:else if saved}
			<Alert kind="success"
				><span class="text-sm">Saved. The next sign in reads the change.</span></Alert
			>
		{/if}

		<label class="flex flex-col gap-1">
			<span class="label">Name</span>
			<input class="input input-sm w-full" bind:value={name} disabled={busy} required />
			<span class="label whitespace-normal">The label of the button on the sign in page.</span>
			<FieldError {errors} pointer="/name" />
		</label>

		{#if takesClient}
			<div class="mt-2 grid gap-3 sm:grid-cols-2">
				{@render fixed('Issuer', provider.issuer ?? '')}
				{@render fixed('User identifier claim', provider.user_id_key ?? '')}
			</div>

			<div class="mt-2 grid gap-3 sm:grid-cols-2">
				<label class="flex flex-col gap-1">
					<span class="label">Client identifier</span>
					<input
						class="input input-sm w-full font-mono"
						bind:value={clientId}
						disabled={busy}
						required
					/>
					<FieldError {errors} pointer="/client_id" />
				</label>
				<label class="flex flex-col gap-1">
					<span class="label">Client secret</span>
					<input
						class="input input-sm w-full font-mono"
						type="password"
						autocomplete="off"
						bind:value={clientSecret}
						disabled={busy}
						placeholder={provider.client_secret_set ? 'Set. Leave empty to keep it.' : ''}
					/>
					<FieldError {errors} pointer="/client_secret" />
				</label>
			</div>

			{#if provider.preset === 'oidc'}
				<label class="mt-2 flex flex-col gap-1">
					<span class="label">Scopes</span>
					<input class="input input-sm w-full font-mono" bind:value={scopes} disabled={busy} />
					<FieldError {errors} pointer="/scopes" />
				</label>
			{:else}
				<div class="mt-2">
					{@render fixed('Scopes', 'openid profile')}
				</div>
			{/if}

			<OptionsField
				bind:value={options}
				{errors}
				rows={5}
				disabled={busy}
				hint="Further options of the connector of Dex, in YAML. A removed key goes away."
			/>
		{/if}

		<div class="mt-3 flex justify-end">
			<button type="submit" class="btn btn-primary btn-sm" disabled={busy}>Save</button>
		</div>
	</Panel>
</form>
