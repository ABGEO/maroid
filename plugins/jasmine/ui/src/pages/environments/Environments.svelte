<script lang="ts">
  import type { PluginHost } from '@maroid/plugin-sdk';

  import { createJasmineApi, type Environment } from '../../api';

  let { host }: { host: PluginHost } = $props();

  const api = createJasmineApi(host);

  let status = $state<'loading' | 'ready' | 'error'>('loading');
  let environments = $state<Environment[]>([]);

  async function load(): Promise<void> {
    status = 'loading';

    try {
      const result = await api.environments.list();
      if (result === null) {
        return;
      }

      environments = result;
      status = 'ready';
    } catch (error) {
      console.error('Failed to load environments', error);
      status = 'error';
    }
  }

  function formatDate(value: string): string {
    return new Date(value).toLocaleDateString();
  }

  $effect(() => {
    void load();
  });
</script>

<div class="page">
    <div class="header">
      <h2>Environments</h2>
      <a href={host.href('/environments/add')}>Add environment</a>
    </div>

    {#if status === 'loading'}
      <p>Loading…</p>
    {:else if status === 'error'}
      <p class="error">Failed to load environments. <button onclick={load}>Retry</button></p>
    {:else if environments.length === 0}
      <p>No environments yet.</p>
    {:else}
      <table>
        <thead>
          <tr><th>Name</th><th>Created</th></tr>
        </thead>
        <tbody>
          {#each environments as environment (environment.id)}
            <tr>
              <td>{environment.name}</td>
              <td>{formatDate(environment.createdAt)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
</div>

<style>
  h2 { margin: 0 0 1rem; }
  .header { display: flex; align-items: center; justify-content: space-between; }
  .error { color: #f87171; }
  table { width: 100%; border-collapse: collapse; }
  th, td { padding: 0.75rem; text-align: left; border-bottom: 1px solid var(--maroid-color-surface, #1e293b); }
  th { font-weight: 600; }
</style>
