<script lang="ts">
  import type { EnvironmentReference, Plant, PlantStatus, PlantUpdatePayload } from '../../api';
  import { localNow } from '../../lib/format';

  interface Props {
    environments: EnvironmentReference[];
    /** The plant to change, or absent to add a plant. */
    plant?: Plant;
    busy: boolean;
    onsave: (payload: PlantUpdatePayload) => void;
    oncancel: () => void;
  }

  let { environments, plant, busy, onsave, oncancel }: Props = $props();

  const today = localNow().slice(0, 10);

  let commonName = $derived(plant?.common_name ?? '');
  let scientificName = $derived(plant?.scientific_name ?? '');
  let environmentId = $derived(plant?.environment.id ?? environments[0]?.id ?? '');
  let acquiredOn = $derived(plant?.acquired_on ?? '');
  let careNotes = $derived(plant?.care_notes ?? '');
  let status = $derived<PlantStatus>(plant?.status ?? 'active');

  const valid = $derived(commonName.trim() !== '' && environmentId !== '');

  function optional(value: string): string | null {
    const trimmed = value.trim();

    return trimmed === '' ? null : trimmed;
  }

  function submit(event: SubmitEvent): void {
    event.preventDefault();

    onsave({
      common_name: commonName.trim(),
      scientific_name: optional(scientificName),
      environment_id: environmentId,
      acquired_on: optional(acquiredOn),
      care_notes: optional(careNotes),
      status
    });
  }
</script>

<form class="panel grid gap-x-4 p-5 sm:grid-cols-2" onsubmit={submit}>
  <fieldset class="fieldset">
    <legend class="fieldset-legend">Common name</legend>
    <input class="input w-full" maxlength="100" bind:value={commonName} disabled={busy} required />
  </fieldset>

  <fieldset class="fieldset">
    <legend class="fieldset-legend">Scientific name</legend>
    <input class="input w-full" maxlength="150" bind:value={scientificName} disabled={busy} />
    <p class="label">Optional</p>
  </fieldset>

  <fieldset class="fieldset">
    <legend class="fieldset-legend">Environment</legend>
    <select class="select w-full" bind:value={environmentId} disabled={busy} required>
      {#each environments as environment (environment.id)}
        <option value={environment.id}>{environment.display_id} {environment.name}</option>
      {/each}
    </select>
  </fieldset>

  <fieldset class="fieldset">
    <legend class="fieldset-legend">Acquired on</legend>
    <input class="input w-full" type="date" max={today} bind:value={acquiredOn} disabled={busy} />
    <p class="label">Optional</p>
  </fieldset>

  {#if plant}
    <fieldset class="fieldset">
      <legend class="fieldset-legend">Status</legend>
      <select class="select w-full" bind:value={status} disabled={busy}>
        <option value="active">Active</option>
        <option value="inactive">Inactive</option>
      </select>
      <p class="label">An inactive plant keeps its history and gets no reminder.</p>
    </fieldset>
  {/if}

  <fieldset class="fieldset sm:col-span-2">
    <legend class="fieldset-legend">Care notes</legend>
    <textarea class="textarea w-full" rows="4" maxlength="10000" bind:value={careNotes} disabled={busy}
    ></textarea>
    <p class="label">Optional</p>
  </fieldset>

  <div class="mt-4 flex items-center gap-2 sm:col-span-2">
    <button class="btn btn-primary btn-sm" type="submit" disabled={busy || !valid}>
      {busy ? 'Saving…' : 'Save'}
    </button>
    <button class="btn btn-ghost btn-sm" type="button" onclick={oncancel}>Cancel</button>
  </div>
</form>
