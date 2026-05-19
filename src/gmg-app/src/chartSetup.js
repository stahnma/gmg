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
