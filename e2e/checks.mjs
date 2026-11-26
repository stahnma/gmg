// End-to-end checks against a running server + emulator.
// usage: node checks.mjs <baseUrl> <outDir> <audio: on|off>
import { chromium } from 'playwright'

const [base, out, audio] = process.argv.slice(2)
const failures = []
const check = (ok, what) => {
  console.log(`${ok ? 'ok  ' : 'FAIL'}  ${what}`)
  if (!ok) failures.push(what)
}
const api = (method, path) => fetch(base + path, { method }).then(async (r) => ({ status: r.status, body: await r.text() }))

// --- API ---
let r = await api('GET', '/api/status')
check(r.status === 200 && JSON.parse(r.body).state === 'off', 'GET /api/status: grill starts off')
r = await api('PUT', '/api/temperature/grill/225')
check(r.status === 400 && r.body === 'Cannot set grill temperature when the grill is off!', 'set temperature while off -> 400')
check((await api('GET', '/api/nope')).status === 404, 'unknown route -> 404')

// --- Browser ---
const browser = await chromium.launch()
const page = await browser.newPage({ viewport: { width: 1280, height: 900 } })
const problems = []
// With audio on, the missing sound file (none ship) is an expected 404 + warning.
const expected = (t) => audio === 'on' && (t.includes('Unable to play alert sound') || t.includes('404'))
page.on('console', (m) => ['error', 'warning'].includes(m.type()) && !expected(m.text()) && problems.push(`console.${m.type()}: ${m.text()}`))
page.on('pageerror', (e) => problems.push(`pageerror: ${e.message}`))
await page.addInitScript(() => {
  window.__audio = []
  const play = HTMLMediaElement.prototype.play
  HTMLMediaElement.prototype.play = function () {
    const entry = { src: this.src }
    window.__audio.push(entry)
    return play.call(this).then(() => (entry.result = 'playing'), (e) => { entry.result = e.name; throw e })
  }
})

await page.goto(base, { waitUntil: 'networkidle' })
await page.getByText('Grill Temp').first().waitFor()
await page.getByRole('img', { name: 'Grill connected' }).waitFor({ timeout: 20000 })
check(true, 'UI loads and receives live status')

await page.getByRole('button', { name: 'Power' }).click()
await page.getByRole('button', { name: 'Set Grill Temperature' }).and(page.locator(':enabled')).waitFor({ timeout: 20000 })
check(true, 'power on enables the temperature controls')
await page.screenshot({ path: `${out}/1-home.png`, fullPage: true })

await page.getByRole('button', { name: 'History' }).click()
await page.getByText('View grilling temperature history.').waitFor()
await page.screenshot({ path: `${out}/2-history.png`, fullPage: true })

await page.getByRole('button', { name: 'Set Grill Temperature' }).click()
const dialog = page.getByRole('dialog')
await dialog.getByRole('textbox').fill('501')
check(await dialog.getByText('Desired temperature must be less than 500 ℉!').isVisible(), 'dialog validates input')
check(await dialog.getByRole('button', { name: 'Set' }).isDisabled(), 'Set is disabled while invalid')
await page.screenshot({ path: `${out}/3-dialog.png` })
await dialog.getByRole('textbox').fill('110')
await dialog.getByRole('button', { name: 'Set' }).click()
await page.getByText('Desired: 110 ℉').waitFor({ timeout: 20000 })
check(true, 'set grill temperature to 110')

await page.getByRole('button', { name: 'Timers' }).click()
await page.getByRole('button', { name: 'Start timer' }).click()
await page.waitForTimeout(2200)
check(!(await page.getByText('Timer: 00:00:00:00').isVisible()), 'stopwatch runs')
await page.screenshot({ path: `${out}/4-timers.png`, fullPage: true })

await page.getByText('Target Grill Temperature Reached').first().waitFor({ timeout: 60000 })
check(true, 'alert notification appears when the target is reached')
await page.waitForTimeout(1500)
const plays = await page.evaluate(() => window.__audio)
// No sound files ship, so playback fails; check only that it was attempted.
if (audio === 'on') check(plays.length === 1 && plays[0].src.endsWith('/alerts/targetGrillTempReached.mp3'), `alert sound is requested (${JSON.stringify(plays)})`)
else check(plays.length === 0, `no alert sound when audio is disabled (${JSON.stringify(plays)})`)
await page.screenshot({ path: `${out}/5-alert.png` })

// --- Mobile layout: nothing overflows or is clipped ---
for (const width of [320, 390, 768]) {
  await page.setViewportSize({ width, height: 844 })
  await page.waitForTimeout(300)
  const issues = await page.evaluate(() => {
    const vw = document.documentElement.clientWidth
    const found = []
    if (document.documentElement.scrollWidth > vw) found.push(`page scrolls horizontally (${document.documentElement.scrollWidth} > ${vw})`)
    for (const b of document.querySelectorAll('button')) {
      const r = b.getBoundingClientRect()
      if (!r.width || b.closest('dialog')) continue
      let el = b.parentElement, clip = null
      while (el) { const s = getComputedStyle(el); if (s.overflow !== 'visible') { clip = el.getBoundingClientRect(); break } el = el.parentElement }
      const right = Math.min(vw, clip ? clip.right : vw), left = Math.max(0, clip ? clip.left : 0)
      if (r.right > right + 1 || r.left < left - 1) found.push(`clipped button "${(b.innerText || b.getAttribute('aria-label')).trim()}"`)
    }
    return found
  })
  check(issues.length === 0, `no overflow at ${width}px ${issues.join('; ')}`)
  if (width === 390) await page.screenshot({ path: `${out}/6-mobile.png`, fullPage: true })
}

check(problems.length === 0, `no browser console errors ${problems.join('; ')}`)
await browser.close()

// --- History persisted ---
const rows = JSON.parse((await api('GET', '/api/temperature/history?since=0')).body)
check(rows.length > 0 && rows.at(-1).grill_temperature >= 100, `history recorded (${rows.length} rows)`)

console.log(failures.length ? `\n${failures.length} check(s) failed` : '\nall checks passed')
process.exit(failures.length ? 1 : 0)
