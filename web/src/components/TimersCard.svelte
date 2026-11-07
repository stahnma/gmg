<script lang="ts">
  import Card from './Card.svelte'
  import Readings from './Readings.svelte'
  import Dialog from './Dialog.svelte'
  import TextField from './TextField.svelte'
  import Icon from './Icon.svelte'
  import image from '../assets/timers.png'
  import { formatSeconds, parseCountdown } from '../lib/timers'

  // Stopwatch and countdown; runs entirely in the browser.
  let elapsed = $state(0)
  let stopwatch: ReturnType<typeof setInterval> | undefined = $state()
  let remaining = $state(0)
  let countdown: ReturnType<typeof setInterval> | undefined = $state()

  let open = $state(false)
  let input = $state('00:00:00:00')
  let error = $state('')

  $effect(() => () => {
    clearInterval(stopwatch)
    clearInterval(countdown)
  })

  function toggleStopwatch() {
    if (stopwatch) {
      clearInterval(stopwatch)
      stopwatch = undefined
      elapsed = 0
    } else {
      stopwatch = setInterval(() => elapsed++, 1000)
    }
  }

  function cancelCountdown() {
    clearInterval(countdown)
    countdown = undefined
    remaining = 0
  }

  function startCountdown() {
    let seconds: number
    try {
      seconds = parseCountdown(input)
    } catch {
      error = 'Invalid input'
      return
    }
    open = false
    remaining = seconds
    countdown = setInterval(() => {
      if (remaining > 0) remaining--
      if (remaining === 0) {
        clearInterval(countdown)
        countdown = undefined
      }
    }, 1000)
  }
</script>

<Card {image} title="Timers" subtitle="Set a grilling stopwatch or countdown timer.">
  <Readings>
    <div class="reading">
      <span class="avatar"><Icon name="time" /></span>
      <span>Timer: {formatSeconds(elapsed)}</span>
    </div>
    <div class="actions">
      <button class="btn" aria-label={stopwatch ? 'Cancel timer' : 'Start timer'} onclick={toggleStopwatch}>
        {stopwatch ? 'Cancel' : 'Start'}
      </button>
    </div>
    <div class="reading">
      <span class="avatar"><Icon name="countdown" /></span>
      <span>Countdown: {formatSeconds(remaining)}</span>
    </div>
    <div class="actions">
      <button
        class="btn"
        aria-label={countdown ? 'Cancel countdown' : 'Start countdown'}
        onclick={() => (countdown ? cancelCountdown() : (open = true))}
      >
        {countdown ? 'Cancel' : 'Start'}
      </button>
    </div>
  </Readings>
</Card>

<Dialog {open} title="Set the countdown time (dd:hh:mm:ss)" oncancel={() => { open = false; input = '00:00:00:00'; error = '' }}>
  <form id="set-countdown" onsubmit={(e) => { e.preventDefault(); startCountdown() }} oninput={() => (error = '')}>
    <TextField bind:value={input} label="00:01:30:00" placeholder="Set countdown" {error} autofocus />
  </form>
  {#snippet actions()}
    <button class="btn text" type="button" onclick={() => { open = false; input = '00:00:00:00'; error = '' }}>Cancel</button>
    <button class="btn text" type="submit" form="set-countdown" disabled={!!error}>Set</button>
  {/snippet}
</Dialog>
