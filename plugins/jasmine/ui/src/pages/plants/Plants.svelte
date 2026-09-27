<script lang="ts">
  import type { PluginHost } from '@maroid/plugin-sdk';

  import { createJasmineApi, type Plant } from '../../api';
  import Pager from '../../lib/Pager.svelte';
  import { createPager } from '../../lib/paging.svelte';

  let { host }: { host: PluginHost } = $props();

  const api = createJasmineApi(host);

  let environmentNames = $state<Record<string, string>>({});

  // A page of plants names only its own environments, so the page reads each of
  // those by identifier rather than every page of environments. An environment
  // that fails to read shows its identifier.
  async function nameEnvironments(plants: Plant[]): Promise<void> {
    const unnamed = [...new Set(plants.map((plant) => plant.environment_id))].filter(
      (id) => !(id in environmentNames)
    );

    const environments = await Promise.all(unnamed.map((id) => api.environments.get(id).catch(() => null)));

    for (const environment of environments) {
      if (environment !== null) {
        environmentNames[environment.id] = environment.name;
      }
    }
  }

  const pager = createPager(async (link?: string) => {
    const page = await api.plants.list(link);
    if (page !== null) {
      await nameEnvironments(page.items);
    }

    return page;
  });

  function environmentName(id: string): string {
    return environmentNames[id] ?? id;
  }

  function formatDate(value: string): string {
    return new Date(value).toLocaleDateString();
  }

  $effect(() => {
    void pager.reload();
  });
</script>

<div class="page">
    <h2>Plants</h2>

    {#if pager.status === 'loading'}
      <p>Loading…</p>
    {:else if pager.status === 'error'}
      <p class="error">Failed to load plants. <button onclick={() => pager.reload()}>Retry</button></p>
    {:else if pager.page?.items.length === 0}
      <p>No plants yet.</p>
    {:else}
      <table>
        <thead>
          <tr><th>Name</th><th>Species</th><th>Environment</th><th>Created</th></tr>
        </thead>
        <tbody>
          {#each pager.page?.items ?? [] as plant (plant.id)}
            <tr>
              <td>{plant.name}</td>
              <td>{plant.species ?? '—'}</td>
              <td>{environmentName(plant.environment_id)}</td>
              <td>{formatDate(plant.created_at)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
      <Pager {pager} />
    {/if}
</div>

<style>
  h2 { margin: 0 0 1rem; }
  .error { color: #f87171; }
  table { width: 100%; border-collapse: collapse; }
  th, td { padding: 0.75rem; text-align: left; border-bottom: 1px solid var(--maroid-color-surface, #1e293b); }
  th { font-weight: 600; }
</style>
