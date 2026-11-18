<script lang="ts">
	import type { ProfilePhoto } from '../../api';

	interface Props {
		name: string;
		photo?: ProfilePhoto;
		/** The Tailwind size class of the circle. */
		size?: string;
		/** The class of the initial when the plant shows no profile picture. */
		textSize?: string;
	}

	let { name, photo, size = 'size-8', textSize = 'text-xs' }: Props = $props();
</script>

{#if photo?.thumbnail}
	<div class="avatar shrink-0">
		<div class="{size} rounded-full">
			<img src={photo.thumbnail.url} alt="" />
		</div>
	</div>
{:else}
	<div class="avatar avatar-placeholder shrink-0" aria-hidden="true">
		<div class="bg-base-300 text-base-content/60 {size} rounded-full">
			<span class={textSize}>{name.slice(0, 1).toUpperCase()}</span>
		</div>
	</div>
{/if}
