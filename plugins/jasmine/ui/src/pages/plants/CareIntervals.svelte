<script lang="ts">
  import type { CareInterval, JasmineApi, MaintenanceType } from '../../api';
  import { formatDays } from '../../lib/format';
  import { MAINTENANCE_TYPES, STATE_BADGE } from '../../lib/maintenance';
  import { failureMessage } from '../../lib/problem';

  interface Props {
    api: JasmineApi;
    plantId: string;
    canWrite: boolean;
    /** Changes when an entry of the plant lands, so the counts read again. */
    revision: number;
  }

  let { api, plantId, canWrite, revision }: Props = $props();

  let intervals = $state<CareInterval[]>([]);
  let drafts = $state<Partial<Record<MaintenanceType, number | null>>>({});
  let status = $state<'loading' | 'ready' | 'error'>('loading');
  let busy = $state(false);
  let failure = $state<string | null>(null);

  const byType = $derived(new Map(intervals.map((care) => [care.type, care])));

  async function load(): Promise<void> {
    try {
      const page = await api.careIntervals.listForPlant(plantId);
      if (page === null) {
        return;
      }

      intervals = page.items;
      drafts = Object.fromEntries(page.items.map((care) => [care.type, care.days]));
      status = 'ready';
    } catch (error) {
      console.error('Failed to load the care intervals', error);
      status = 'error';
    }
  }

  async function write(action: () => Promise<unknown>, fallback: string): Promise<void> {
    if (busy) {
      return;
    }

    busy = true;
    failure = null;

    try {
      await action();
      await load();
    } catch (error) {
      failure = failureMessage(error, fallback);
    } finally {
      busy = false;
    }
  }

  function set(type: MaintenanceType): Promise<void> {
    return write(
      () => api.careIntervals.put(plantId, type, { days: Number(drafts[type]) }),
      'The care interval stays. Try again.'
    );
  }

  function remove(type: MaintenanceType): Promise<void> {
    return write(() => api.careIntervals.remove(plantId, type), 'The care interval stays. Try again.');
  }

  $effect(() => {
    void revision;
    void load();
  });
</script>

<section class="section">
  <h2 class="section-title">Care intervals</h2>

  {#if failure}
    <div role="alert" class="alert alert-error alert-soft mb-4">
      <span>{failure}</span>
    </div>
  {/if}

  {#if status === 'loading'}
    <span class="skeleton h-40 w-full" aria-hidden="true"></span>
  {:else if status === 'error'}
    <div role="alert" class="alert alert-error alert-soft">
      <span>Failed to load the care intervals.</span>
      <button type="button" class="btn btn-sm" onclick={() => load()}>Retry</button>
    </div>
  {:else}
    <div class="panel overflow-x-auto">
      <table class="table">
        <thead>
          <tr><th>Care</th><th>Every</th><th>Since</th><th>State</th><th></th></tr>
        </thead>
        <tbody>
          {#each MAINTENANCE_TYPES as option (option.value)}
            {@const care = byType.get(option.value)}
            <tr>
              <td class="font-medium">{option.label}</td>
              <td>
                {#if canWrite}
                  <label class="input input-sm w-32">
                    <input
                      type="number"
                      min="1"
                      max="365"
                      placeholder="Days"
                      bind:value={drafts[option.value]}
                      disabled={busy}
                    />
                    <span class="meta">days</span>
                  </label>
                {:else if care}
                  {formatDays(care.days)}
                {/if}
              </td>
              <td class="meta">
                {#if care}
                  {formatDays(care.days_since)}
                  {care.last_performed_at ? '' : '(no entry yet)'}
                {/if}
              </td>
              <td>
                {#if care?.state}
                  <span class="badge badge-sm badge-soft {STATE_BADGE[care.state].tone}">
                    {STATE_BADGE[care.state].label}
                  </span>
                {/if}
              </td>
              <td class="text-right whitespace-nowrap">
                {#if canWrite}
                  <button
                    class="btn btn-ghost btn-sm"
                    type="button"
                    disabled={busy || !drafts[option.value]}
                    onclick={() => set(option.value)}
                  >
                    Set
                  </button>
                  {#if care}
                    <button class="btn btn-ghost btn-sm" type="button" disabled={busy} onclick={() => remove(option.value)}>
                      Remove
                    </button>
                  {/if}
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</section>
