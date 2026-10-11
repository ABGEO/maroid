<script lang="ts">
  import { createPager, Pagination } from '@maroid/ui';

  import type {
    JasmineApi,
    MaintenanceEntry,
    MaintenanceEntryFilter,
    MaintenanceType
  } from '../../api';
  import { formatMoment, localNow, toLocalInput, toMoment } from '../../lib/format';
  import { MAINTENANCE_TYPES, typeLabel } from '../../lib/maintenance';
  import { failureMessage } from '../../lib/problem';

  interface Props {
    api: JasmineApi;
    filter: MaintenanceEntryFilter;
    canWrite: boolean;
  }

  let { api, filter, canWrite }: Props = $props();

  const pager = createPager((link?: string) => api.maintenanceEntries.list(filter, link));

  let editing = $state<{ id: string; etag?: string } | null>(null);
  let type = $state<MaintenanceType>('watering');
  let performedAt = $state('');
  let note = $state('');
  let confirming = $state<string | null>(null);
  let busy = $state(false);
  let failure = $state<string | null>(null);

  async function startEdit(entry: MaintenanceEntry): Promise<void> {
    failure = null;
    confirming = null;

    try {
      const tagged = await api.maintenanceEntries.get(entry.id);
      if (tagged === null) {
        return;
      }

      editing = { id: entry.id, etag: tagged.etag };
      type = tagged.value.type;
      performedAt = toLocalInput(tagged.value.performed_at);
      note = tagged.value.note ?? '';
    } catch (error) {
      failure = failureMessage(error, 'The entry could not be opened. Try again.');
    }
  }

  async function run(action: () => Promise<unknown>, fallback: string): Promise<void> {
    if (busy) {
      return;
    }

    busy = true;
    failure = null;

    try {
      await action();
      editing = null;
      confirming = null;
      await pager.reload();
    } catch (error) {
      failure = failureMessage(error, fallback);
    } finally {
      busy = false;
    }
  }

  function save(event: SubmitEvent): Promise<void> {
    event.preventDefault();

    const target = editing;
    const moment = toMoment(performedAt);
    if (target === null || moment === undefined) {
      return Promise.resolve();
    }

    const trimmed = note.trim();

    return run(
      () =>
        api.maintenanceEntries.update(
          target.id,
          { type, performed_at: moment, note: trimmed === '' ? null : trimmed },
          target.etag
        ),
      'The entry keeps its values. Try again.'
    );
  }

  function remove(id: string): Promise<void> {
    return run(() => api.maintenanceEntries.remove(id), 'The entry stays. Try again.');
  }

  $effect(() => {
    void pager.reload();
  });
</script>

{#if failure}
  <div role="alert" class="alert alert-error alert-soft mb-4">
    <span>{failure}</span>
  </div>
{/if}

{#if pager.status === 'loading' && pager.page === null}
  <div class="flex flex-col gap-2" aria-hidden="true">
    {#each [0, 1, 2] as i (i)}
      <span class="skeleton h-12 w-full"></span>
    {/each}
  </div>
{:else if pager.status === 'error'}
  <div role="alert" class="alert alert-error alert-soft">
    <span>Failed to load the maintenance log.</span>
    <button type="button" class="btn btn-sm" onclick={() => pager.reload()}>Retry</button>
  </div>
{:else if pager.page?.items.length === 0}
  <div class="empty-state">No entry matches.</div>
{:else}
  <ul class="list panel">
    {#each pager.page?.items ?? [] as entry (entry.id)}
      <li class="list-row items-center">
        {#if editing?.id === entry.id}
          <form class="list-col-grow flex flex-wrap items-end gap-2" onsubmit={save}>
            <span class="meta self-center">{entry.plant.display_id} {entry.plant.common_name}</span>
            <select class="select select-sm w-auto" aria-label="Care" bind:value={type} disabled={busy}>
              {#each MAINTENANCE_TYPES as option (option.value)}
                <option value={option.value}>{option.label}</option>
              {/each}
            </select>
            <input
              class="input input-sm w-auto"
              type="datetime-local"
              aria-label="When"
              max={localNow()}
              bind:value={performedAt}
              disabled={busy}
              required
            />
            <input
              class="input input-sm grow"
              aria-label="Note"
              placeholder="Note"
              maxlength="1000"
              bind:value={note}
              disabled={busy}
            />
            <button class="btn btn-primary btn-sm" type="submit" disabled={busy || performedAt === ''}>
              Save
            </button>
            <button class="btn btn-ghost btn-sm" type="button" onclick={() => (editing = null)}>
              Cancel
            </button>
          </form>
        {:else}
          <span class="meta whitespace-nowrap">{formatMoment(entry.performed_at)}</span>
          <span class="whitespace-nowrap">
            <span class="meta">{entry.plant.display_id}</span>
            {entry.plant.common_name}
          </span>
          <span class="font-medium">{typeLabel(entry.type)}</span>
          <span class="list-col-grow">{entry.note ?? ''}</span>
          {#if canWrite}
            {#if confirming === entry.id}
              <button class="btn btn-error btn-sm" type="button" disabled={busy} onclick={() => remove(entry.id)}>
                Delete the entry
              </button>
              <button class="btn btn-ghost btn-sm" type="button" onclick={() => (confirming = null)}>
                Keep
              </button>
            {:else}
              <button class="btn btn-ghost btn-sm" type="button" onclick={() => startEdit(entry)}>Edit</button>
              <button class="btn btn-ghost btn-sm" type="button" onclick={() => (confirming = entry.id)}>
                Delete
              </button>
            {/if}
          {/if}
        {/if}
      </li>
    {/each}
  </ul>
  <Pagination {pager} class="mt-4 ml-auto w-fit" />
{/if}
