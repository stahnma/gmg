<script lang="ts">
  import { icons, type IconName } from '../lib/icons'

  let { name, size = '1.5rem', color = 'currentColor' }: { name: IconName; size?: string; color?: string } = $props()
</script>

<svg viewBox="0 0 24 24" width={size} height={size} fill={color} aria-hidden="true" focusable="false">
  {#each icons[name] as d}
    <path {d} />
  {/each}
</svg>

<style>
  svg { display: inline-block; flex-shrink: 0; transition: fill 200ms; }
</style>
