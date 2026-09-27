<script lang="ts">
  import type { PluginHost } from '@maroid/plugin-sdk';

  import { createJasmineApi } from '../../api';
  import Pager from '../../lib/Pager.svelte';
  import { createPager } from '../../lib/paging.svelte';

  let { host }: { host: PluginHost } = $props();

  const api = createJasmineApi(host);
  const pager = createPager(api.environments.list);

  function formatDate(value: string): string {
    return new Date(value).toLocaleDateString();
  }

  $effect(() => {
    void pager.reload();
  });
</script>

<div class="page">
    <div class="header">
      <h2>Environments</h2>
      <a href={host.href('/environments/add')}>Add environment</a>
    </div>

    {#if pager.status === 'loading'}
      <p>Loading…</p>
    {:else if pager.status === 'error'}
      <p class="error">Failed to load environments. <button onclick={() => pager.reload()}>Retry</button></p>
    {:else if pager.page?.items.length === 0}
      <p>No environments yet.</p>
    {:else}
      <table>
        <thead>
          <tr><th>Name</th><th>Created</th></tr>
        </thead>
        <tbody>
          {#each pager.page?.items ?? [] as environment (environment.id)}
            <tr>
              <td>{environment.name}</td>
              <td>{formatDate(environment.created_at)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
      <Pager {pager} />
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
