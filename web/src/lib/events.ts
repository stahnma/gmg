import type { GrillAlert, GrillHealth, GrillStatus } from './types'

export interface EventHandlers {
  onStatus(status: GrillStatus): void
  onAlert(alert: GrillAlert): void
  onHealth?(health: GrillHealth): void
  onConnectionChange(connected: boolean): void
}

/**
 * Subscribes to the server's live events (Server-Sent Events). The browser
 * reconnects automatically. Returns a function that closes the stream.
 */
export function connectEvents(url: string, h: EventHandlers): () => void {
  const source = new EventSource(url)
  source.onopen = () => h.onConnectionChange(true)
  source.onerror = () => h.onConnectionChange(false)
  source.addEventListener('status', (e) => h.onStatus(JSON.parse((e as MessageEvent).data)))
  source.addEventListener('connection', (e) => h.onHealth?.(JSON.parse((e as MessageEvent).data)))
  source.addEventListener('alert', (e) => h.onAlert(JSON.parse((e as MessageEvent).data)))
  return () => source.close()
}
