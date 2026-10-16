<script lang="ts">
	import { ConfirmDialog } from '@maroid/ui';

	import {
		ApiError,
		PROBLEM_TYPE,
		api,
		type NotificationProvider,
		type NotificationProviderKind,
		type NotificationRoute,
		type NotificationTopic
	} from '$lib/api';
	import ProviderDialog from '$lib/components/notifications/ProviderDialog.svelte';
	import RouteDialog from '$lib/components/notifications/RouteDialog.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import Section from '$lib/components/ui/Section.svelte';
	import SkeletonList from '$lib/components/ui/SkeletonList.svelte';
	import { describeDestination } from '$lib/notifications/form';
	import { PERMISSION, holds } from '$lib/permissions';
	import { displayNameOf } from '$lib/plugins/capabilities';
	import { knownPlugin } from '$lib/state/plugins.svelte';

	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let kinds = $state<NotificationProviderKind[]>([]);
	let providers = $state<NotificationProvider[]>([]);
	let topics = $state<NotificationTopic[]>([]);
	let outcomes = $state<Record<string, { kind: 'success' | 'error'; text: string }>>({});
	let busy = $state<string | null>(null);
	let actionFailure = $state<string | null>(null);

	let providerDialog = $state<ReturnType<typeof ProviderDialog>>();
	let routeDialog = $state<ReturnType<typeof RouteDialog>>();
	let confirm = $state<ReturnType<typeof ConfirmDialog>>();

	const workspaceId = $derived(data.workspace.id);
	const manager = $derived(holds(data.workspace, PERMISSION.notificationsWrite));
	const topicsByPlugin = $derived(
		topics.reduce((groups, topic) => {
			groups.set(topic.plugin_id, [...(groups.get(topic.plugin_id) ?? []), topic]);

			return groups;
		}, new Map<string, NotificationTopic[]>())
	);
	const routeCounts = $derived(
		topics
			.flatMap((topic) => topic.routes)
			.reduce(
				(counts, route) => counts.set(route.provider_id, (counts.get(route.provider_id) ?? 0) + 1),
				new Map<string, number>()
			)
	);

	function kindOf(kindId: string): NotificationProviderKind | undefined {
		return kinds.find((one) => one.id === kindId);
	}

	function providerOf(providerId: string): NotificationProvider | undefined {
		return providers.find((one) => one.id === providerId);
	}

	function routeCountOf(providerId: string): number {
		return routeCounts.get(providerId) ?? 0;
	}

	function reasonOf(error: unknown, fallback: string): string {
		if (error instanceof ApiError && error.problem !== null) {
			return error.problem.detail ?? error.problem.title;
		}

		return fallback;
	}

	async function load(): Promise<void> {
		try {
			const [kindList, providerList, topicList] = await Promise.all([
				api.notifications.kinds(workspaceId),
				api.notifications.providers(workspaceId),
				api.notifications.topics(workspaceId)
			]);

			if (kindList === null || providerList === null || topicList === null) {
				return;
			}

			kinds = kindList;
			providers = providerList;
			topics = topicList;
			status = 'ready';
		} catch (error) {
			console.error('Failed to load the notifications', error);
			status = 'error';
		}
	}

	/** Runs one change of the page, and reloads the lists after it. */
	async function act(key: string, change: () => Promise<void>, fallback: string): Promise<void> {
		busy = key;
		actionFailure = null;

		try {
			await change();
			await load();
		} catch (error) {
			actionFailure = reasonOf(error, fallback);
		} finally {
			busy = null;
		}
	}

	function toggleProvider(provider: NotificationProvider): Promise<void> {
		return act(
			provider.id,
			async () => {
				await api.notifications.changeProvider(workspaceId, provider.id, {
					enabled: !provider.enabled
				});
			},
			'The provider stays as it was. Try again.'
		);
	}

	async function removeProvider(provider: NotificationProvider): Promise<void> {
		const count = routeCountOf(provider.id);
		const routes = count === 1 ? '1 route' : `${count} routes`;
		const confirmed = await confirm?.ask(
			`Maroid removes ${provider.name}, its credential, and its ${routes}. Each topic it served becomes unrouted.`,
			{ title: 'Remove the provider?', confirmLabel: 'Remove' }
		);

		if (confirmed !== true) {
			return;
		}

		await act(
			provider.id,
			() => api.notifications.removeProvider(workspaceId, provider.id),
			'The provider stays. Try again.'
		);
	}

	function toggleRoute(route: NotificationRoute): Promise<void> {
		return act(
			route.id,
			async () => {
				await api.notifications.changeRoute(workspaceId, route.id, { enabled: !route.enabled });
			},
			'The route stays as it was. Try again.'
		);
	}

	async function removeRoute(route: NotificationRoute): Promise<void> {
		const confirmed = await confirm?.ask('The topic stops reaching this destination.', {
			title: 'Remove the route?',
			confirmLabel: 'Remove'
		});

		if (confirmed !== true) {
			return;
		}

		await act(
			route.id,
			() => api.notifications.removeRoute(workspaceId, route.id),
			'The route stays. Try again.'
		);
	}

	async function testRoute(route: NotificationRoute): Promise<void> {
		busy = route.id;

		try {
			await api.notifications.testRoute(workspaceId, route.id);
			outcomes[route.id] = { kind: 'success', text: 'Sent. Check the destination.' };
		} catch (error) {
			const service = kindOf(providerOf(route.provider_id)?.kind ?? '')?.name ?? 'The service';
			let text = reasonOf(error, 'Sending failed. Try again.');

			if (error instanceof ApiError && error.is(PROBLEM_TYPE.deliveryRefused)) {
				text = `${service} refused it: ${error.problem?.detail ?? ''}`;
			} else if (error instanceof ApiError && error.is(PROBLEM_TYPE.messagingUnreachable)) {
				text = `${service} did not answer.`;
			}

			outcomes[route.id] = { kind: 'error', text };
		} finally {
			busy = null;
		}
	}

	$effect(() => {
		void load();
	});
