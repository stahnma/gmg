<script lang="ts">
  import { toasts, AUTO_CLOSE_MS, type Toast } from '../lib/toasts.svelte'
  import Icon from './Icon.svelte'

  // Auto-dismiss after AUTO_CLOSE_MS; hovering pauses the timer.
  function autoClose(node: HTMLElement, t: Toast) {
    let left = AUTO_CLOSE_MS
    let started = Date.now()
    let timer = setTimeout(() => toasts.dismiss(t.id), left)
    const pause = () => {
      clearTimeout(timer)
      left -= Date.now() - started
    }
    const resume = () => {
      started = Date.now()
      timer = setTimeout(() => toasts.dismiss(t.id), left)
    }
    node.addEventListener('mouseenter', pause)
    node.addEventListener('mouseleave', resume)
    return { destroy: () => clearTimeout(timer) }
  }
</script>

<div class="toasts" aria-live="polite">
  {#each toasts.visible as t (t.id)}
    <div class="toast {t.level}" role={t.level === 'error' ? 'alert' : 'status'} use:autoClose={t}>
      <Icon name={t.level} size="20px" />
      <div class="message">{t.message}</div>
      <button aria-label="Dismiss" onclick={() => toasts.dismiss(t.id)}><Icon name="close" size="16px" /></button>
      <div class="bar" style="animation-duration: {AUTO_CLOSE_MS}ms"></div>
    </div>
  {/each}
</div>

<style>
  .toasts {
    position: fixed;
    top: 16px;
    right: 16px;
    z-index: 9999999;
    display: flex;
    flex-direction: column;
    gap: 16px;
    width: 320px;
  }
  .toast {
    position: relative;
    display: flex;
    align-items: center;
    gap: 10px;
    min-height: 64px;
    padding: 8px 28px 8px 12px;
    overflow: hidden;
    border-radius: 6px;
    color: #fff;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    animation: enter 300ms ease-out;
  }
  .info { background: #3498db; }
  .success { background: #07bc0c; }
  .warning { background: #f1c40f; }
  .error { background: #e74c3c; }
  .message { flex: 1; font-size: 1rem; }
  button {
    position: absolute;
    top: 6px;
    right: 6px;
    display: inline-flex;
    padding: 0;
    border: none;
    background: none;
    color: #fff;
    opacity: 0.7;
    cursor: pointer;
  }
  button:hover { opacity: 1; }
  .bar {
    position: absolute;
    left: 0; bottom: 0;
    width: 100%;
    height: 5px;
    background: rgba(255, 255, 255, 0.7);
    transform-origin: left;
    animation: shrink linear forwards;
  }
  .toast:hover .bar { animation-play-state: paused; }
  @keyframes shrink { to { transform: scaleX(0); } }
  @keyframes enter { from { transform: translateX(110%); } to { transform: none; } }
  @media (max-width: 480px) {
    .toasts { top: 0; right: 0; left: 0; width: 100%; gap: 0; }
    .toast { border-radius: 0; }
  }
</style>
