<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'

import { createAlbum } from '../api/generated'
import type { CreateAlbumRequest } from '../api/generated'
import AlbumForm from '../components/AlbumForm.vue'
import AppNotice from '../components/AppNotice.vue'
import { problemFieldErrors, problemMessage } from '../utils/problem'

const router = useRouter()
const busy = ref(false)
const error = ref('')
const fieldErrors = ref<Record<string, string>>({})

async function save(body: CreateAlbumRequest) {
  busy.value = true
  error.value = ''
  fieldErrors.value = {}
  try {
    const response = await createAlbum({ body })
    await router.push({ name: 'album-detail', params: { albumId: response.data.id } })
  } catch (cause) {
    fieldErrors.value = problemFieldErrors(cause)
    error.value = Object.keys(fieldErrors.value).length ? '' : problemMessage(cause)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <section class="narrow-page page-shell">
    <RouterLink class="back-link" :to="{ name: 'catalog' }">← Back to catalog</RouterLink>
    <p class="kicker">Catalog contribution</p>
    <h1>Add a missing album</h1>
    <p class="page-intro">Search the catalog first. Add only missing albums; write your opinion on the album page.</p>
    <div class="panel form-panel">
      <AppNotice v-if="error" kind="error" :message="error" />
      <AlbumForm :busy="busy" :errors="fieldErrors" submit-label="Add to catalog" @submit="save" />
    </div>
  </section>
</template>
