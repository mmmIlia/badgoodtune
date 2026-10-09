<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import AppNotice from '../components/AppNotice.vue'
import { register } from '../state/auth'
import { returnPath } from '../utils/navigation'
import { problemFieldErrors, problemMessage } from '../utils/problem'

const route = useRoute()
const router = useRouter()
const username = ref('')
const password = ref('')
const confirmation = ref('')
const busy = ref(false)
const error = ref('')
const fieldErrors = ref<Record<string, string>>({})
const passwordsMatch = computed(() => !confirmation.value || password.value === confirmation.value)

async function submit() {
  if (password.value !== confirmation.value) {
    error.value = 'Passwords do not match.'
    return
  }
  busy.value = true
  error.value = ''
  fieldErrors.value = {}
  try {
    await register(username.value, password.value)
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
      <p class="kicker">Join the argument</p>
      <h1>Good taste is earned one listen at a time.</h1>
      <p>Your username is public. Your password never leaves the sign-up request and is stored as an Argon2id hash.</p>
    </div>
    <div class="panel auth-panel">
      <h2>Create account</h2>
      <AppNotice v-if="error" kind="error" :message="error" />
      <form class="form-stack" @submit.prevent="submit">
        <label>
          <span>Username</span>
          <input
            v-model.trim="username"
            required
            minlength="3"
            maxlength="32"
            pattern="[A-Za-z0-9_]+"
            autocomplete="username"
            autofocus
          />
          <small>3–32 letters, digits, or underscores.</small>
          <small v-if="fieldErrors.username" class="field-error">{{ fieldErrors.username }}</small>
        </label>
        <label>
          <span>Password</span>
          <input
            v-model="password"
            required
            minlength="12"
            maxlength="128"
            type="password"
            autocomplete="new-password"
          />
          <small>At least 12 characters.</small>
          <small v-if="fieldErrors.password" class="field-error">{{ fieldErrors.password }}</small>
        </label>
        <label>
          <span>Confirm password</span>
          <input
            v-model="confirmation"
            required
            minlength="12"
            maxlength="128"
            type="password"
            autocomplete="new-password"
          />
          <small v-if="!passwordsMatch" class="field-error">Passwords do not match.</small>
        </label>
        <button class="button button-accent" type="submit" :disabled="busy || !passwordsMatch">
          {{ busy ? 'Creating…' : 'Create account' }}
        </button>
      </form>
      <p class="auth-switch">
        Already a member? <RouterLink :to="{ name: 'login', query: route.query }">Sign in</RouterLink>
      </p>
    </div>
  </section>
</template>
