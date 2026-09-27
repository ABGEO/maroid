<script lang="ts">
  import type { Pager } from './paging.svelte';

  let { pager }: { pager: Pager<unknown> } = $props();
</script>

{#if pager.hasPrevious || pager.hasNext}
  <nav class="pager">
    <button onclick={() => pager.previous()} disabled={!pager.hasPrevious}>Previous</button>
    <button onclick={() => pager.next()} disabled={!pager.hasNext}>Next</button>
  </nav>
{/if}

<style>
  .pager { display: flex; justify-content: flex-end; gap: 0.5rem; margin-top: 1rem; }
</style>
