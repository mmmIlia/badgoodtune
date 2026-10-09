<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { getAlbum, updateAlbum } from '../api/generated'
import type { Album, CreateAlbumRequest } from '../api/generated'
import AlbumForm from '../components/AlbumForm.vue'
import AppNotice from '../components/AppNotice.vue'
import { authState } from '../state/auth'
import { problemFieldErrors, problemMessage } from '../utils/problem'

const route = useRoute()
const router = useRouter()
const album = ref<Album>()
const loading = ref(true)
const busy = ref(false)
const error = ref('')
const fieldErrors = ref<Record<string, string>>({})
const albumId = computed(() => String(route.params.albumId))
let operation = 0

async function load() {
  const currentOperation = ++operation
  const requestedAlbumId = albumId.value
  loading.value = true
  busy.value = false
  error.value = ''
  fieldErrors.value = {}
  album.value = undefined
  try {
    const response = await getAlbum({ path: { albumId: requestedAlbumId } })
    if (currentOperation !== operation || requestedAlbumId !== albumId.value) return
    album.value = response.data
    if (response.data.owner.id !== authState.user?.id) {
      await router.replace({ name: 'album-detail', params: { albumId: requestedAlbumId } })
    }
  } catch (cause) {
    if (currentOperation === operation && requestedAlbumId === albumId.value) error.value = problemMessage(cause)
  } finally {
    if (currentOperation === operation) loading.value = false
  }
}

async function save(body: CreateAlbumRequest) {
  if (!album.value) return
  const currentOperation = ++operation
  const requestedAlbumId = albumId.value
  const version = album.value.version
  busy.value = true
  error.value = ''
  fieldErrors.value = {}
  try {
    const response = await updateAlbum({ path: { albumId: requestedAlbumId }, body: { ...body, version } })
    if (currentOperation !== operation || requestedAlbumId !== albumId.value) return
    await router.push({ name: 'album-detail', params: { albumId: response.data.id } })
  } catch (cause) {
    if (currentOperation !== operation || requestedAlbumId !== albumId.value) return
    fieldErrors.value = problemFieldErrors(cause)
    error.value = Object.keys(fieldErrors.value).length ? '' : problemMessage(cause)
  } finally {
    if (currentOperation === operation) busy.value = false
  }
}

watch(albumId, load, { immediate: true })
</script>

<template>
  <section class="narrow-page page-shell">
    <RouterLink class="back-link" :to="{ name: 'album-detail', params: { albumId } }">← Back to album</RouterLink>
    <p class="kicker">Catalog maintenance</p>
    <h1>Edit album</h1>
    <AppNotice v-if="error" kind="error" :message="error" />
    <div v-if="loading" class="panel">Loading album…</div>
    <div v-else-if="album" class="panel form-panel">
      <AlbumForm :album="album" :busy="busy" :errors="fieldErrors" @submit="save" />
    </div>
  </section>
</template>
