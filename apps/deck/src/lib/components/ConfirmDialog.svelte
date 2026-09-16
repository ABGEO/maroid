<script lang="ts">
	interface AskOptions {
		title?: string;
		confirmLabel?: string;
		cancelLabel?: string;
	}

	let dialog: HTMLDialogElement | undefined = $state();
	let message = $state('');
	let title = $state('');
	let confirmLabel = $state('Confirm');
	let cancelLabel = $state('Cancel');
	let settle: ((confirmed: boolean) => void) | undefined;

	/** Opens the dialog and resolves with whether the person confirmed. */
	export function ask(text: string, options: AskOptions = {}): Promise<boolean> {
		message = text;
		title = options.title ?? '';
		confirmLabel = options.confirmLabel ?? 'Confirm';
		cancelLabel = options.cancelLabel ?? 'Cancel';

		dialog?.showModal();

		return new Promise((resolve) => {
			settle = resolve;
		});
	}

	function close(confirmed: boolean): void {
		settle?.(confirmed);
		settle = undefined;
		dialog?.close();
	}
</script>

<dialog bind:this={dialog} class="modal" onclose={() => close(false)}>
	<div class="modal-box">
		{#if title}
			<h3 class="text-base font-semibold">{title}</h3>
		{/if}
		<p class="text-base-content/70 py-4 text-sm">{message}</p>
		<div class="modal-action">
			<button type="button" class="btn btn-ghost btn-sm" onclick={() => close(false)}>
				{cancelLabel}
			</button>
			<button type="button" class="btn btn-error btn-sm" onclick={() => close(true)}>
				{confirmLabel}
			</button>
		</div>
	</div>
	<form method="dialog" class="modal-backdrop">
		<button>close</button>
	</form>
</dialog>
