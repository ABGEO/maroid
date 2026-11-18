<script lang="ts">
	import { failureMessage, type PluginHost } from '@maroid/plugin-sdk';
	import { ConfirmDialog } from '@maroid/ui';

	import type { JasmineApi, Moment, Photo, Plant } from '../../api';
	import { ageOf, formatDay } from '../../lib/format';
	import { readAll } from '../../lib/pages';
	import MomentDialog from './MomentDialog.svelte';
	import PhotoViewer, { type ViewedPhoto } from './PhotoViewer.svelte';

	interface Props {
		host: PluginHost;
		api: JasmineApi;
		plant: Plant;
		canWrite: boolean;
		/** Called after a write that can move the profile picture of the plant. */
		onchanged: () => void;
	}

	let { host, api, plant, canWrite, onchanged }: Props = $props();

	const PAGE_LIMIT = 100;
	/** A failed image reloads the moments at most once in this span, as an address expires. */
	const RELOAD_SPAN_MS = 60_000;

	let moments = $state<Moment[]>([]);
	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let busy = $state(false);
	let failure = $state<string | null>(null);
	let loadedAt = 0;

	let momentDialog = $state<ReturnType<typeof MomentDialog>>();
	let viewer = $state<ReturnType<typeof PhotoViewer>>();
	let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();

	const plantId = $derived(plant.id);
	const profileId = $derived(plant.profile_photo?.id);
	const viewed = $derived<ViewedPhoto[]>(
		moments.flatMap((moment) => moment.photos.map((photo) => ({ photo, moment })))
	);

	async function load(): Promise<void> {
		loadedAt = Date.now();

		try {
			moments = await readAll((link?: string) => api.moments.list(plantId, PAGE_LIMIT, link));
			status = 'ready';
		} catch (error) {
			console.error('Failed to load the photos', error);
			status = 'error';
		}
	}

	function expired(): void {
		if (Date.now() - loadedAt > RELOAD_SPAN_MS) {
			void load();
		}
	}

	async function change(write: () => Promise<unknown>, keeps: string): Promise<void> {
		busy = true;
		failure = null;

		try {
			await write();
			await load();
			onchanged();
		} catch (error) {
			failure = failureMessage(error, keeps);
		} finally {
			busy = false;
		}
	}

	/** Closes the open dropdown, which daisyUI keeps open while it holds the focus. */
	function closeMenu(): void {
		if (document.activeElement instanceof HTMLElement) {
			document.activeElement.blur();
		}
	}

	function photoCount(count: number): string {
		return count === 1 ? '1 photo' : `${count} photos`;
	}

	async function deleteMoment(moment: Moment): Promise<void> {
		closeMenu();
		const holdsProfile = moment.photos.some((photo) => photo.id === profileId);
		const proceed = await confirmDialog?.ask(
			`The moment of ${formatDay(moment.taken_on)} goes with its ${photoCount(moment.photos.length)}.` +
				(holdsProfile ? ' The plant then has no profile picture.' : ''),
			{ title: 'Delete this moment?', confirmLabel: 'Delete' }
		);

		if (proceed) {
			await change(() => api.moments.remove(plantId, moment.id), 'The moment stays. Try again.');
		}
	}

	async function deletePhoto(moment: Moment, photo: Photo): Promise<void> {
		closeMenu();
		const last = moment.photos.length === 1;
		const proceed = await confirmDialog?.ask(
			(last ? 'This is the last photo of the moment, so the moment goes too.' : 'The photo goes.') +
				(photo.id === profileId ? ' The plant then has no profile picture.' : ''),
			{ title: 'Delete this photo?', confirmLabel: 'Delete' }
		);

		if (proceed) {
			await change(
				() => api.moments.removePhoto(plantId, moment.id, photo.id),
				'The photo stays. Try again.'
			);
		}
	}

	function editMoment(moment: Moment): void {
		closeMenu();
		momentDialog?.open(moment);
	}

	function setProfile(photo: Photo): Promise<void> {
		closeMenu();

		return change(
			() => api.plants.setProfilePhoto(plantId, photo.id),
			'The profile picture stays. Try again.'
		);
	}

	$effect(() => {
		void load();
	});
</script>

