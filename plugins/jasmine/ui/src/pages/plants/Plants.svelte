<script lang="ts">
  import type { PluginHost } from '@maroid/plugin-sdk';
  import { createPager, Pagination } from '@maroid/ui';

  import '../../app.css';
  import { createJasmineApi } from '../../api';

  let { host }: { host: PluginHost } = $props();

  const api = createJasmineApi(host);

  const pager = createPager(api.plants.list);

  function formatDate(value: string): string {
    return new Date(value).toLocaleDateString();
  }

  $effect(() => {
    void pager.reload();
  });
</script>

<div class="max-w-4xl">
  <header class="page-header">
    <div>
      <h1 class="page-title">Plants</h1>
      <p class="page-lead">The plants of this workspace and the environment of each one.</p>
    </div>
  </header>

  {#if pager.status === 'loading'}
    <div class="flex flex-col gap-2" aria-hidden="true">
      {#each [0, 1, 2] as i (i)}
        <span class="skeleton h-12 w-full"></span>
      {/each}
    </div>
  {:else if pager.status === 'error'}
    <div role="alert" class="alert alert-error alert-soft">
      <span>Failed to load the plants.</span>
      <button type="button" class="btn btn-sm" onclick={() => pager.reload()}>Retry</button>
    </div>
  {:else if pager.page?.items.length === 0}
    <div class="empty-state">This workspace holds no plant.</div>
  {:else}
    <div class="panel overflow-x-auto">
      <table class="table">
        <thead>
          <tr><th>ID</th><th>Name</th><th>Scientific name</th><th>Environment</th><th>Created</th></tr>
        </thead>
        <tbody>
          {#each pager.page?.items ?? [] as plant (plant.id)}
            <tr>
              <td class="meta">{plant.display_id}</td>
              <td class="font-medium">{plant.common_name}</td>
              <td>{plant.scientific_name ?? ''}</td>
              <td>{plant.environment.display_id} {plant.environment.name}</td>
              <td class="meta">{formatDate(plant.created_at)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    <Pagination {pager} class="mt-4 ml-auto w-fit" />
  {/if}
</div>
