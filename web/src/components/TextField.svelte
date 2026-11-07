<script lang="ts">
  // Outlined text field with a floating label and helper/error text.
  let {
    value = $bindable(''),
    label,
    placeholder = '',
    error = '',
    autofocus = false,
  }: { value?: string; label: string; placeholder?: string; error?: string; autofocus?: boolean } = $props()

  const id = `tf-${Math.random().toString(36).slice(2, 9)}`
</script>

<div class="field" class:error={!!error}>
  <!-- svelte-ignore a11y_autofocus -->
  <input {id} bind:value {placeholder} {autofocus} aria-invalid={!!error} aria-describedby={error ? `${id}-help` : undefined} />
  <label for={id}>{label}</label>
  {#if error}<p id="{id}-help" class="help">{error}</p>{/if}
</div>

<style>
  .field { position: relative; margin-top: 8px; margin-bottom: 4px; }
  input {
    width: 100%;
    padding: 16.5px 14px;
    border: 1px solid rgba(255, 255, 255, 0.23);
    border-radius: 4px;
    background: transparent;
    color: var(--text);
    font: inherit;
    outline: none;
  }
  input:hover { border-color: var(--text); }
  input:focus { border: 2px solid var(--primary); padding: 15.5px 13px; }
  label {
    position: absolute;
    left: 9px;
    top: -9px;
    padding: 0 5px;
    background: #424242;
    color: rgba(255, 255, 255, 0.7);
    font-size: 0.75rem;
    line-height: 1.4375em;
    letter-spacing: 0.00938em;
  }
  .error input, .error input:focus { border-color: var(--error); }
  .error label, .help { color: var(--error); }
  .help { margin: 3px 14px 0; font-size: 0.75rem; line-height: 1.66; letter-spacing: 0.03333em; }
</style>
