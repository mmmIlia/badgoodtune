<script setup lang="ts">
import type { Album } from '../api/generated'
import { formatScore } from '../utils/format'

defineProps<{ album: Album }>()
</script>

<template>
  <article class="album-card">
    <RouterLink
      class="album-art"
      :to="{ name: 'album-detail', params: { albumId: album.id } }"
      tabindex="-1"
      aria-hidden="true"
    >
      <span>{{ album.artist.slice(0, 1) }}</span>
      <span class="album-art-ring"></span>
    </RouterLink>
    <div class="album-card-content">
      <div class="eyebrow-row">
        <span class="eyebrow">{{ album.genre }}</span>
        <span>{{ album.releaseYear }}</span>
      </div>
      <h2>
        <RouterLink :to="{ name: 'album-detail', params: { albumId: album.id } }">{{ album.title }}</RouterLink>
      </h2>
      <p class="album-artist">{{ album.artist }}</p>
      <div class="album-card-footer">
        <span class="score"
          ><strong>{{ formatScore(album.averageScore) }}</strong
          >/10</span
        >
        <span class="muted">{{ album.reviewCount }} {{ album.reviewCount === 1 ? 'review' : 'reviews' }}</span>
      </div>
    </div>
  </article>
</template>
