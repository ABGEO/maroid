<script lang="ts">
	import { useDropzone } from '@uppy/svelte';

	interface Props {
		label: string;
		hint: string;
		accept: string[];
	}

	let { label, hint, accept }: Props = $props();

	const dropzone = useDropzone();
	let over = $state(false);

	const root = dropzone.getRootProps();
	const input = dropzone.getInputProps();
</script>

<div
	role="button"
	tabindex="0"
	class={[
		'border-base-300 hover:border-primary rounded-box flex cursor-pointer flex-col items-center gap-1 border-2 border-dashed p-6 text-center transition-colors',
		over && 'border-primary bg-primary/5'
	]}
	ondragenter={(event) => {
		over = true;
		root.ondragenter(event);
	}}
	ondragover={root.ondragover}
	ondragleave={(event) => {
		over = false;
		root.ondragleave(event);
	}}
	ondrop={(event) => {
		over = false;
		root.ondrop(event);
	}}
	onclick={root.onclick}
	onkeydown={(event) => {
		if (event.key === 'Enter' || event.key === ' ') {
			event.preventDefault();
			root.onclick();
		}
	}}
>
	<input {...input} class="hidden" accept={accept.join(',')} />
	<span class="text-sm font-medium">{label}</span>
	<span class="text-base-content/60 text-xs">{hint}</span>
</div>
