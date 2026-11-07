<script lang="ts">
  import Icon from './Icon.svelte'

  let {
    powerOn = false,
    timersOn = false,
    historyOn = false,
    loading = true,
    fanModeActive = false,
    lowPelletAlarmActive = false,
    grillConnected = false,
    onpower,
    ontimers,
    onhistory,
  }: {
    powerOn?: boolean
    timersOn?: boolean
    historyOn?: boolean
    loading?: boolean
    fanModeActive?: boolean
    lowPelletAlarmActive?: boolean
    grillConnected?: boolean
    onpower?: () => void
    ontimers?: () => void
    onhistory?: () => void
  } = $props()

  const buttonColor = (on: boolean) => (on ? 'var(--active)' : 'var(--idle)')
  const alert = $derived(fanModeActive ? 'Fan mode is active!' : lowPelletAlarmActive ? 'Low Pellet alarm is active!' : '')
</script>

<div class="bar">
  <div class="nav">
    <div class="item" title={fanModeActive ? 'Grill cannot be powered on during fan mode!' : undefined}>
      <button aria-label="Power" disabled={!grillConnected || fanModeActive} onclick={onpower}>
        <Icon name="power" size="2rem" color={buttonColor(powerOn)} />
      </button>
      <span>Power</span>
    </div>
    <div class="item">
      <button aria-label="Timers" class:on={timersOn} disabled={loading} onclick={ontimers}>
        <Icon name="time" size="2rem" color={buttonColor(timersOn)} />
      </button>
      <span>Timers</span>
    </div>
    <div class="item">
      <button aria-label="History" class:on={historyOn} disabled={loading} onclick={onhistory}>
        <Icon name="history" size="2rem" color={buttonColor(historyOn)} />
      </button>
      <span>History</span>
    </div>
  </div>
  <div class="status">
    <span
      role="img"
      aria-label={grillConnected ? 'Grill connected' : 'Grill not connected'}
      title={grillConnected
        ? 'The grill is connected!'
        : 'The grill is not connected! The application will continue trying to connect in the background.'}
    >
      <Icon name="wifi" color={grillConnected ? 'var(--active)' : 'var(--muted)'} />
    </span>
    <span role="img" aria-label={alert || 'No grill alerts'} title={alert || undefined}>
      <Icon name="warning" color={alert ? 'var(--warn)' : 'var(--muted)'} />
    </span>
  </div>
</div>
{#if loading}<div class="progress" role="progressbar" aria-label="Working"></div>{/if}

<style>
  .bar {
    display: flex;
    align-items: center;
    height: 5rem;
    padding: 0 24px;
    background: var(--bar);
  }
  .nav { display: flex; align-items: center; }
  .item { display: flex; flex-direction: column; align-items: center; margin-right: 42px; }
  .item span { margin-top: -10px; color: var(--idle); font-size: 14px; }
  .nav button {
    display: inline-flex;
    padding: 8px;
    border: none;
    border-radius: 50%;
    background: transparent;
    cursor: pointer;
    transition: background-color 150ms;
  }
  .nav button:hover:not(:disabled), .nav button.on { background: rgba(255, 255, 255, 0.08); }
  .nav button:disabled { cursor: default; }
  .nav button:focus-visible { outline: 2px solid var(--primary); }
  .status { display: flex; gap: 16px; margin-left: auto; padding-right: 8px; }
  .status span { display: inline-flex; }
  .progress {
    position: relative;
    height: 4px;
    overflow: hidden;
    background: rgba(250, 250, 250, 0.3);
  }
  .progress::after {
    content: '';
    position: absolute;
    top: 0; bottom: 0;
    width: 40%;
    background: var(--primary);
    animation: slide 1.6s infinite ease-in-out;
  }
  @keyframes slide { from { left: -40%; } to { left: 100%; } }
  @media (max-width: 600px) {
    .bar { padding: 0 16px; }
    .item { margin-right: 8px; }
  }
</style>
