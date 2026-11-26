<script lang="ts">
  import Icon from './Icon.svelte'
</script>

<nav aria-label="Main">
  <a href="/" aria-current="page"><Icon name="home" /><span>Home</span></a>
  <a href="/api"><Icon name="code" /><span>API</span></a>
</nav>

<style>
  nav {
    position: fixed;
    left: 0; right: 0; bottom: 0;
    z-index: 99999;
    display: flex;
    justify-content: center;
    height: 56px;
    background: var(--bar);
    box-shadow: var(--shadow);
  }
  a {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-width: 80px;
    max-width: 168px;
    padding: 0 12px;
    color: var(--primary);
    font-size: 0.875rem;
    text-decoration: none;
  }
</style>
