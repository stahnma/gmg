<script lang="ts">
  import type { Snippet } from 'svelte'

  let {
    open,
    title,
    oncancel,
    children,
    actions,
  }: { open: boolean; title: string; oncancel: () => void; children: Snippet; actions: Snippet } = $props()

  let dialog: HTMLDialogElement | undefined = $state()

  $effect(() => {
    if (!dialog) return
    if (open && !dialog.open) dialog.showModal()
    if (!open && dialog.open) dialog.close()
  })
</script>

<!-- Native modal dialog: focus trapping and Escape (= Cancel) come from the browser. -->
<dialog
  bind:this={dialog}
  aria-label={title}
  oncancel={(e) => {
    e.preventDefault()
    oncancel()
  }}
>
  <h2>{title}</h2>
  <div class="content">{@render children()}</div>
  <div class="actions">{@render actions()}</div>
</dialog>

<style>
  dialog {
    /* sized to content like MUI's dialog, up to 600px */
    width: fit-content;
    min-width: min(300px, calc(100% - 64px));
    max-width: min(600px, calc(100% - 64px));
    padding: 0;
    border: none;
    border-radius: 4px;
    background: #424242;
    color: var(--text);
    box-shadow: 0 11px 15px -7px rgba(0,0,0,.2), 0 24px 38px 3px rgba(0,0,0,.14), 0 9px 46px 8px rgba(0,0,0,.12);
  }
  dialog::backdrop { background: rgba(0, 0, 0, 0.5); }
  h2 { margin: 0; padding: 16px 24px; font-size: 1.25rem; font-weight: 500; line-height: 1.6; letter-spacing: 0.0075em; }
  .content { padding: 0 24px 20px; }
  .actions { display: flex; justify-content: flex-end; gap: 8px; padding: 8px; }
</style>
