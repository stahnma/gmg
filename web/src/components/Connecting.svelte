<script lang="ts">
  let { message = 'Connecting' }: { message?: string } = $props()
  let dots = $state(3)

  $effect(() => {
    const t = setInterval(() => (dots = dots === 3 ? 0 : dots + 1), 1000)
    return () => clearInterval(t)
  })
</script>

<div class="box" role="status" aria-live="polite">
  <div class="container"><span class="text">{message} {'.'.repeat(dots)}</span></div>
</div>

<style>
  .box {
    position: fixed;
    inset: 0;
    z-index: 999999;
    background: black;
    opacity: 0.8;
  }
  .container {
    position: relative;
    top: 50%;
    background: #232323;
    opacity: 0.95;
    border-top: 2px solid gray;
    border-bottom: 2px solid gray;
    border-radius: 2px;
  }
  .text {
    position: relative;
    left: calc(50% - 100px);
    display: block;
    max-width: 200px;
    color: white;
    font-size: xx-large;
    font-weight: bold;
  }
</style>
