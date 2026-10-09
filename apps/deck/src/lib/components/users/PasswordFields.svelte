<script lang="ts">
	interface Props {
		password: string;
		repeated: string;
		label?: string;
		disabled?: boolean;
	}

	let {
		password = $bindable(),
		repeated = $bindable(),
		label = 'New password',
		disabled = false
	}: Props = $props();
</script>

<label class="flex flex-col gap-1">
	<span class="label">{label}</span>
	<input
		class="input input-sm w-full"
		type="password"
		autocomplete="new-password"
		bind:value={password}
		{disabled}
		required
	/>
</label>
<label class="flex flex-col gap-1">
	<span class="label">Repeat the password</span>
	<input
		class="input input-sm w-full"
		type="password"
		autocomplete="new-password"
		bind:value={repeated}
		{disabled}
		required
	/>
</label>
