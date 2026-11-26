<script lang="ts">
  import Icon from './Icon.svelte'

  // Shown while the grill isn't answering: the readings below are old.
  let { lastSeen, error = '' }: { lastSeen: string | null; error?: string } = $props()

  let now = $state(Date.now())
  $effect(() => {
    const t = setInterval(() => (now = Date.now()), 10_000)
    return () => clearInterval(t)
  })

  const ago = $derived.by(() => {
    if (!lastSeen) return ''
    const s = Math.max(0, Math.round((now - Date.parse(lastSeen)) / 1000))
    if (s < 60) return 'less than a minute ago'
    const m = Math.round(s / 60)
    if (m < 60) return `${m} min ago`
    const h = Math.floor(m / 60)
    return `${h} h ${m % 60} min ago`
  })
  const at = $derived(lastSeen ? new Date(lastSeen).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : '')
</script>

<div class="banner" role="alert" title={error || undefined}>
  <Icon name="warning" color="var(--error)" />
  <span>
    <strong>Grill not responding.</strong>
    {#if lastSeen}Readings are from {at} ({ago}).{/if}
    Check the grill's power and Wi-Fi.
  </span>
</div>

<style>
  .banner {
    display: flex;
    align-items: center;
    gap: 12px;
    margin: 1rem 2rem 0;
    padding: 12px 16px;
    border-radius: 4px;
    border-left: 4px solid var(--error);
    background: var(--paper);
    box-shadow: var(--shadow);
  }
  @media (max-width: 600px) {
    .banner { margin: 1rem 0.5rem 0; }
  }
</style>
