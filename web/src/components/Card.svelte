<script lang="ts">
  import type { Snippet } from 'svelte'

  let { image, title, subtitle, children }: { image: string; title: string; subtitle: string; children: Snippet } = $props()
</script>

<section class="card" aria-label={title}>
  <div class="header">
    <img src={image} alt="" />
    <div class="overlay">
      <h2>{title}</h2>
      <p>{subtitle}</p>
    </div>
  </div>
  {@render children()}
</section>

<style>
  .card {
    background: var(--paper);
    border-radius: 4px;
    box-shadow: var(--shadow);
    overflow: hidden;
  }
  .header { position: relative; }
  img {
    display: block;
    width: 100%;
    /* the banners are ~4.4:1; keep the title overlay from outgrowing them on phones */
    min-height: 7rem;
    object-fit: cover;
  }
  .overlay {
    position: absolute;
    left: 0; right: 0; bottom: 0;
    padding: 8px 16px;
    background: rgba(0, 0, 0, 0.54);
  }
  h2 { margin: 0; color: #fff; font-size: 1.5rem; font-weight: 400; line-height: 1.334; }
  p { margin: 0; color: var(--text-secondary); font-size: 0.875rem; line-height: 1.43; letter-spacing: 0.01071em; }
</style>
