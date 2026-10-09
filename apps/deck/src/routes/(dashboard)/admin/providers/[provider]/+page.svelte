<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';

	import { ApiError, api, type Provider, type ProviderChange, type UserRef } from '$lib/api';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import {
		failureMessage,
		fieldErrors,
		optionsPatch,
		optionsText,
		parseOptions,
		parseScopes,
		presetLabel
	} from '$lib/providers/form';

	const providerId = $derived(page.params.provider ?? '');

	let status = $state<'loading' | 'ready' | 'missing' | 'error'>('loading');
	let provider = $state<Provider | null>(null);
	let etag = $state<string | undefined>(undefined);
	let busy = $state(false);
	let banner = $state<string | null>(null);
	let saved = $state(false);
	let errors = $state<Record<string, string>>({});
	let copied = $state(false);
	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();

	let name = $state('');
	let clientId = $state('');
	let clientSecret = $state('');
	let scopes = $state('');
	let options = $state('');

	const takesClient = $derived(provider?.preset === 'telegram' || provider?.preset === 'oidc');

	function show(record: Provider, tag: string | undefined): void {
		provider = record;
		etag = tag;
		name = record.name;
		clientId = record.client_id ?? '';
		clientSecret = '';
		scopes = (record.scopes ?? []).join(' ');
		options = optionsText(record.options);
	}

	async function read(): Promise<Provider | null> {
		const tagged = await api.providers.get(providerId);
		if (tagged === null) {
			return null;
		}

		show(tagged.value, tagged.etag);

		return tagged.value;
	}

	async function load(): Promise<void> {
		status = 'loading';

		try {
			if ((await read()) !== null) {
				status = 'ready';
			}
		} catch (error) {
			if (error instanceof ApiError && error.status === 404) {
				status = 'missing';

				return;
			}

			console.error('Failed to load the provider', error);
			status = 'error';
		}
	}

	/** Answers the merge patch of the fields that the administrator changed. */
	function changes(stored: Provider): ProviderChange | null {
		const change: ProviderChange = {};

		if (name.trim() !== stored.name) {
			change.name = name.trim();
		}

		if (takesClient) {
			if (clientId.trim() !== (stored.client_id ?? '')) {
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

			change.options = optionsPatch(stored.options, parsed.options ?? {});
		}

		if (stored.preset === 'oidc' && scopes.trim() !== (stored.scopes ?? []).join(' ')) {
			change.scopes = parseScopes(scopes);
		}

		return change;
	}

	async function save(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		if (provider === null) {
			return;
		}

		errors = {};
		banner = null;
		saved = false;

		const change = changes(provider);
		if (change === null) {
			return;
		}

		busy = true;

		try {
			await api.providers.change(providerId, change, etag);
			await read();
			saved = true;
		} catch (error) {
			errors = fieldErrors(error);
			banner = failureMessage(error, 'The provider stays as it was. Try again.');
		} finally {
			busy = false;
		}
	}

	function names(people: UserRef[]): string {
		return people
			.map((one) => [one.first_name, one.last_name].filter(Boolean).join(' ') || one.id)
			.join(', ');
	}

	/** Reads the provider again, so the report of the removal is the one of this moment. */
	async function remove(): Promise<void> {
		banner = null;

		let current: Provider | null;

		try {
			current = await read();
		} catch (error) {
			banner = failureMessage(error, 'The hub answered no report. Try again.');

			return;
		}

		if (current === null) {
			return;
		}

		const count = current.identity_count;
		const stranded = current.administrators_without_sign_in;
		let message = `Removing ${current.name} deletes ${count} ${count === 1 ? 'identity' : 'identities'}.`;

		if (current.preset === 'local') {
			message += ' Every local account goes with it.';
		}

		if (stranded.length > 0) {
			message += ` ${names(stranded)} then ${stranded.length === 1 ? 'holds' : 'hold'} no way to sign in, and only maroid user password brings ${stranded.length === 1 ? 'that administrator' : 'them'} back.`;
		}

		const confirmed = await confirmDialog?.ask(message, {
			title: 'Remove this provider?',
			confirmLabel: 'Remove'
		});
		if (!confirmed) {
			return;
		}

		busy = true;

		try {
			await api.providers.remove(providerId);
			await goto(resolve('/admin/providers'));
		} catch (error) {
			banner = failureMessage(error, 'The provider stays. Try again.');
		} finally {
			busy = false;
		}
	}

	async function copy(address: string): Promise<void> {
		try {
			await navigator.clipboard.writeText(address);
			copied = true;
		} catch (error) {
			console.error('Failed to copy the redirect address', error);
		}
	}

	$effect(() => {
		void providerId;
		void load();
	});
</script>

{#snippet failure(pointer: string)}
	{#if errors[pointer]}
		<span class="text-error text-xs">{errors[pointer]}</span>
	{/if}
{/snippet}

{#snippet fixed(label: string, value: string)}
	<div class="flex flex-col gap-1">
		<span class="label">{label}</span>
		<input class="input input-sm w-full font-mono" readonly {value} aria-label={label} />
	</div>
{/snippet}

<div class="max-w-2xl">
	{#if status === 'loading'}
		<div class="flex flex-col gap-2" aria-hidden="true">
			<span class="skeleton h-12 w-64"></span>
			{#each [0, 1, 2] as i (i)}
				<span class="skeleton h-24 w-full"></span>
			{/each}
		</div>
	{:else if status === 'missing'}
		<header class="page-header">
			<div>
				<h1 class="page-title">Not found</h1>
				<p class="page-lead">Dex holds no provider with this identifier.</p>
			</div>
		</header>
	{:else if status === 'error'}
		<div role="alert" class="alert alert-error alert-soft">
			<span>The hub answered no provider. Reload the page.</span>
		</div>
	{:else if provider}
		<header class="page-header">
			<div>
				<h1 class="page-title">{provider.name}</h1>
				<div class="mt-2 flex flex-wrap items-center gap-2">
					<span class="badge badge-outline badge-sm">{presetLabel(provider)}</span>
					<span class="meta font-mono">{provider.id}</span>
				</div>
			</div>
		</header>

		{#if banner}
			<div role="alert" class="alert alert-error alert-soft mb-6">
				<span>{banner}</span>
			</div>
		{:else if saved}
			<div role="status" class="alert alert-success alert-soft mb-6">
				<span>Saved. The next sign in reads the change.</span>
			</div>
		{/if}

		{#if provider.static}
			<div role="status" class="alert alert-info alert-soft mb-6">
				<span class="text-sm">
					The configuration file of Dex holds this provider, so Maroid changes nothing on it. Edit
					the file and restart Dex to change it.
				</span>
			</div>
		{:else}
			<section class="section">
				<h2 class="section-title">Provider</h2>
				<form onsubmit={save}>
					<fieldset class="fieldset panel w-full min-w-0 p-5">
						<label class="flex flex-col gap-1">
							<span class="label">Name</span>
							<input class="input input-sm w-full" bind:value={name} disabled={busy} required />
							<span class="label whitespace-normal"
								>The label of the button on the sign in page.</span
							>
							{@render failure('/name')}
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
									{@render failure('/client_id')}
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
									{@render failure('/client_secret')}
								</label>
							</div>

							{#if provider.preset === 'oidc'}
								<label class="mt-2 flex flex-col gap-1">
									<span class="label">Scopes</span>
									<input
										class="input input-sm w-full font-mono"
										bind:value={scopes}
										disabled={busy}
									/>
									{@render failure('/scopes')}
								</label>
							{:else}
								<div class="mt-2">
									{@render fixed('Scopes', 'openid profile')}
								</div>
							{/if}

							<label class="mt-2 flex flex-col gap-1">
								<span class="label">Options</span>
								<textarea
									class="textarea textarea-sm w-full font-mono text-xs"
									rows="5"
									bind:value={options}
									disabled={busy}
								></textarea>
								<span class="label whitespace-normal">
									Further options of the connector of Dex, in YAML. A removed key goes away.
								</span>
								{@render failure('/options')}
								{#each Object.entries(errors).filter( ([pointer]) => pointer.startsWith('/options/') ) as [pointer, detail] (pointer)}
									<span class="text-error text-xs"
										>{pointer.slice('/options/'.length)}: {detail}</span
									>
								{/each}
							</label>
						{/if}

						<div class="mt-3 flex justify-end">
							<button type="submit" class="btn btn-primary btn-sm" disabled={busy}>Save</button>
						</div>
					</fieldset>
				</form>
			</section>

			{#if takesClient && provider.redirect_uri}
				<section class="section">
					<h2 class="section-title">Redirect address</h2>
					<fieldset class="fieldset panel w-full min-w-0 p-5">
						<span class="label whitespace-normal">
							{provider.preset === 'telegram'
								? 'Set the domain of this address in BotFather.'
								: 'Register this address at the provider.'}
						</span>
						<div class="flex w-full gap-2">
							<input
								class="input input-sm w-full font-mono text-xs"
								readonly
								value={provider.redirect_uri}
								aria-label="The redirect address of Dex"
							/>
							<button
								type="button"
								class="btn btn-sm"
								onclick={() => provider?.redirect_uri && copy(provider.redirect_uri)}
							>
								{copied ? 'Copied' : 'Copy'}
							</button>
						</div>
					</fieldset>
				</section>
			{/if}

			<section class="section">
				<h2 class="section-title text-error">Remove</h2>
				<fieldset class="fieldset panel border-error/40 w-full min-w-0 p-5">
					<span class="label whitespace-normal">
						{provider.identity_count}
						{provider.identity_count === 1 ? 'identity' : 'identities'} of this provider go with it.
					</span>
					<div class="flex justify-end">
						<button type="button" class="btn btn-error btn-sm" disabled={busy} onclick={remove}>
							Remove the provider
						</button>
					</div>
				</fieldset>
			</section>
		{/if}
	{/if}
</div>

<ConfirmDialog bind:this={confirmDialog} />
