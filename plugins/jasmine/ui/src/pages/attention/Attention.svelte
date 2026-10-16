<script lang="ts">
	import { createWriteIntent, failureMessage, type PluginHost } from '@maroid/plugin-sdk';

	import '../../app.css';
	import { createJasmineApi, type CareInterval, type EnvironmentReference } from '../../api';
	import { dueLabel, formatDays } from '../../lib/format';
	import { MAINTENANCE_TYPES, STATE_BADGE, typeLabel } from '../../lib/maintenance';
	import { readAll } from '../../lib/pages';

	interface Group {
		environment: EnvironmentReference;
		lines: CareInterval[];
	}

	let { host }: { host: PluginHost } = $props();

	const api = $derived(createJasmineApi(host));
	const canLog = $derived(host.can('maintenance.write'));
	const intent = createWriteIntent();

	const UPCOMING_DAYS = 3;

	let lines = $state<CareInterval[]>([]);
	let upcoming = $state<CareInterval[]>([]);
	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let pending = $state<string | null>(null);
	let failure = $state<string | null>(null);

	const groups = $derived(group(lines));
	const nextDays = $derived(byDueDate(upcoming));

	function plantHref(care: CareInterval): string {
		return host.href(`/plants/${care.plant.id}`);
	}

	function displayNumber(displayId: string): number {
		return Number(displayId.slice(1));
	}

	function typeOrder(care: CareInterval): number {
		return MAINTENANCE_TYPES.findIndex((option) => option.value === care.type);
	}

	/**
	 * The lines grouped by environment, in the order of the reminder: the
	 * environment number, then overdue before due, then the plant number, then the
	 * order of the maintenance types.
	 */
	function group(items: CareInterval[]): Group[] {
		const sorted = [...items].sort(
			(a, b) =>
				displayNumber(a.plant.environment.display_id) -
					displayNumber(b.plant.environment.display_id) ||
				Number(b.state === 'overdue') - Number(a.state === 'overdue') ||
				displayNumber(a.plant.display_id) - displayNumber(b.plant.display_id) ||
				typeOrder(a) - typeOrder(b)
		);

		const groups: Group[] = [];
		for (const care of sorted) {
			const last = groups.at(-1);
			if (last?.environment.id === care.plant.environment.id) {
				last.lines.push(care);
			} else {
				groups.push({ environment: care.plant.environment, lines: [care] });
			}
		}

		return groups;
	}

	/**
	 * The upcoming lines by the day they become due, then in the order of the
	 * attention list: the environment number, the plant number, the type.
	 */
	function byDueDate(items: CareInterval[]): CareInterval[] {
		return [...items].sort(
			(a, b) =>
				a.due_in - b.due_in ||
				displayNumber(a.plant.environment.display_id) -
					displayNumber(b.plant.environment.display_id) ||
				displayNumber(a.plant.display_id) - displayNumber(b.plant.display_id) ||
				typeOrder(a) - typeOrder(b)
		);
	}

	async function load(): Promise<void> {
		try {
			[lines, upcoming] = await Promise.all([
				readAll((link?: string) => api.careIntervals.list({ state: ['due', 'overdue'] }, link)),
				readAll((link?: string) => api.careIntervals.list({ due_within: UPCOMING_DAYS }, link))
			]);
			status = 'ready';
		} catch (error) {
			console.error('Failed to load the attention list', error);
			status = 'error';
		}
	}

	async function done(care: CareInterval): Promise<void> {
		if (pending !== null) {
			return;
		}

		pending = care.id;
		failure = null;

		try {
			const payload = { plant_ids: [care.plant.id], type: care.type };
			await api.maintenanceEntries.create(payload, { idempotencyKey: intent.keyFor(payload) });
			intent.settle();
			await load();
		} catch (error) {
			failure = failureMessage(error, 'The entry was not recorded. Try again.');
		} finally {
			pending = null;
		}
	}

	$effect(() => {
		void load();
	});
</script>

<div class="max-w-4xl">
	<header class="page-header">
		<div>
			<h1 class="page-title">Needs attention</h1>
			<p class="page-lead">
				The care that is due or overdue, by environment, then the care of the next 3 days.
			</p>
		</div>
	</header>

	{#if failure}
		<div role="alert" class="alert alert-error alert-soft mb-4">
			<span>{failure}</span>
		</div>
	{/if}

	{#if status === 'loading'}
		<div class="flex flex-col gap-2" aria-hidden="true">
			{#each [0, 1, 2] as i (i)}
				<span class="skeleton h-12 w-full"></span>
			{/each}
		</div>
	{:else if status === 'error'}
		<div role="alert" class="alert alert-error alert-soft">
			<span>Failed to load the attention list.</span>
			<button type="button" class="btn btn-sm" onclick={() => load()}>Retry</button>
		</div>
	{:else}
		{#if groups.length === 0}
			<div class="empty-state">Every plant is on schedule.</div>
		{:else}
			{#each groups as { environment, lines: group } (environment.id)}
				<section class="section">
					<h2 class="section-title">
						<span class="meta">{environment.display_id}</span>
						{environment.name}
					</h2>
					<ul class="list panel">
						{#each group as care (care.id)}
							<li class="list-row items-center">
								<span class="meta">{care.plant.display_id}</span>
								<span class="list-col-grow">
									<a class="link link-hover font-medium" href={plantHref(care)}
										>{care.plant.common_name}</a
									>
									<span class="meta ml-2">
										{typeLabel(care.type)}, {formatDays(care.days_since)}, every {formatDays(
											care.days
										)}
									</span>
								</span>
								{#if care.state}
									<span class="badge badge-sm badge-soft {STATE_BADGE[care.state].tone}">
										{STATE_BADGE[care.state].label}
									</span>
								{/if}
								{#if canLog}
									<button
										class="btn btn-primary btn-sm"
										type="button"
										disabled={pending !== null}
										onclick={() => done(care)}
									>
										{pending === care.id ? 'Saving…' : 'Done'}
									</button>
								{/if}
							</li>
						{/each}
					</ul>
				</section>
			{/each}
		{/if}

		<section class="section">
			<h2 class="section-title">Next 3 days</h2>
			{#if nextDays.length === 0}
				<div class="empty-state">No care in the next 3 days.</div>
			{:else}
				<ul class="list panel">
					{#each nextDays as care (care.id)}
						<li class="list-row items-center">
							<span class="meta">{care.plant.display_id}</span>
							<span class="list-col-grow">
								<a class="link link-hover font-medium" href={plantHref(care)}
									>{care.plant.common_name}</a
								>
								<span class="meta ml-2">
									{typeLabel(care.type)}, every {formatDays(care.days)}
								</span>
							</span>
							<span class="badge badge-sm badge-ghost">{dueLabel(care.due_in)}</span>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}
</div>