</script>

<div class="max-w-3xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">Notifications</h1>
			<p class="page-lead">
				The accounts that deliver the notifications of this workspace, and where each notification
				goes. A notification with no route reaches nobody.
			</p>
		</div>
	</header>

	{#if actionFailure}
		<Alert kind="error" class="mb-6"><span>{actionFailure}</span></Alert>
	{/if}

	{#if status === 'loading'}
		<SkeletonList />
	{:else if status === 'error'}
		<Alert kind="error" class="mb-6">
			<span>The notification settings did not load.</span>
			<button type="button" class="btn btn-sm" onclick={() => void load()}>Retry</button>
		</Alert>
	{:else}
		<Section title="Providers">
			{#if manager}
				<div class="mb-3">
					<button
						type="button"
						class="btn btn-primary btn-sm"
						disabled={kinds.length === 0}
						onclick={() => providerDialog?.open()}
					>
						Add a provider
					</button>
				</div>
			{/if}

			{#if providers.length === 0}
				<div class="empty-state">
					This workspace holds no provider. {manager
						? 'Add one to deliver its notifications.'
						: 'A manager adds one.'}
				</div>
			{:else}
				<ul class="list panel">
					{#each providers as provider (provider.id)}
						<li class="list-row flex flex-wrap items-center justify-between gap-3">
							<span class="min-w-0">
								<span class="block text-sm font-semibold">{provider.name}</span>
								<span class="meta block">
									{routeCountOf(provider.id)}
									{routeCountOf(provider.id) === 1 ? 'route' : 'routes'}
								</span>
							</span>
							<span class="flex flex-wrap items-center gap-2">
								<span class="badge badge-outline badge-sm">
									{kindOf(provider.kind)?.name ?? provider.kind}
								</span>
								{#if !provider.enabled}
									<span class="badge badge-warning badge-soft badge-sm">Disabled</span>
								{/if}
								{#if manager}
									<button
										type="button"
										class="btn btn-ghost btn-xs"
										disabled={busy !== null}
										onclick={() => providerDialog?.open(provider)}
									>
										Edit
									</button>
									<button
										type="button"
										class="btn btn-ghost btn-xs"
										disabled={busy !== null}
										onclick={() => void toggleProvider(provider)}
									>
										{provider.enabled ? 'Disable' : 'Enable'}
									</button>
									<button
										type="button"
										class="btn btn-ghost btn-xs text-error"
										disabled={busy !== null}
										onclick={() => void removeProvider(provider)}
									>
										Remove
									</button>
								{/if}
							</span>
						</li>
					{/each}
				</ul>
			{/if}
		</Section>

		<Section title="Notification topics">
			{#if topics.length === 0}
				<div class="empty-state">No plugin of this workspace sends a notification.</div>
			{/if}

			{#each [...topicsByPlugin] as [pluginId, pluginTopics] (pluginId)}
				<h3 class="mt-4 mb-2 text-sm font-semibold">
					{displayNameOf(knownPlugin(pluginId), pluginId)}
				</h3>
				<ul class="list panel">
					{#each pluginTopics as topic (topic.topic_id)}
						<li class="list-row flex flex-col items-stretch gap-2">
							<span class="flex flex-wrap items-center justify-between gap-2">
								<span class="min-w-0">
									<span class="block text-sm font-semibold">{topic.name}</span>
									{#if topic.description}
										<span class="meta block">{topic.description}</span>
									{/if}
								</span>
								<span class="flex items-center gap-2">
									{#if topic.unrouted}
										<span class="badge badge-warning badge-soft badge-sm">Unrouted</span>
									{/if}
									{#if manager}
										<button
											type="button"
											class="btn btn-ghost btn-xs"
											disabled={providers.length === 0 || busy !== null}
											onclick={() => routeDialog?.open(topic)}
										>
											Add route
										</button>
									{/if}
								</span>
							</span>

							{#each topic.routes as route (route.id)}
								{@const provider = providerOf(route.provider_id)}
								<span
									class="border-base-300 flex flex-wrap items-center justify-between gap-2 border-t pt-2"
								>
									<span class="min-w-0 text-sm">
										{provider?.name ?? 'Unknown provider'}
										<span class="meta">
											{describeDestination(route.destination, kindOf(provider?.kind ?? ''))}
										</span>
										{#if outcomes[route.id]}
											<span
												class={[
													'block text-xs',
													outcomes[route.id].kind === 'success' ? 'text-success' : 'text-error'
												]}
											>
												{outcomes[route.id].text}
											</span>
										{/if}
									</span>
									<span class="flex flex-wrap items-center gap-2">
										{#if !route.enabled}
											<span class="badge badge-warning badge-soft badge-sm">Disabled</span>
										{/if}
										{#if manager}
											<button
												type="button"
												class="btn btn-ghost btn-xs"
												disabled={busy !== null}
												onclick={() => void testRoute(route)}
											>
												{#if busy === route.id}
													<span class="loading loading-spinner loading-xs"></span>
												{/if}
												Test
											</button>
											<button
												type="button"
												class="btn btn-ghost btn-xs"
												disabled={busy !== null}
												onclick={() => routeDialog?.open(topic, route)}
											>
												Edit
											</button>
											<button
												type="button"
												class="btn btn-ghost btn-xs"
												disabled={busy !== null}
												onclick={() => void toggleRoute(route)}
											>
												{route.enabled ? 'Disable' : 'Enable'}
											</button>
											<button
												type="button"
												class="btn btn-ghost btn-xs text-error"
												disabled={busy !== null}
												onclick={() => void removeRoute(route)}
											>
												Remove
											</button>
										{/if}
									</span>
								</span>
							{/each}
						</li>
					{/each}
				</ul>
			{/each}
		</Section>
	{/if}
</div>

<ProviderDialog bind:this={providerDialog} {workspaceId} {kinds} onsaved={() => void load()} />
<RouteDialog
	bind:this={routeDialog}
	{workspaceId}
	{kinds}
	{providers}
	onsaved={() => void load()}
/>
<ConfirmDialog bind:this={confirm} />
