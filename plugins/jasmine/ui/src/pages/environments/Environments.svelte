<script lang="ts">
  import type { PluginHost } from '@maroid/plugin-sdk';
  import { createPager, Pagination } from '@maroid/ui';

  import '../../app.css';
  import { createJasmineApi } from '../../api';

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

<div class="max-w-3xl">
  <header class="page-header">
    <div>
      <h1 class="page-title">Environments</h1>
      <p class="page-lead">The places where your plants grow.</p>
    </div>
    <a class="btn btn-primary btn-sm" href={host.href('/environments/add')}>Add environment</a>
  </header>

  {#if pager.status === 'loading'}
    <div class="flex flex-col gap-2" aria-hidden="true">
      {#each [0, 1, 2] as i (i)}
        <span class="skeleton h-12 w-full"></span>
      {/each}
    </div>
  {:else if pager.status === 'error'}
    <div role="alert" class="alert alert-error alert-soft">
      <span>Failed to load the environments.</span>
      <button type="button" class="btn btn-sm" onclick={() => pager.reload()}>Retry</button>
    </div>
  {:else if pager.page?.items.length === 0}
    <div class="empty-state">This workspace holds no environment.</div>
  {:else}
    <ul class="list panel">
      {#each pager.page?.items ?? [] as environment (environment.id)}
        <li class="list-row items-center">
          <span class="list-col-grow font-medium">{environment.name}</span>
          <span class="meta">{formatDate(environment.created_at)}</span>
        </li>
      {/each}
    </ul>
    <Pagination {pager} class="mt-4 ml-auto w-fit" />
  {/if}
</div>
