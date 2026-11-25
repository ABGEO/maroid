<script lang="ts">
	import { formatMoment } from '@maroid/plugin-sdk';

	import type { Invitation } from '#lib/api/index.ts';
	import Alert from '#lib/components/ui/Alert.svelte';
	import CopyField from '#lib/components/ui/CopyField.svelte';
	import { timeZone } from '#lib/state/user.svelte.ts';

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
		{formatMoment(invitation.expires_at, timeZone())}.
	</span>
	<CopyField value={invitation.address} label="The address of the invitation">
		<button type="button" class="btn btn-ghost btn-sm" onclick={onclose}>Close</button>
	</CopyField>
</Alert>
