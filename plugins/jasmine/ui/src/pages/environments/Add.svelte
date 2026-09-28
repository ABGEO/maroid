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

<div>
  <h2 class="mb-4 text-lg font-semibold">Add Environment</h2>

  <form class="flex max-w-sm flex-col gap-2" onsubmit={submit}>
    <label class="label" for="name">Name</label>
    <input class="input" id="name" bind:value={name} required disabled={submitting} />

    {#if error}
      <p class="text-error">{error}</p>
    {/if}

    <div class="mt-2 flex items-center gap-4">
      <button class="btn btn-primary btn-sm" type="submit" disabled={submitting || name.trim() === ''}>
        {submitting ? 'Saving…' : 'Save'}
      </button>
      <a class="btn btn-ghost btn-sm" href={host.href('/environments')}>Cancel</a>
    </div>
  </form>
</div>
