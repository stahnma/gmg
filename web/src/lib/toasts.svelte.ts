export type ToastLevel = 'info' | 'warning' | 'error' | 'success'

export interface Toast {
  id: number
  level: ToastLevel
  message: string
}

const MAX_VISIBLE = 3
export const AUTO_CLOSE_MS = 5000

let nextId = 1

/** Notification queue: at most three visible, the rest wait their turn. */
class Toasts {
  visible = $state<Toast[]>([])
  private queue: Toast[] = []

  show(level: ToastLevel, message: string) {
    const t = { id: nextId++, level, message }
    if (this.visible.length < MAX_VISIBLE) this.visible.push(t)
    else this.queue.push(t)
    return t.id
  }

  dismiss(id: number) {
    this.visible = this.visible.filter((t) => t.id !== id)
    const next = this.queue.shift()
    if (next) this.visible.push(next)
  }

  clear() {
    this.visible = []
    this.queue = []
  }

  info = (m: string) => this.show('info', m)
  warning = (m: string) => this.show('warning', m)
  error = (m: string) => this.show('error', m)
}

export const toasts = new Toasts()
