<script lang="ts">
  import type { PluginHost } from '@maroid/plugin-sdk';

  import '../../app.css';
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

<div>
    <div class="mb-4 flex items-center justify-between">
      <h2 class="text-lg font-semibold">Environments</h2>
      <a class="btn btn-primary btn-sm" href={host.href('/environments/add')}>Add environment</a>
    </div>

    {#if pager.status === 'loading'}
      <p>Loading…</p>
    {:else if pager.status === 'error'}
      <p class="text-error">Failed to load environments. <button class="btn btn-sm" onclick={() => pager.reload()}>Retry</button></p>
    {:else if pager.page?.items.length === 0}
      <p>No environments yet.</p>
    {:else}
      <table class="table">
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