{#snippet ellipsis()}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width="16"
		height="16"
		viewBox="0 0 24 24"
		fill="currentColor"
		aria-hidden="true"
	>
		<circle cx="5" cy="12" r="1.75" />
		<circle cx="12" cy="12" r="1.75" />
		<circle cx="19" cy="12" r="1.75" />
	</svg>
{/snippet}

<section class="section">
	<div class="mb-3 flex items-center justify-between gap-2">
		<h2 class="section-title mb-0">Photos</h2>
		{#if canWrite && moments.length > 0}
			<button class="btn btn-sm" type="button" onclick={() => momentDialog?.open()}>
				Add a moment
			</button>
		{/if}
	</div>

	{#if failure}
		<div role="alert" class="alert alert-error alert-soft mb-4">
			<span>{failure}</span>
		</div>
	{/if}

	{#if status === 'loading'}
		<span class="skeleton h-32 w-full" aria-hidden="true"></span>
	{:else if status === 'error'}
		<div role="alert" class="alert alert-error alert-soft">
			<span>Failed to load the photos.</span>
			<button type="button" class="btn btn-sm" onclick={() => load()}>Retry</button>
		</div>
	{:else if moments.length === 0}
		<div class="empty-state flex flex-col items-center gap-3">
			<span>No photo yet. Add one when you repot, or when a new leaf comes out.</span>
			{#if canWrite}
				<button class="btn btn-primary btn-sm" type="button" onclick={() => momentDialog?.open()}>
					Add the first moment
				</button>
			{/if}
		</div>
	{:else}
		<ul class="timeline timeline-vertical timeline-compact timeline-snap-icon">
			{#each moments as moment, position (moment.id)}
				<li>
					{#if position > 0}<hr class="bg-base-300" />{/if}
					<div class="timeline-middle">
						<span class="bg-primary block size-2.5 rounded-full"></span>
					</div>
					<div class="timeline-end mb-8 w-full pl-2">
						<div class="flex items-start justify-between gap-2">
							<p>
								<time class="font-medium" datetime={moment.taken_on}
									>{formatDay(moment.taken_on)}</time
								>
								<span class="meta ml-1">{ageOf(moment.taken_on)}</span>
							</p>
							{#if canWrite}
								<div class="dropdown dropdown-end">
									<div
										tabindex="0"
										role="button"
										class={['btn btn-ghost btn-square btn-xs', busy && 'btn-disabled']}
										aria-label="Actions of the moment"
									>
										{@render ellipsis()}
									</div>
									<ul
										tabindex="-1"
										class="dropdown-content menu menu-sm bg-base-100 border-base-300 rounded-box z-50 mt-1 w-40 border p-2 shadow-lg"
									>
										<li>
											<button type="button" onclick={() => editMoment(moment)}>Edit</button>
										</li>
										<li>
											<button class="text-error" type="button" onclick={() => deleteMoment(moment)}>
												Delete
											</button>
										</li>
									</ul>
								</div>
							{/if}
						</div>

						{#if moment.note}
							<p class="mt-1 max-w-prose text-sm whitespace-pre-line">{moment.note}</p>
						{/if}

						<ul class="mt-3 grid grid-cols-3 gap-2 sm:grid-cols-4 lg:grid-cols-5">
							{#each moment.photos as photo (photo.id)}
								<li class="group relative">
									<button
										class="rounded-box focus-visible:outline-primary bg-base-200 block aspect-square w-full overflow-hidden"
										type="button"
										aria-label="Open the photo of {formatDay(moment.taken_on)}"
										onclick={() => viewer?.open(photo.id)}
									>
										{#if photo.thumbnail}
											<img
												class="size-full object-cover transition-transform group-hover:scale-105"
												src={photo.thumbnail.url}
												alt=""
												loading="lazy"
												onerror={expired}
											/>
										{:else}
											<span class="loading loading-spinner loading-sm"></span>
										{/if}
									</button>

									{#if photo.id === profileId}
										<span class="badge badge-sm badge-primary absolute top-1.5 left-1.5">
											Profile
										</span>
									{/if}

									{#if canWrite}
										<div
											class="dropdown dropdown-end absolute top-1 right-1 pointer-fine:opacity-0 pointer-fine:group-focus-within:opacity-100 pointer-fine:group-hover:opacity-100"
										>
											<div
												tabindex="0"
												role="button"
												class={['btn btn-circle btn-xs', busy && 'btn-disabled']}
												aria-label="Actions of the photo"
											>
												{@render ellipsis()}
											</div>
											<ul
												tabindex="-1"
												class="dropdown-content menu menu-sm bg-base-100 border-base-300 rounded-box z-50 mt-1 w-52 border p-2 shadow-lg"
											>
												{#if photo.id !== profileId}
													<li>
														<button type="button" onclick={() => setProfile(photo)}>
															Set as profile picture
														</button>
													</li>
												{/if}
												<li>
													<button
														class="text-error"
														type="button"
														onclick={() => deletePhoto(moment, photo)}>Delete photo</button
													>
												</li>
											</ul>
										</div>
									{/if}
								</li>
							{/each}
						</ul>
					</div>
					{#if position < moments.length - 1}<hr class="bg-base-300" />{/if}
				</li>
			{/each}
		</ul>
	{/if}

	<MomentDialog
		bind:this={momentDialog}
		{host}
		{api}
		{plantId}
		onsaved={async () => {
			await load();
			onchanged();
		}}
	/>
	<PhotoViewer
		bind:this={viewer}
		photos={viewed}
		{profileId}
		{canWrite}
		{busy}
		onprofile={setProfile}
		onexpired={expired}
	/>
	<ConfirmDialog bind:this={confirmDialog} />
</section>
