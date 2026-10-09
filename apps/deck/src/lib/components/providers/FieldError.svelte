<script lang="ts">
	interface Props {
		errors: Record<string, string>;
		pointer: string;
	}

	let { errors, pointer }: Props = $props();
</script>

{#if errors[pointer]}
	<span class="text-error text-xs">{errors[pointer]}</span>
{/if}
