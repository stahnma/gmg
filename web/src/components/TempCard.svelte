<script lang="ts">
  import Card from './Card.svelte'
  import Readings from './Readings.svelte'
  import Dialog from './Dialog.svelte'
  import TextField from './TextField.svelte'
  import Icon from './Icon.svelte'

  // Grill and food cards: current/desired temperature and a "set" dialog.
  let {
    title,
    subtitle,
    image,
    what,
    current,
    desired,
    enabled,
    min,
    max,
    onsubmit,
  }: {
    title: string
    subtitle: string
    image: string
    what: 'Grill' | 'Food'
    current: number
    desired: number
    enabled: boolean
    min: number // accepted target range, °F (from /api/limits)
    max: number
    onsubmit: (temperature: string) => void
  } = $props()

  let open = $state(false)
  let value = $state('')

  // Same rules as the server: a whole number within the limits.
  const error = $derived.by(() => {
    if (value.trim() === '') return ''
    const n = Number(value)
    if (isNaN(n)) return 'Desired temperature must be a number!'
    if (!Number.isInteger(n)) return 'Desired temperature must be a whole number!'
    if (n < min) return `Desired temperature must be at least ${min} ℉!`
    if (n > max) return `Desired temperature must be at most ${max} ℉!`
    return ''
  })

  function cancel() {
    open = false
    value = ''
  }

  function submit() {
    if (error || value.trim() === '') return
    open = false
    onsubmit(value.trim())
    value = ''
  }
</script>

<Card {image} {title} {subtitle}>
  <Readings>
    <div class="reading">
      <span class="avatar"><Icon name="thermostat" /></span>
      <span>Current: {current} ℉</span>
    </div>
    <div class="reading">
      <span class="avatar"><Icon name="thermostat" /></span>
      <span>{desired ? `Desired: ${desired} ℉` : 'Desired: Not set'}</span>
    </div>
    <div class="actions">
      <button class="btn" disabled={!enabled} onclick={() => (open = true)}>Set {what} Temperature</button>
    </div>
  </Readings>
</Card>

<Dialog {open} title="Set the desired {what.toLowerCase()} temperature" oncancel={cancel}>
  <form id="set-{what}" onsubmit={(e) => { e.preventDefault(); submit() }}>
    <TextField bind:value label="{min}–{max} ℉" placeholder="{what} temperature ℉" {error} autofocus />
  </form>
  {#snippet actions()}
    <button class="btn text" type="button" onclick={cancel}>Cancel</button>
    <button class="btn text" type="submit" form="set-{what}" disabled={!!error || value.trim() === ''}>Set</button>
  {/snippet}
</Dialog>
