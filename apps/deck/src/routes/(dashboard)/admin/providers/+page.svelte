<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import {
		api,
		createWriteIntent,
		type NewProvider,
		type Provider,
		type ProviderPreset
	} from '$lib/api';
	import {
		failureMessage,
		fieldErrors,
		parseOptions,
		parseScopes,
		presetLabel
	} from '$lib/providers/form';

	const intent = createWriteIntent();

	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let providers = $state<Provider[]>([]);
	let busy = $state(false);
	let banner = $state<string | null>(null);
	let errors = $state<Record<string, string>>({});
	let copied = $state(false);

	let preset = $state<ProviderPreset>('oidc');
	let id = $state('');
	let name = $state('');
	let issuer = $state('');
	let clientId = $state('');
	let clientSecret = $state('');
	let userIdKey = $state('');
	let scopes = $state('');
	let options = $state('');

	const redirectUri = $derived(providers.find((one) => one.redirect_uri)?.redirect_uri ?? '');
	const taken = $derived(new Set(providers.map((one) => one.id)));
	const presets: { value: ProviderPreset; label: string; once?: string }[] = [
		{ value: 'oidc', label: 'OpenID Connect' },
		{ value: 'telegram', label: 'Telegram', once: 'telegram' },
		{ value: 'local', label: 'Email', once: 'local' }
	];

	async function load(): Promise<void> {
		status = 'loading';

		try {
			const list = await api.providers.list();
			if (list === null) {
				return;
			}

			providers = list;
			status = 'ready';
		} catch (error) {
			console.error('Failed to load the providers', error);
			status = 'error';
		}
	}

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

	async function create(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		errors = {};
		banner = null;

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
			banner = failureMessage(error, 'Dex holds no new provider. Try again.');
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
		void load();
	});
</script>

{#snippet failure(pointer: string)}
	{#if errors[pointer]}
		<span class="text-error text-xs">{errors[pointer]}</span>
	{/if}
{/snippet}

<div class="max-w-3xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">Sign-in providers</h1>
			<p class="page-lead">
				The accounts that a person signs in with. A change reaches the sign in page of Dex at the
				next sign in.
			</p>
		</div>
	</header>

	{#if banner}
		<div role="alert" class="alert alert-error alert-soft mb-6">
			<span>{banner}</span>
		</div>
	{/if}

	{#if redirectUri}
		<div role="status" class="alert alert-info alert-soft mb-6 flex flex-col items-start gap-2">
			<span class="text-sm">
				Register this redirect address at an OpenID Connect provider, or set its domain in BotFather
				for Telegram, before you add the provider here.
			</span>
			<div class="flex w-full gap-2">
				<input
					class="input input-sm w-full font-mono text-xs"
					readonly
					value={redirectUri}
					aria-label="The redirect address of Dex"
				/>
				<button type="button" class="btn btn-sm" onclick={() => copy(redirectUri)}>
					{copied ? 'Copied' : 'Copy'}
				</button>
			</div>
		</div>
	{/if}

	<section class="section">
		<h2 class="section-title">Add a provider</h2>
		<form onsubmit={create}>
			<fieldset class="fieldset panel w-full min-w-0 p-5">
				<div class="join" role="radiogroup" aria-label="The kind of the provider">
					{#each presets as one (one.value)}
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
						Adds "Log in with Email" to the sign in page. A person signs in with an email address
						and a password that an administrator gives on the page of the user.
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
								<span class="label whitespace-normal"
									>It never changes after the provider exists.</span
								>
								{@render failure('/id')}
							</label>
							<label class="flex flex-col gap-1">
								<span class="label">Name</span>
								<input class="input input-sm w-full" bind:value={name} required />
								<span class="label whitespace-normal"
									>The label of the button on the sign in page.</span
								>
								{@render failure('/name')}
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
							<span class="label whitespace-normal"
								>It never changes after the provider exists.</span
							>
							{@render failure('/issuer')}
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
							{@render failure('/client_id')}
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
							{@render failure('/client_secret')}
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
								<span class="label whitespace-normal"
									>It never changes after the provider exists.</span
								>
								{@render failure('/user_id_key')}
							</label>
							<label class="flex flex-col gap-1">
								<span class="label">Scopes</span>
								<input
									class="input input-sm w-full font-mono"
									bind:value={scopes}
									placeholder="openid profile email"
								/>
								{@render failure('/scopes')}
							</label>
						</div>
					{/if}

					<label class="mt-2 flex flex-col gap-1">
						<span class="label">Options</span>
						<textarea
							class="textarea textarea-sm w-full font-mono text-xs"
							rows="4"
							bind:value={options}
							placeholder="getUserInfo: true"
						></textarea>
						<span class="label whitespace-normal">
							Further options of the connector of Dex, in YAML. A field above sets its own key.
						</span>
						{@render failure('/options')}
						{#each Object.entries(errors).filter( ([pointer]) => pointer.startsWith('/options/') ) as [pointer, detail] (pointer)}
							<span class="text-error text-xs">{pointer.slice('/options/'.length)}: {detail}</span>
						{/each}
					</label>
				{/if}

				<div class="mt-3 flex justify-end">
					<button type="submit" class="btn btn-primary btn-sm" disabled={busy}>Add</button>
				</div>
			</fieldset>
		</form>
	</section>

	<section class="section">
		<h2 class="section-title">All providers</h2>

		{#if status === 'loading'}
			<div class="flex flex-col gap-2" aria-hidden="true">
				{#each [0, 1, 2] as i (i)}
					<span class="skeleton h-12 w-full"></span>
				{/each}
			</div>
		{:else if status === 'error'}
			<div role="alert" class="alert alert-error alert-soft mb-6">
				<span>The hub answered no provider. Reload the page.</span>
			</div>
		{:else}
			<ul class="list panel">
				{#each providers as provider (provider.id)}
					<li class="list-row flex flex-wrap items-center justify-between gap-3">
						<span class="min-w-0">
							<a
								class="link link-hover text-sm font-semibold"
								href={resolve('/(dashboard)/admin/providers/[provider]', { provider: provider.id })}
							>
								{provider.name}
							</a>
							<span class="meta block truncate font-mono">
								{provider.id}
							</span>
						</span>
						<span class="flex items-center gap-2">
							<span class="badge badge-outline badge-sm">{presetLabel(provider)}</span>
							<span class="text-base-content/60 text-xs">
								{provider.identity_count}
								{provider.identity_count === 1 ? 'identity' : 'identities'}
							</span>
						</span>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</div>
