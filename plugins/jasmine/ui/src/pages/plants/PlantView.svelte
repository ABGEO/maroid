<script lang="ts">
  import { ApiError, PROBLEM_TYPE, type PluginHost } from '@maroid/plugin-sdk';
  import { ConfirmDialog } from '@maroid/ui';

  import type { EnvironmentReference, JasmineApi, Plant, PlantUpdatePayload } from '../../api';
  import { formatMoment } from '../../lib/format';
  import { failureMessage } from '../../lib/problem';
  import CareIntervals from './CareIntervals.svelte';
  import LogForm from './LogForm.svelte';
  import PlantEntries from './PlantEntries.svelte';
  import PlantForm from './PlantForm.svelte';

  interface Props {
    host: PluginHost;
    api: JasmineApi;
    plantId: string;
    environments: EnvironmentReference[];
  }

  let { host, api, plantId, environments }: Props = $props();

  const canWritePlants = $derived(host.can('plants.write'));
  const canLog = $derived(host.can('maintenance.write'));

  let plant = $state<Plant | null>(null);
  let etag = $state<string | undefined>(undefined);
  let status = $state<'loading' | 'ready' | 'missing' | 'error'>('loading');
  let editing = $state(false);
  let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();
  let busy = $state(false);
  let failure = $state<string | null>(null);
  let revision = $state(0);

  async function load(): Promise<void> {
    try {
      const tagged = await api.plants.get(plantId);
      if (tagged === null) {
        return;
      }

      plant = tagged.value;
      etag = tagged.etag;
      status = 'ready';
      host.setTitle(`${plant.display_id} ${plant.common_name}`);
    } catch (error) {
      if (error instanceof ApiError && error.is(PROBLEM_TYPE.notFound)) {
        status = 'missing';
        host.setTitle('Not found');

        return;
      }

      console.error('Failed to load the plant', error);
      status = 'error';
    }
  }

  async function save(payload: PlantUpdatePayload): Promise<void> {
    busy = true;
    failure = null;

    try {
      const saved = await api.plants.update(plantId, payload, etag);
      if (saved === null) {
        return;
      }

      editing = false;
      await load();
    } catch (error) {
      failure = failureMessage(error, 'The plant keeps its values. Try again.');
    } finally {
      busy = false;
    }
  }

  async function remove(): Promise<void> {
    const proceed = await confirmDialog?.ask(
      `${plant?.display_id} ${plant?.common_name} goes, with its care intervals and every maintenance entry.`,
      { title: 'Delete this plant?', confirmLabel: 'Delete' }
    );
    if (!proceed) {
      return;
    }

    busy = true;
    failure = null;

    try {
      await api.plants.remove(plantId, etag);
      host.navigate('/plants');
    } catch (error) {
      failure = failureMessage(error, 'The plant stays. Try again.');
    } finally {
      busy = false;
    }
  }

  $effect(() => {
    void load();
  });
</script>

<div>
  {#if status === 'loading'}
    <span class="skeleton h-32 w-full" aria-hidden="true"></span>
  {:else if status === 'missing'}
    <header class="page-header">
      <div>
        <h1 class="page-title">Not found</h1>
        <p class="page-lead">This workspace holds no plant with this address.</p>
      </div>
    </header>
    <a class="btn btn-sm" href={host.href('/plants')}>Plants</a>
  {:else if status === 'error' || plant === null}
    <div role="alert" class="alert alert-error alert-soft">
      <span>Failed to load the plant.</span>
      <button type="button" class="btn btn-sm" onclick={() => load()}>Retry</button>
    </div>
  {:else}
    <header class="page-header">
      <div>
        <h1 class="page-title">
          <span class="meta">{plant.display_id}</span>
          {plant.common_name}
        </h1>
        <p class="page-lead">
          {plant.environment.display_id}
          {plant.environment.name}
          {#if plant.status === 'inactive'}
            <span class="badge badge-sm badge-ghost ml-2">Inactive</span>
          {/if}
        </p>
      </div>
      {#if canWritePlants && !editing}
        <div class="flex gap-2">
          <button class="btn btn-sm" type="button" onclick={() => (editing = true)}>Edit</button>
          <button class="btn btn-ghost btn-sm text-error" type="button" disabled={busy} onclick={remove}>
            Delete
          </button>
        </div>
      {/if}
    </header>

    {#if failure}
      <div role="alert" class="alert alert-error alert-soft mb-4">
        <span>{failure}</span>
      </div>
    {/if}

    {#if editing}
      <PlantForm {environments} {plant} {busy} onsave={save} oncancel={() => (editing = false)} />
    {:else}
      <dl class="panel grid gap-x-6 gap-y-2 p-5 sm:grid-cols-[auto_1fr]">
        <dt class="meta">Scientific name</dt>
        <dd>{plant.scientific_name ?? 'Not set'}</dd>
        <dt class="meta">Acquired on</dt>
        <dd>{plant.acquired_on ?? 'Not set'}</dd>
        <dt class="meta">Added</dt>
        <dd>{formatMoment(plant.created_at)}</dd>
        <dt class="meta">Care notes</dt>
        <dd class="whitespace-pre-line">{plant.care_notes ?? 'Not set'}</dd>
      </dl>
    {/if}

    <CareIntervals {api} {plantId} canWrite={canWritePlants} {revision} />

    {#if canLog}
      <section class="section">
        <h2 class="section-title">Record care</h2>
        <LogForm {api} plants={[plant]} onrecorded={() => revision++} />
      </section>
    {/if}

    <PlantEntries {api} {plantId} {revision} />
  {/if}

  <ConfirmDialog bind:this={confirmDialog} />
</div>
