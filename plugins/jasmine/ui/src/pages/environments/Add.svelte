<script lang="ts">
  import { createWriteIntent, type PluginHost } from '@maroid/plugin-sdk';

  import '../../app.css';
  import { createJasmineApi } from '../../api';

  let { host }: { host: PluginHost } = $props();

  const api = createJasmineApi(host);
  const intent = createWriteIntent();

  let name = $state('');
  let submitting = $state(false);
  let error = $state<string | null>(null);

  async function submit(event: SubmitEvent): Promise<void> {
    event.preventDefault();

    if (submitting) {
      return;
    }

    submitting = true;
    error = null;

    try {
      const payload = { name };
      const created = await api.environments.create(payload, {
        idempotencyKey: intent.keyFor(payload)
      });
      if (created === null) {
        return;
      }

      intent.settle();
      host.navigate('/environments');
    } catch (err) {
      console.error('Failed to create environment', err);
      error = 'Failed to create environment.';
    } finally {
      submitting = false;
    }
  }
</script>

<div class="max-w-2xl">
  <header class="page-header">
    <div>
      <h1 class="page-title">Add an environment</h1>
      <p class="page-lead">A place where your plants grow, such as a room or a balcony.</p>
    </div>
  </header>

  <form class="panel max-w-md p-5" onsubmit={submit}>
    <fieldset class="fieldset">
      <legend class="fieldset-legend">Name</legend>
      <input class="input w-full" id="name" bind:value={name} required disabled={submitting} />
    </fieldset>

    {#if error}
      <div role="alert" class="alert alert-error alert-soft mt-4">
        <span>{error}</span>
      </div>
    {/if}

    <div class="mt-4 flex items-center gap-2">
      <button class="btn btn-primary btn-sm" type="submit" disabled={submitting || name.trim() === ''}>
        {submitting ? 'Saving…' : 'Save'}
      </button>
      <a class="btn btn-ghost btn-sm" href={host.href('/environments')}>Cancel</a>
    </div>
  </form>
</div>
