<script lang="ts">
	import type { Invitation } from '$lib/api';
	import Alert from '$lib/components/ui/Alert.svelte';
	import CopyField from '$lib/components/ui/CopyField.svelte';

	interface Props {
		name: string;
		invitation: Invitation;
		onclose: () => void;
	}

	let { name, invitation, onclose }: Props = $props();
</script>

<Alert kind="info" class="mb-6 flex flex-col items-start gap-2">
	<span class="text-sm">
		The invitation of {name}. It shows this one time, and it expires on
		{new Date(invitation.expires_at).toLocaleString()}.
	</span>
	<CopyField value={invitation.address} label="The address of the invitation">
		<button type="button" class="btn btn-ghost btn-sm" onclick={onclose}>Close</button>
	</CopyField>
</Alert>
