import type { HistoryRow } from './types'

const trimSlash = (uri: string) => uri.replace(/[/\\]$/, '')

/** HTTP client for the grill API. Failed requests throw with the server's message. */
export class GrillClient {
  readonly base: string

  constructor(base: string) {
    this.base = trimSlash(base)
  }

  powerToggle() {
    return this.put('/api/powertoggle')
  }

  setDesiredGrillTemp(temperature: number | string) {
    return this.put(`/api/temperature/grill/${temperature}`)
  }

  setDesiredFoodTemp(temperature: number | string) {
    return this.put(`/api/temperature/food/${temperature}`)
  }

  async getTemperatureHistory(since: number): Promise<HistoryRow[]> {
    const res = await fetch(`${this.base}/api/temperature/history?since=${since}`)
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  }

  private async put(path: string) {
    const res = await fetch(`${this.base}${path}`, { method: 'PUT' })
    if (!res.ok) throw new Error(await res.text())
  }
}
