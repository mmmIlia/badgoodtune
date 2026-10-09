<script setup lang="ts">
import { reactive, watch } from 'vue'

import type { Album, CreateAlbumRequest } from '../api/generated'

const props = withDefaults(
  defineProps<{ album?: Album; busy?: boolean; submitLabel?: string; errors?: Record<string, string> }>(),
  {
    busy: false,
    submitLabel: 'Save album',
    errors: () => ({}),
  },
)

const emit = defineEmits<{ submit: [value: CreateAlbumRequest] }>()

const fields = reactive<CreateAlbumRequest>({ artist: '', title: '', releaseYear: new Date().getFullYear(), genre: '' })

watch(
  () => props.album,
  (album) => {
    if (!album) return
    fields.artist = album.artist
    fields.title = album.title
    fields.releaseYear = album.releaseYear
    fields.genre = album.genre
  },
  { immediate: true },
)

function submit() {
  emit('submit', { ...fields })
}
</script>

<template>
  <form class="form-stack" @submit.prevent="submit">
    <label>
      <span>Artist</span>
      <input v-model.trim="fields.artist" required maxlength="200" autocomplete="off" placeholder="Portishead" />
      <small v-if="errors.artist" class="field-error">{{ errors.artist }}</small>
    </label>
    <label>
      <span>Album title</span>
      <input v-model.trim="fields.title" required maxlength="200" autocomplete="off" placeholder="Dummy" />
      <small v-if="errors.title" class="field-error">{{ errors.title }}</small>
    </label>
    <div class="form-row">
      <label>
        <span>Release year</span>
        <input
          v-model.number="fields.releaseYear"
          required
          type="number"
          min="1877"
          :max="new Date().getFullYear() + 1"
        />
        <small v-if="errors.releaseYear" class="field-error">{{ errors.releaseYear }}</small>
      </label>
      <label>
        <span>Genre</span>
        <input v-model.trim="fields.genre" required maxlength="50" autocomplete="off" placeholder="Trip-hop" />
        <small v-if="errors.genre" class="field-error">{{ errors.genre }}</small>
      </label>
    </div>
    <button class="button button-accent" type="submit" :disabled="busy">
      {{ busy ? 'Saving…' : submitLabel }}
    </button>
  </form>
</template>
