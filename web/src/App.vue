<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'

import AppNotice from './components/AppNotice.vue'
import { authState, logout } from './state/auth'
import { problemMessage } from './utils/problem'

const router = useRouter()
const signOutError = ref('')

async function signOut() {
  signOutError.value = ''
  try {
    await logout()
    await router.push({ name: 'catalog' })
  } catch (cause) {
    signOutError.value = problemMessage(cause)
  }
}
</script>

<template>
  <a class="skip-link" href="#main-content">Skip to content</a>
  <header class="site-header">
    <RouterLink class="brand" :to="{ name: 'catalog' }" aria-label="BadGoodTune home">
      <span class="brand-mark" aria-hidden="true">B/G</span>
      <span>BadGoodTune</span>
    </RouterLink>
    <nav class="site-nav" aria-label="Primary navigation">
      <RouterLink :to="{ name: 'catalog' }">Discover</RouterLink>
      <template v-if="authState.user">
        <RouterLink class="button button-small button-accent" :to="{ name: 'album-create' }"
          >Add missing album</RouterLink
        >
        <span class="nav-user">@{{ authState.user.username }}</span>
        <button class="button button-small button-ghost" type="button" @click="signOut">Sign out</button>
      </template>
      <template v-else-if="authState.ready">
        <RouterLink :to="{ name: 'login' }">Sign in</RouterLink>
        <RouterLink class="button button-small" :to="{ name: 'register' }">Join</RouterLink>
      </template>
    </nav>
  </header>

  <div v-if="signOutError" class="page-shell app-notice">
    <AppNotice kind="error" :message="signOutError" />
  </div>

  <main id="main-content">
    <RouterView />
  </main>

  <footer class="site-footer">
    <p>Built for careful listening and honest disagreement.</p>
    <p class="muted">BadGoodTune · {{ new Date().getFullYear() }}</p>
  </footer>
</template>
