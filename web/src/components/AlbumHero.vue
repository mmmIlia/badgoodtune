<script setup lang="ts">
import type { Album } from '../api/generated'
import { formatDate, formatScore } from '../utils/format'

withDefaults(defineProps<{ album: Album; editable?: boolean; deleting?: boolean }>(), {
  editable: false,
  deleting: false,
})

defineEmits<{ remove: [] }>()
</script>

<template>
  <header class="album-hero">
    <div class="album-cover-large" aria-hidden="true">
      <span>{{ album.artist.slice(0, 1) }}</span>
      <span class="album-art-ring"></span>
    </div>
    <div class="album-hero-copy">
      <div class="eyebrow-row detail-meta">
        <span class="eyebrow">{{ album.genre }}</span
        ><span>{{ album.releaseYear }}</span>
      </div>
      <h1>{{ album.title }}</h1>
      <p class="detail-artist">{{ album.artist }}</p>
      <div class="rating-lockup">
        <span class="rating-number">{{ formatScore(album.averageScore) }}</span>
        <span
          ><strong>out of 10</strong><br /><span class="muted"
            >from {{ album.reviewCount }} {{ album.reviewCount === 1 ? 'review' : 'reviews' }}</span
          ></span
        >
      </div>
      <p class="contributor">Added by @{{ album.owner.username }} · {{ formatDate(album.createdAt) }}</p>
      <div v-if="editable" class="button-row">
        <RouterLink class="button button-small" :to="{ name: 'album-edit', params: { albumId: album.id } }"
          >Edit metadata</RouterLink
        >
        <button class="text-button danger" type="button" :disabled="deleting" @click="$emit('remove')">
          {{ deleting ? 'Deleting…' : 'Delete album' }}
        </button>
      </div>
    </div>
  </header>
</template>
