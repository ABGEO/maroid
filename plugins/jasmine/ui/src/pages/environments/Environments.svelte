<script lang="ts">
  import { createWriteIntent, failureMessage, type PluginHost } from '@maroid/plugin-sdk';
  import { ConfirmDialog, createPager, Pagination } from '@maroid/ui';

  import '../../app.css';
  import { createJasmineApi, type Environment } from '../../api';

  const MAX_NAME = 100;

  let { host }: { host: PluginHost } = $props();

  const api = $derived(createJasmineApi(host));
  const canWrite = $derived(host.can('environments.write'));
  const pager = createPager((link?: string) => api.environments.list(link));
  const intent = createWriteIntent();

  let newName = $state('');
  let editing = $state<string | null>(null);
  let editName = $state('');
  let confirmDialog = $state<ReturnType<typeof ConfirmDialog>>();
  let busy = $state(false);
  let failure = $state<string | null>(null);

  async function run(action: () => Promise<unknown>, fallback: string): Promise<boolean> {
    if (busy) {
      return false;
    }

    busy = true;
    failure = null;

    try {
      await action();
      await pager.reload();

      return true;
    } catch (error) {
      failure = failureMessage(error, fallback);

      return false;
    } finally {
      busy = false;
    }
  }

  async function add(event: SubmitEvent): Promise<void> {
    event.preventDefault();

    const payload = { name: newName.trim() };
    const added = await run(
      () => api.environments.create(payload, { idempotencyKey: intent.keyFor(payload) }),
      'The environment was not added. Try again.'
    );
    if (added) {
      intent.settle();
      newName = '';
    }
  }

  function startRename(environment: Environment): void {
    editing = environment.id;
    editName = environment.name;
  }

  async function rename(event: SubmitEvent, id: string): Promise<void> {
    event.preventDefault();

    const renamed = await run(
      () => api.environments.update(id, { name: editName.trim() }),
      'The name stays. Try again.'
    );
    if (renamed) {
      editing = null;
    }
  }

  async function remove(environment: Environment): Promise<void> {
    const proceed = await confirmDialog?.ask(
      `${environment.display_id} ${environment.name} goes. An environment that holds a plant stays.`,
      { title: 'Delete this environment?', confirmLabel: 'Delete' }
    );
    if (proceed) {
      await run(() => api.environments.remove(environment.id), 'The environment stays. Try again.');
    }
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
  </header>

  {#if canWrite}
    <form class="panel mb-6 flex items-end gap-2 p-4" onsubmit={add}>
      <fieldset class="fieldset grow">
        <legend class="fieldset-legend">New environment</legend>
        <input
          class="input w-full"
          placeholder="Office"
          maxlength={MAX_NAME}
          bind:value={newName}
          disabled={busy}
          required
        />
      </fieldset>
      <button class="btn btn-primary btn-sm mb-1" type="submit" disabled={busy || newName.trim() === ''}>
        Add
      </button>
    </form>
  {/if}

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
      <span>Failed to load the environments.</span>
      <button type="button" class="btn btn-sm" onclick={() => pager.reload()}>Retry</button>
    </div>
  {:else if pager.page?.items.length === 0}
    <div class="empty-state">This workspace holds no environment.</div>
  {:else}
    <ul class="list panel">
      {#each pager.page?.items ?? [] as environment (environment.id)}
        <li class="list-row items-center">
          <span class="meta">{environment.display_id}</span>
          {#if editing === environment.id}
            <form class="list-col-grow flex gap-2" onsubmit={(event) => rename(event, environment.id)}>
              <input
                class="input input-sm grow"
                maxlength={MAX_NAME}
                bind:value={editName}
                disabled={busy}
                required
              />
              <button class="btn btn-primary btn-sm" type="submit" disabled={busy || editName.trim() === ''}>
                Save
              </button>
              <button class="btn btn-ghost btn-sm" type="button" onclick={() => (editing = null)}>
                Cancel
              </button>
            </form>
          {:else}
            <span class="list-col-grow font-medium">{environment.name}</span>
            <span class="meta">
              {environment.active_plant_count}
              {environment.active_plant_count === 1 ? 'active plant' : 'active plants'}
            </span>
            {#if canWrite}
              <button class="btn btn-ghost btn-sm" type="button" disabled={busy} onclick={() => startRename(environment)}>
                Rename
              </button>
              <button
                class="btn btn-ghost btn-sm text-error"
                type="button"
                disabled={busy}
                onclick={() => remove(environment)}
              >
                Delete
              </button>
            {/if}
          {/if}
        </li>
      {/each}
    </ul>
    <Pagination {pager} class="mt-4 ml-auto w-fit" />
  {/if}

  <ConfirmDialog bind:this={confirmDialog} />
</div>
