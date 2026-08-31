<script lang="ts">
  import type { PluginHost } from '@maroid/plugin-sdk';

  import { createJasmineApi, type Plant } from '../../api';

  let { host }: { host: PluginHost } = $props();

  const api = createJasmineApi(host);

  let status = $state<'loading' | 'ready' | 'error'>('loading');
  let plants = $state<Plant[]>([]);
  let environmentNames = $state<Record<string, string>>({});

  async function load(): Promise<void> {
    status = 'loading';

    try {
      const [plantList, environmentList] = await Promise.all([
        api.plants.list(),
        api.environments.list()
      ]);

      if (plantList === null || environmentList === null) {
        return;
      }

      plants = plantList;
      environmentNames = Object.fromEntries(
        environmentList.map((environment) => [environment.id, environment.name])
      );
      status = 'ready';
    } catch (error) {
      console.error('Failed to load plants', error);
      status = 'error';
    }
  }

  function environmentName(id: string): string {
    return environmentNames[id] ?? id;
  }

  function formatDate(value: string): string {
    return new Date(value).toLocaleDateString();
  }

  $effect(() => {
    void load();
  });
</script>

<div class="page">
    <h2>Plants</h2>

    {#if status === 'loading'}
      <p>Loading…</p>
    {:else if status === 'error'}
      <p class="error">Failed to load plants. <button onclick={load}>Retry</button></p>
    {:else if plants.length === 0}
      <p>No plants yet.</p>
    {:else}
      <table>
        <thead>
          <tr><th>Name</th><th>Species</th><th>Environment</th><th>Created</th></tr>
        </thead>
        <tbody>
          {#each plants as plant (plant.id)}
            <tr>
              <td>{plant.name}</td>
              <td>{plant.species ?? '—'}</td>
              <td>{environmentName(plant.environmentId)}</td>
              <td>{formatDate(plant.createdAt)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
</div>

<style>
  h2 { margin: 0 0 1rem; }
  .error { color: #f87171; }
  table { width: 100%; border-collapse: collapse; }
  th, td { padding: 0.75rem; text-align: left; border-bottom: 1px solid var(--maroid-color-surface, #1e293b); }
  th { font-weight: 600; }
</style>
