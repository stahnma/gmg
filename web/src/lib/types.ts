// Shapes returned by the Go server (same as the original Node API).

export interface GrillStatus {
  state: 'off' | 'on' | 'fan mode' | 'unknown'
  isOn: boolean
  currentGrillTemp: number
  desiredGrillTemp: number
  currentFoodTemp: number
  desiredFoodTemp: number
  fanModeActive: boolean
  lowPelletAlarmActive: boolean
}

/** Whether the grill is answering the server's polls (`connection` event). */
export interface GrillHealth {
  reachable: boolean
  lastSeen: string | null // RFC 3339; null until the first good status
  consecutiveFailures: number
  error?: string
}

export interface GrillAlert {
  type: string
  name: string
  reason: string
  beep?: string // sound file, e.g. alerts/lowPelletAlarmActive.mp3; omitted when audio is disabled
  level: 'info' | 'warning' | 'error'
}

export interface HistoryRow {
  temperature_log_id: number
  timestamp: number // Unix seconds
  grill_temperature: number
  food_temperature: number | null
}

/** A chart point: time in ms, temperature in °F. */
export interface Point {
  x: number
  y: number | null
}
