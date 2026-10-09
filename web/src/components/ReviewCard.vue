<script setup lang="ts">
import type { CreateReviewRequest, Review } from '../api/generated'
import { formatDate } from '../utils/format'
import ReviewForm from './ReviewForm.vue'

withDefaults(
  defineProps<{
    review: Review
    editing?: boolean
    busy?: boolean
    editable?: boolean
    errors?: Record<string, string>
  }>(),
  {
    editing: false,
    busy: false,
    editable: false,
    errors: () => ({}),
  },
)

defineEmits<{
  edit: []
  save: [value: CreateReviewRequest]
  cancel: []
  remove: []
}>()
</script>

<template>
  <article class="review-card">
    <ReviewForm
      v-if="editing"
      :review="review"
      :busy="busy"
      :errors="errors"
      submit-label="Update review"
      @submit="$emit('save', $event)"
      @cancel="$emit('cancel')"
    />
    <template v-else>
      <header class="review-card-header">
        <div>
          <span class="review-score">{{ review.score }}</span
          ><span class="muted">/10</span>
        </div>
        <div class="review-byline">
          <strong>@{{ review.author.username }}</strong
          ><span>{{ formatDate(review.createdAt) }}</span>
        </div>
      </header>
      <h3>{{ review.headline }}</h3>
      <p class="review-body">{{ review.body }}</p>
      <div v-if="editable" class="button-row review-actions">
        <button class="text-button" type="button" @click="$emit('edit')">Edit</button>
        <button class="text-button danger" type="button" @click="$emit('remove')">Delete</button>
      </div>
    </template>
  </article>
</template>
