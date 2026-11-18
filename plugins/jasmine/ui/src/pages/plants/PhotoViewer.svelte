<script lang="ts" module>
	import type { Moment, Photo } from '../../api';

	export interface ViewedPhoto {
		photo: Photo;
		moment: Moment;
	}
</script>

<script lang="ts">
	import { ageOf, formatDay } from '../../lib/format';

	interface Props {
		/** Every photo of the plant, in the order of the timeline. */
		photos: ViewedPhoto[];
		profileId?: string;
		canWrite: boolean;
		/** True while the page writes, so a second choice waits. */
		busy?: boolean;
		onprofile: (photo: Photo) => void;
		/** Called when an image does not load, because its address expired. */
		onexpired?: () => void;
	}

	let { photos, profileId, canWrite, busy = false, onprofile, onexpired }: Props = $props();

	let dialog = $state<HTMLDialogElement>();
	let index = $state(0);

	const current = $derived(photos[Math.min(index, photos.length - 1)]);

	/** Opens the viewer at the photo. */
	export function open(photoId: string): void {
		index = Math.max(
			0,
			photos.findIndex((item) => item.photo.id === photoId)
		);
		dialog?.showModal();
	}

	function step(by: number): void {
		index = (index + by + photos.length) % photos.length;
	}

	function keydown(event: KeyboardEvent): void {
		if (event.key === 'ArrowLeft') {
			step(-1);
		} else if (event.key === 'ArrowRight') {
			step(1);
		}
	}
</script>

{#snippet chevron(path: string)}
	<svg
		xmlns="http://www.w3.org/2000/svg"
		width="16"
		height="16"
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		aria-hidden="true"
	>
		<path d={path} />
	</svg>
{/snippet}

<dialog bind:this={dialog} class="modal" onkeydown={keydown}>
	<div class="modal-box flex max-h-[95vh] max-w-6xl flex-col gap-3 p-3 sm:p-4">
		{#if current}
			<div class="bg-base-200 rounded-box relative flex min-h-0 grow items-center justify-center">
				{#if current.photo.display}
					<img
						class="max-h-[72vh] w-full object-contain"
						src={current.photo.display.url}
						alt={current.moment.note ?? formatDay(current.moment.taken_on)}
						onerror={() => onexpired?.()}
					/>
				{:else}
					<span class="meta flex aspect-[4/3] w-full items-center justify-center">
						The photo is not ready yet.
					</span>
				{/if}

				{#if photos.length > 1}
					<button
						class="btn btn-circle btn-sm absolute top-1/2 left-2 -translate-y-1/2"
						type="button"
						aria-label="Previous photo"
						onclick={() => step(-1)}
					>
						{@render chevron('m15 18-6-6 6-6')}
					</button>
					<button
						class="btn btn-circle btn-sm absolute top-1/2 right-2 -translate-y-1/2"
						type="button"
						aria-label="Next photo"
						onclick={() => step(1)}
					>
						{@render chevron('m9 18 6-6-6-6')}
					</button>
				{/if}
			</div>

			<div class="flex flex-wrap items-start justify-between gap-3 px-1">
				<div class="min-w-0">
					<p class="font-medium">
						{formatDay(current.moment.taken_on)}
						<span class="meta ml-1">{ageOf(current.moment.taken_on)}</span>
					</p>
					{#if current.moment.note}
						<p class="mt-1 text-sm whitespace-pre-line">{current.moment.note}</p>
					{/if}
				</div>

				<div class="flex shrink-0 items-center gap-2">
					<span class="meta">{index + 1} of {photos.length}</span>
					{#if current.photo.id === profileId}
						<span class="badge badge-sm badge-soft badge-primary">Profile picture</span>
					{:else if canWrite}
						<button
							class="btn btn-sm"
							type="button"
							disabled={busy}
							onclick={() => onprofile(current.photo)}
						>
							Set as profile picture
						</button>
					{/if}
					<button class="btn btn-ghost btn-sm" type="button" onclick={() => dialog?.close()}>
						Close
					</button>
				</div>
			</div>
		{/if}
	</div>
	<form method="dialog" class="modal-backdrop">
		<button>close</button>
	</form>
</dialog>
