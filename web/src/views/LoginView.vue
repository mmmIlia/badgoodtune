<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import AppNotice from '../components/AppNotice.vue'
import { login } from '../state/auth'
import { returnPath } from '../utils/navigation'
import { problemFieldErrors, problemMessage } from '../utils/problem'

const route = useRoute()
const router = useRouter()
const username = ref('')
const password = ref('')
const busy = ref(false)
const error = ref('')
const fieldErrors = ref<Record<string, string>>({})

async function submit() {
  busy.value = true
  error.value = ''
  fieldErrors.value = {}
  try {
    await login(username.value, password.value)
    await router.push(returnPath(route.query.returnTo))
  } catch (cause) {
    fieldErrors.value = problemFieldErrors(cause)
    error.value = Object.keys(fieldErrors.value).length ? '' : problemMessage(cause)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <section class="auth-layout page-shell">
    <div class="auth-pitch">
      <p class="kicker">Welcome back</p>
      <h1>Pick up where the needle left off.</h1>
      <p>Sign in to add missing albums, publish reviews, and manage what you wrote.</p>
    </div>
    <div class="panel auth-panel">
      <h2>Sign in</h2>
      <AppNotice v-if="error" kind="error" :message="error" />
      <form class="form-stack" @submit.prevent="submit">
        <label>
          <span>Username</span>
          <input v-model.trim="username" required maxlength="32" autocomplete="username" autofocus />
          <small v-if="fieldErrors.username" class="field-error">{{ fieldErrors.username }}</small>
        </label>
        <label>
          <span>Password</span>
          <input v-model="password" required maxlength="128" type="password" autocomplete="current-password" />
          <small v-if="fieldErrors.password" class="field-error">{{ fieldErrors.password }}</small>
        </label>
        <button class="button button-accent" type="submit" :disabled="busy">
          {{ busy ? 'Signing in…' : 'Sign in' }}
        </button>
      </form>
      <p class="auth-switch">
        New here? <RouterLink :to="{ name: 'register', query: route.query }">Create an account</RouterLink>
      </p>
    </div>
  </section>
</template>
