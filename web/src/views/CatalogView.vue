<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'

import { listAlbums } from '../api/generated'
import type { Album, ListAlbumsData } from '../api/generated'
import AlbumCard from '../components/AlbumCard.vue'
import AppNotice from '../components/AppNotice.vue'
import { authState } from '../state/auth'
import { problemMessage } from '../utils/problem'

type AlbumOrder = NonNullable<NonNullable<ListAlbumsData['query']>['orderBy']>
const pageSize = 12
const skeletonCount = 6
const maxReleaseYear = new Date().getFullYear() + 1

const albums = ref<Album[]>([])
const nextPageToken = ref<string>()
const loading = ref(true)
const loadingMore = ref(false)
const error = ref('')
const filters = reactive<{ q: string; genre: string; releaseYear: string; orderBy: AlbumOrder }>({
  q: '',
  genre: '',
  releaseYear: '',
  orderBy: 'createdAt desc',
})
let latestRequest = 0

function query(pageToken?: string): NonNullable<ListAlbumsData['query']> {
  return {
    pageSize,
    pageToken,
    q: filters.q || undefined,
    genre: filters.genre || undefined,
    releaseYear: filters.releaseYear ? Number(filters.releaseYear) : undefined,
    orderBy: filters.orderBy,
  }
}

async function searchAlbums() {
  const request = ++latestRequest
  loading.value = true
  loadingMore.value = false
  albums.value = []
  nextPageToken.value = undefined
  error.value = ''
  try {
    const response = await listAlbums({ query: query() })
    if (request !== latestRequest) return
    albums.value = response.data.albums
    nextPageToken.value = response.data.nextPageToken
  } catch (cause) {
    if (request === latestRequest) error.value = problemMessage(cause)
  } finally {
    if (request === latestRequest) loading.value = false
  }
}

async function loadMoreAlbums() {
  if (!nextPageToken.value) return
  const request = ++latestRequest
  loadingMore.value = true
  error.value = ''
  try {
    const response = await listAlbums({ query: query(nextPageToken.value) })
    if (request !== latestRequest) return
    albums.value = [...albums.value, ...response.data.albums]
    nextPageToken.value = response.data.nextPageToken
  } catch (cause) {
    if (request === latestRequest) error.value = problemMessage(cause)
  } finally {
    if (request === latestRequest) loadingMore.value = false
  }
}

function resetFilters() {
  filters.q = ''
  filters.genre = ''
  filters.releaseYear = ''
  filters.orderBy = 'createdAt desc'
  void searchAlbums()
}

onMounted(searchAlbums)
</script>

<template>
  <section class="hero page-shell">
    <p class="kicker">A living record shelf</p>
    <h1>Music is subjective.<br /><em>Arguments can still be good.</em></h1>
    <p class="hero-copy">Find an album, see its average score, and read what other listeners think.</p>
  </section>

  <section class="catalog-section page-shell" aria-labelledby="catalog-heading">
    <div class="section-heading">
      <div>
        <p class="kicker">The catalog</p>
        <h2 id="catalog-heading">Find your next opinion</h2>
      </div>
    </div>

    <form class="filter-bar" role="search" @submit.prevent="searchAlbums">
      <label class="search-field">
        <span class="sr-only">Search artist or album</span>
        <input v-model.trim="filters.q" type="search" maxlength="200" placeholder="Search artist or album…" />
      </label>
      <input v-model.trim="filters.genre" aria-label="Filter by genre" maxlength="50" placeholder="Genre" />
      <input
        v-model="filters.releaseYear"
        aria-label="Filter by release year"
        type="number"
        min="1877"
        :max="maxReleaseYear"
        placeholder="Year"
      />
      <select v-model="filters.orderBy" aria-label="Sort albums">
        <option value="createdAt desc">Newest additions</option>
        <option value="releaseYear desc">Release year</option>
        <option value="rating desc">Highest rated</option>
      </select>
      <button class="button" type="submit">Apply</button>
      <button class="text-button" type="button" @click="resetFilters">Reset</button>
    </form>

    <AppNotice v-if="error" kind="error" :message="error" />
    <div v-if="loading" class="album-grid" aria-label="Loading albums">
      <div v-for="item in skeletonCount" :key="item" class="album-card skeleton-card" />
    </div>
    <div v-else-if="albums.length" class="album-grid">
      <AlbumCard v-for="album in albums" :key="album.id" :album="album" />
    </div>
    <div v-else-if="!error" class="empty-state">
      <span class="empty-glyph">♪</span>
      <h3>No albums found</h3>
      <p>Try broader filters. If the album is missing from the catalog, add it once for everyone.</p>
      <RouterLink v-if="authState.user" class="button button-accent" :to="{ name: 'album-create' }">
        Add missing album
      </RouterLink>
    </div>
    <div v-if="nextPageToken" class="load-more">
      <button class="button button-ghost" type="button" :disabled="loadingMore" @click="loadMoreAlbums">
        {{ loadingMore ? 'Loading…' : 'Load more' }}
      </button>
    </div>
  </section>
</template>
