<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Chart, LineController, LineElement, PointElement, LinearScale, TimeScale, Legend, Tooltip,
    type ChartDataset,
  } from 'chart.js'
  import 'chartjs-adapter-date-fns'
  import Card from './Card.svelte'
  import image from '../assets/history.png'

  Chart.register(LineController, LineElement, PointElement, LinearScale, TimeScale, Legend, Tooltip)

  import type { Point } from '../lib/types'

  let { grill, food }: { grill: Point[]; food: Point[] } = $props()

  const WINDOW_MS = 30 * 60 * 1000
  const TEXT = 'rgba(255, 255, 255, 0.7)'
  const GRID = 'rgba(255, 255, 255, 0.1)'

  let canvas: HTMLCanvasElement
  let chart: Chart<'line', Point[]> | undefined

  const dataset = (label: string, rgb: string, data: Point[]): ChartDataset<'line', Point[]> => ({
    label,
    data,
    borderColor: `rgb(${rgb})`,
    backgroundColor: `rgba(${rgb}, 0.5)`,
    fill: false,
    pointRadius: 0,
  })

  onMount(() => {
    chart = new Chart<'line', Point[]>(canvas, {
      type: 'line',
      data: { datasets: [dataset('Grill Temp', '255, 99, 132', []), dataset('Food Temp', '54, 162, 235', [])] },
      options: {
        animation: false,
        scales: {
          x: {
            type: 'time',
            time: { unit: 'minute', displayFormats: { minute: 'h:mm aaa' } },
            grid: { color: GRID },
            ticks: { color: TEXT, maxRotation: 0, stepSize: 5 },
          },
          y: { grid: { color: GRID }, ticks: { color: TEXT } },
        },
        plugins: {
          tooltip: { mode: 'nearest', intersect: false },
          legend: { labels: { color: TEXT } },
        },
        hover: { mode: 'nearest', intersect: false },
      },
    })
    return () => chart?.destroy()
  })

  // Show the last 30 minutes, moving with each new reading.
  $effect(() => {
    const g = grill.slice()
    const f = food.slice()
    if (!chart) return
    const now = Date.now()
    chart.data.datasets[0].data = g
    chart.data.datasets[1].data = f
    chart.options.scales!.x!.min = now - WINDOW_MS
    chart.options.scales!.x!.max = now
    chart.update('none')
  })
</script>

<Card {image} title="Temperature History" subtitle="View grilling temperature history.">
  <div class="chart" role="img" aria-label="Grill and food temperature over the last 30 minutes"><canvas bind:this={canvas}></canvas></div>
</Card>

<style>
  .chart { padding: 16px; }
</style>
