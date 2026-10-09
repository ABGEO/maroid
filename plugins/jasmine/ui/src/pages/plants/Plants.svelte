<script lang="ts">
  import type { PluginHost } from '@maroid/plugin-sdk';
  import { createPager, Pagination } from '@maroid/ui';

  import '../../app.css';
  import { createJasmineApi, type Plant } from '../../api';

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
          <tr><th>Name</th><th>Species</th><th>Environment</th><th>Created</th></tr>
        </thead>
        <tbody>
          {#each pager.page?.items ?? [] as plant (plant.id)}
            <tr>
              <td class="font-medium">{plant.name}</td>
              <td>{plant.species ?? '—'}</td>
              <td>{environmentName(plant.environment_id)}</td>
              <td class="meta">{formatDate(plant.created_at)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    <Pagination {pager} class="mt-4 ml-auto w-fit" />
  {/if}
</div>
