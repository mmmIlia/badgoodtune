<template>
  <section class="not-found page-shell">
    <p class="kicker">404 · Lost track</p>
    <h1>This side has no music.</h1>
    <p>The page may have moved, or the album was never cataloged.</p>
    <RouterLink class="button button-accent" :to="{ name: 'catalog' }">Return to catalog</RouterLink>
  </section>
</template>
