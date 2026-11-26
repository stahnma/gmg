<script lang="ts">
  import { onMount } from 'svelte'
  import Toolbar from './components/Toolbar.svelte'
  import TempCard from './components/TempCard.svelte'
  import TimersCard from './components/TimersCard.svelte'
  import HistoryCard from './components/HistoryCard.svelte'
  import Connecting from './components/Connecting.svelte'
  import Toasts from './components/Toasts.svelte'
  import StaleBanner from './components/StaleBanner.svelte'
  import BottomNav from './components/BottomNav.svelte'
  import grillImage from './assets/grill.png'
  import foodImage from './assets/food.png'
  import { GrillClient } from './lib/api'
  import { connectEvents } from './lib/events'
  import { toasts } from './lib/toasts.svelte'
  import { DEFAULT_LIMITS, type GrillAlert, type GrillHealth, type GrillStatus, type Limits, type Point } from './lib/types'

  const HISTORY_BACKFILL_S = 8 * 60 * 60
  const client = new GrillClient(window.location.origin)

  let status = $state<GrillStatus>({
    state: 'off', isOn: false, currentGrillTemp: 0, desiredGrillTemp: 0, currentFoodTemp: 0,
    desiredFoodTemp: 0, fanModeActive: false, lowPelletAlarmActive: false,
  })
  let connected = $state(false) // live event stream
  let hasStatus = $state(false) // at least one status received
  let health = $state<GrillHealth | null>(null) // null until the server says
  // The grill is answering. Older servers send no `connection` events, so
  // until one arrives, a status counts as connected.
  const grillConnected = $derived(hasStatus && (health?.reachable ?? true))
  const stale = $derived(hasStatus && health?.reachable === false)
  let pending = $state(0) // commands in flight
  let showTimers = $state(false)
  let showHistory = $state(false)
  let limits = $state<Limits>(DEFAULT_LIMITS)
  let grillPoints = $state.raw<Point[]>([])
  let foodPoints = $state.raw<Point[]>([])

  const loading = $derived(pending > 0)
  const canChangeTemp = $derived(grillConnected && status.isOn && !loading && !status.fanModeActive)
  const canPowerOn = $derived(grillConnected && !loading && !status.fanModeActive)

  function playAlertSound(beep?: string) {
    if (!beep) return
    const audio = new Audio(new URL(beep, window.location.origin).href)
    // Browsers block autoplay until the user has interacted with the page
    audio.play().catch((err: Error) => console.warn(`Unable to play alert sound: ${err.message}`))
  }

  function onAlert(alert: GrillAlert) {
    playAlertSound(alert.beep)
    const message = `${alert.name}: ${alert.reason}`
    if (alert.level === 'error') toasts.error(message)
    else if (alert.level === 'warning') toasts.warning(message)
    else toasts.info(message)
  }

  function onStatus(s: GrillStatus) {
    const now = Date.now()
    grillPoints = [...grillPoints, { x: now, y: s.currentGrillTemp }]
    foodPoints = [...foodPoints, { x: now, y: s.currentFoodTemp }]
    status = s
    hasStatus = true
  }

  function onHealth(h: GrillHealth) {
    health = h
  }

  async function run(command: () => Promise<void>) {
    pending++
    try {
      await command()
    } catch (err) {
      toasts.error((err as Error).message)
    } finally {
      pending--
    }
  }

  onMount(() => {
    client
      .getLimits()
      .then((l) => {
        if (l?.grill && l?.food) limits = l // an older server answers 404
      })
      .catch((err: Error) => console.warn(`Unable to load temperature limits: ${err.message}`))
    const since = Math.floor(Date.now() / 1000) - HISTORY_BACKFILL_S
    client
      .getTemperatureHistory(since)
      .then((rows) => {
        if (rows.length === 0) return
        const history = (pick: (r: (typeof rows)[number]) => number | null) => rows.map((r) => ({ x: r.timestamp * 1000, y: pick(r) }))
        // History first, then anything received live while it loaded.
        grillPoints = [...history((r) => r.grill_temperature), ...grillPoints]
        foodPoints = [...history((r) => r.food_temperature), ...foodPoints]
      })
      .catch((err: Error) => console.warn(`Unable to load history: ${err.message}`))

    return connectEvents('/api/events', { onStatus, onAlert, onHealth, onConnectionChange: (c) => (connected = c) })
  })
</script>

<div class="app">
  <div class="container">
    {#if !connected}<Connecting />{/if}
    {#if stale && health}<StaleBanner lastSeen={health.lastSeen} error={health.error} />{/if}
    <Toolbar
      powerOn={status.isOn}
      timersOn={showTimers}
      historyOn={showHistory}
      {loading}
      fanModeActive={status.fanModeActive}
      lowPelletAlarmActive={status.lowPelletAlarmActive}
      {grillConnected}
      onpower={() => canPowerOn && run(() => client.powerToggle())}
      ontimers={() => !loading && (showTimers = !showTimers)}
      onhistory={() => !loading && (showHistory = !showHistory)}
    />
    <div class="card" class:stale>
      <TempCard
        title="Grill Temp ℉"
        subtitle="Set the temperature of the grill"
        image={grillImage}
        what="Grill"
        current={status.currentGrillTemp}
        desired={status.desiredGrillTemp}
        enabled={canChangeTemp}
        min={limits.grill.min}
        max={limits.grill.max}
        onsubmit={(t) => canChangeTemp && run(() => client.setDesiredGrillTemp(t))}
      />
    </div>
    <div class="card" class:stale>
      <TempCard
        title="Food Temp ℉"
        subtitle="Set the temperature of the food"
        image={foodImage}
        what="Food"
        current={status.currentFoodTemp}
        desired={status.desiredFoodTemp}
        enabled={canChangeTemp}
        min={limits.food.min}
        max={limits.food.max}
        onsubmit={(t) => canChangeTemp && run(() => client.setDesiredFoodTemp(t))}
      />
    </div>
    {#if showTimers}<div class="card"><TimersCard /></div>{/if}
    {#if showHistory}<div class="card"><HistoryCard grill={grillPoints} food={foodPoints} /></div>{/if}
  </div>
  <BottomNav />
</div>
<Toasts />

<style>
  .app { margin-bottom: 4.5rem; }
  .container {
    display: flex;
    flex-direction: column;
    max-width: 50rem;
    margin: 1.5rem auto;
  }
  .card { margin: 1.5rem 0 0.5rem; padding: 0 2rem; }
  .card.stale { opacity: 0.5; }
  @media (max-width: 600px) {
    .card { padding: 0 0.5rem; }
  }
</style>
