// Chart.js v3 + chartjs-plugin-streaming v2 + luxon adapter registration.
//
// Chart.js v3+ is tree-shakable: nothing renders until the components it
// needs are explicitly registered. Importing this module once (from
// src/index.js, before any chart renders) wires up everything the app uses.
import {
  Chart as ChartJS,
  LineElement,
  PointElement,
  LineController,
  LinearScale,
  TimeScale,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js'
import StreamingPlugin, { RealTimeScale } from 'chartjs-plugin-streaming'
// Side-effect import: registers itself with Chart.js's date-adapter slot so
// `time.unit: 'minute'` and friends actually parse/format timestamps.
import 'chartjs-adapter-luxon'

ChartJS.register(
  LineElement,
  PointElement,
  LineController,
  LinearScale,
  TimeScale,
  RealTimeScale,
  Tooltip,
  Legend,
  Filler,
  StreamingPlugin,
)

// Chart.js defaults assume a light page background — dark text (#666) and
// near-black gridlines (rgba(0,0,0,0.1)) — which render as low-contrast smudges
// on the MUI dark theme's grey[800] card. Re-pin them for dark mode so axis
// ticks, the legend, and gridlines are actually legible. Tooltips already use
// a dark backdrop with white text out of the box, so leave those alone.
ChartJS.defaults.color = 'rgba(255, 255, 255, 0.87)'
ChartJS.defaults.borderColor = 'rgba(255, 255, 255, 0.15)'
