<script setup lang="ts">
import { reactive, watch } from 'vue'

import type { CreateReviewRequest, Review } from '../api/generated'

const props = withDefaults(
  defineProps<{ review?: Review; busy?: boolean; submitLabel?: string; errors?: Record<string, string> }>(),
  {
    busy: false,
    submitLabel: 'Publish review',
    errors: () => ({}),
  },
)
const emit = defineEmits<{ submit: [value: CreateReviewRequest]; cancel: [] }>()

const fields = reactive<CreateReviewRequest>({ headline: '', body: '', score: 8 })

watch(
  () => props.review,
  (review) => {
    if (!review) return
    fields.headline = review.headline
    fields.body = review.body
    fields.score = review.score
  },
  { immediate: true },
)
</script>

<template>
  <form class="form-stack review-form" @submit.prevent="emit('submit', { ...fields })">
    <label>
      <span>Headline</span>
      <input v-model.trim="fields.headline" required maxlength="120" placeholder="A record that rewards patience" />
      <small v-if="errors.headline" class="field-error">{{ errors.headline }}</small>
    </label>
    <label>
      <span>Your review</span>
      <textarea
        v-model.trim="fields.body"
        required
        maxlength="10000"
        rows="7"
        placeholder="What works, what does not, and why?"
      />
      <small v-if="errors.body" class="field-error">{{ errors.body }}</small>
    </label>
    <label class="score-field">
      <span
        >Score: <strong>{{ fields.score }}/10</strong></span
      >
      <input v-model.number="fields.score" type="range" min="1" max="10" step="1" />
      <small v-if="errors.score" class="field-error">{{ errors.score }}</small>
    </label>
    <div class="button-row">
      <button class="button button-accent" type="submit" :disabled="busy">{{ busy ? 'Saving…' : submitLabel }}</button>
      <button v-if="review" class="button button-ghost" type="button" :disabled="busy" @click="emit('cancel')">
        Cancel
      </button>
    </div>
  </form>
</template>
