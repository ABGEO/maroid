<script lang="ts">
  import type { PluginHost } from '@maroid/plugin-sdk';

  import { createJasmineApi } from '../../api';

  let { host }: { host: PluginHost } = $props();

  const api = createJasmineApi(host);

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
      const created = await api.environments.create({ name });
      if (created === null) {
        return;
      }

      host.navigate('/environments');
    } catch (err) {
      console.error('Failed to create environment', err);
      error = 'Failed to create environment.';
    } finally {
      submitting = false;
    }
  }
</script>

<div class="page">
  <h2>Add Environment</h2>

  <form onsubmit={submit}>
    <label for="name">Name</label>
    <input id="name" bind:value={name} required disabled={submitting} />

    {#if error}
      <p class="error">{error}</p>
    {/if}

    <div class="actions">
      <button type="submit" disabled={submitting || name.trim() === ''}>
        {submitting ? 'Saving…' : 'Save'}
      </button>
      <a href={host.href('/environments')}>Cancel</a>
    </div>
  </form>
</div>

<style>
  h2 { margin: 0 0 1rem; }
  form { display: flex; flex-direction: column; gap: 0.5rem; max-width: 24rem; }
  .actions { display: flex; align-items: center; gap: 1rem; margin-top: 0.5rem; }
  .error { color: #f87171; }
</style>
