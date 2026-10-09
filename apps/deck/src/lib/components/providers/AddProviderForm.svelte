<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import { api, createWriteIntent, type NewProvider, type ProviderPreset } from '$lib/api';
	import FieldError from '$lib/components/providers/FieldError.svelte';
	import OptionsField from '$lib/components/providers/OptionsField.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import Panel from '$lib/components/ui/Panel.svelte';
	import { failureMessage, fieldErrors, parseOptions, parseScopes } from '$lib/providers/form';

	interface Props {
		taken: Set<string>;
	}

	let { taken }: Props = $props();

	const PRESETS: { value: ProviderPreset; label: string; once?: string }[] = [
		{ value: 'oidc', label: 'OpenID Connect' },
		{ value: 'telegram', label: 'Telegram', once: 'telegram' },
		{ value: 'local', label: 'Email', once: 'local' }
	];

	const intent = createWriteIntent();

	let busy = $state(false);
	let failure = $state<string | null>(null);
	let errors = $state<Record<string, string>>({});

	let preset = $state<ProviderPreset>('oidc');
	let id = $state('');
	let name = $state('');
	let issuer = $state('');
	let clientId = $state('');
	let clientSecret = $state('');
	let userIdKey = $state('');
	let scopes = $state('');
	let options = $state('');

	function input(): NewProvider | null {
		if (preset === 'local') {
			return { preset };
		}

		const parsed = parseOptions(options);
		if (parsed.error) {
			errors = { '/options': parsed.error };

			return null;
		}

		const provider: NewProvider = {
			preset,
			client_secret: clientSecret,
			options: Object.keys(parsed.options ?? {}).length > 0 ? parsed.options : undefined
		};

		if (clientId.trim() !== '') {
			provider.client_id = clientId.trim();
		}

		if (preset === 'oidc') {
			provider.id = id.trim();
			provider.name = name.trim();
			provider.issuer = issuer.trim();

			if (userIdKey.trim() !== '') {
				provider.user_id_key = userIdKey.trim();
			}

			if (scopes.trim() !== '') {
				provider.scopes = parseScopes(scopes);
			}
		}

		return provider;
	}

	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		errors = {};
		failure = null;

		const provider = input();
		if (provider === null) {
			return;
		}

		busy = true;

		try {
			const created = await api.providers.create(provider, intent.keyFor(provider));
			intent.settle();

			if (created !== null) {
				await goto(resolve('/(dashboard)/admin/providers/[provider]', { provider: created.id }));
			}
		} catch (error) {
			errors = fieldErrors(error);
			failure = failureMessage(error, 'Dex holds no new provider. Try again.');
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

		<div class="join" role="radiogroup" aria-label="The kind of the provider">
			{#each PRESETS as one (one.value)}
				<input
					class="join-item btn btn-sm"
					type="radio"
					name="preset"
					aria-label={one.label}
					value={one.value}
					bind:group={preset}
					disabled={busy || (one.once !== undefined && taken.has(one.once))}
				/>
			{/each}
		</div>

		{#if preset === 'local'}
			<span class="label mt-2 whitespace-normal">
				Adds "Log in with Email" to the sign in page. A person signs in with an email address and a
				password that an administrator gives on the page of the user.
			</span>
		{:else}
			{#if preset === 'oidc'}
				<div class="mt-2 grid gap-3 sm:grid-cols-2">
					<label class="flex flex-col gap-1">
						<span class="label">Identifier</span>
						<input
							class="input input-sm w-full font-mono"
							bind:value={id}
							placeholder="oidc-provider"
							required
						/>
						<span class="label whitespace-normal">It never changes after the provider exists.</span>
						<FieldError {errors} pointer="/id" />
					</label>
					<label class="flex flex-col gap-1">
						<span class="label">Name</span>
						<input class="input input-sm w-full" bind:value={name} required />
						<span class="label whitespace-normal">The label of the button on the sign in page.</span
						>
						<FieldError {errors} pointer="/name" />
					</label>
				</div>

				<label class="mt-2 flex flex-col gap-1">
					<span class="label">Issuer</span>
					<input
						class="input input-sm w-full font-mono"
						type="url"
						bind:value={issuer}
						placeholder="https://auth.example.com"
						required
					/>
					<span class="label whitespace-normal">It never changes after the provider exists.</span>
					<FieldError {errors} pointer="/issuer" />
				</label>
			{/if}

			<div class="mt-2 grid gap-3 sm:grid-cols-2">
				<label class="flex flex-col gap-1">
					<span class="label">Client identifier</span>
					<input
						class="input input-sm w-full font-mono"
						bind:value={clientId}
						placeholder={preset === 'telegram' ? 'The bot of this hub' : ''}
						required={preset === 'oidc'}
					/>
					{#if preset === 'telegram'}
						<span class="label whitespace-normal">Leave empty to use the bot of this hub.</span>
					{/if}
					<FieldError {errors} pointer="/client_id" />
				</label>
				<label class="flex flex-col gap-1">
					<span class="label">Client secret</span>
					<input
						class="input input-sm w-full font-mono"
						type="password"
						autocomplete="off"
						bind:value={clientSecret}
						required
					/>
					<FieldError {errors} pointer="/client_secret" />
				</label>
			</div>

			{#if preset === 'oidc'}
				<div class="mt-2 grid gap-3 sm:grid-cols-2">
					<label class="flex flex-col gap-1">
						<span class="label">User identifier claim</span>
						<input
							class="input input-sm w-full font-mono"
							bind:value={userIdKey}
							placeholder="sub"
						/>
						<span class="label whitespace-normal">It never changes after the provider exists.</span>
						<FieldError {errors} pointer="/user_id_key" />
					</label>
					<label class="flex flex-col gap-1">
						<span class="label">Scopes</span>
						<input
							class="input input-sm w-full font-mono"
							bind:value={scopes}
							placeholder="openid profile email"
						/>
						<FieldError {errors} pointer="/scopes" />
					</label>
				</div>
			{/if}

			<OptionsField
				bind:value={options}
				{errors}
				placeholder="getUserInfo: true"
				hint="Further options of the connector of Dex, in YAML. A field above sets its own key."
			/>
		{/if}

		<div class="mt-3 flex justify-end">
			<button type="submit" class="btn btn-primary btn-sm" disabled={busy}>Add</button>
		</div>
	</Panel>
</form>
