<script lang="ts">
	import { ApiError, createWriteIntent, failureMessage } from '@maroid/plugin-sdk';

	import type { JasmineApi, MaintenanceType, PlantReference } from '../../api';
	import { localNow, toMoment } from '../../lib/format';
	import { MAINTENANCE_TYPES } from '../../lib/maintenance';

	interface Props {
		api: JasmineApi;
		/** The plants that each receive one entry. */
		plants: PlantReference[];
		onrecorded: () => void;
	}

	let { api, plants, onrecorded }: Props = $props();

	const intent = createWriteIntent();
	const missingPlant = /^#\/plant_ids\/(\d+)$/;

	let type = $state<MaintenanceType>('watering');
	let performedAt = $state('');
	let note = $state('');
	let busy = $state(false);
	let failure = $state<string | null>(null);

	function reason(error: unknown): string {
		const index =
			error instanceof ApiError ? error.fields[0]?.pointer.match(missingPlant)?.[1] : undefined;
		const plant = index === undefined ? undefined : plants[Number(index)];
		if (plant) {
			return `${plant.display_id} ${plant.common_name} no longer exists. No entry was recorded.`;
		}

		return failureMessage(error, 'No entry was recorded. Try again.');
	}

	async function submit(event: SubmitEvent): Promise<void> {
		event.preventDefault();

		if (busy) {
			return;
		}

		busy = true;
		failure = null;

		const trimmed = note.trim();
		const payload = {
			plant_ids: plants.map((plant) => plant.id),
			type,
			performed_at: toMoment(performedAt),
			note: trimmed === '' ? null : trimmed
		};

		try {
			const recorded = await api.maintenanceEntries.create(payload, {
				idempotencyKey: intent.keyFor(payload)
			});
			if (recorded === null) {
				return;
			}

			intent.settle();
			performedAt = '';
			note = '';
			onrecorded();
		} catch (error) {
			failure = reason(error);
		} finally {
			busy = false;
		}
	}
</script>

<form class="panel flex flex-wrap items-end gap-2 p-4" onsubmit={submit}>
	<fieldset class="fieldset">
		<legend class="fieldset-legend">Care</legend>
		<select class="select select-sm" bind:value={type} disabled={busy}>
			{#each MAINTENANCE_TYPES as option (option.value)}
				<option value={option.value}>{option.label}</option>
			{/each}
		</select>
	</fieldset>

	<fieldset class="fieldset">
		<legend class="fieldset-legend">When</legend>
		<input
			class="input input-sm"
			type="datetime-local"
			max={localNow()}
			bind:value={performedAt}
			disabled={busy}
		/>
	</fieldset>

	<fieldset class="fieldset grow">
		<legend class="fieldset-legend">Note</legend>
		<input class="input input-sm w-full" maxlength="1000" bind:value={note} disabled={busy} />
	</fieldset>

	<button class="btn btn-primary btn-sm mb-1" type="submit" disabled={busy || plants.length === 0}>
		{plants.length === 1 ? 'Record' : `Record for ${plants.length} plants`}
	</button>

	{#if failure}
		<div role="alert" class="alert alert-error alert-soft w-full">
			<span>{failure}</span>
		</div>
	{/if}
	<p class="label w-full">An empty time records the current time.</p>
</form>
