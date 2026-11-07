<script lang="ts">
  import type { Snippet } from 'svelte'

  // Row of avatar + text readings and action buttons. Wraps on narrow screens.
  let { children }: { children: Snippet } = $props()
</script>

<div class="readings">{@render children()}</div>

<style>
  .readings {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    padding: 8px 16px;
  }
  .readings :global(.reading) {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 16px;
    font-size: 1rem;
  }
  .readings :global(.avatar) {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 50px;
    height: 50px;
    border-radius: 50%;
    background: var(--avatar-bg);
    color: var(--avatar-fg);
  }
  .readings :global(.actions) { padding: 8px; }
</style>
