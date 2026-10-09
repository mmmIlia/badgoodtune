<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import AlbumHero from '../components/AlbumHero.vue'
import AppNotice from '../components/AppNotice.vue'
import ReviewCard from '../components/ReviewCard.vue'
import ReviewForm from '../components/ReviewForm.vue'
import { useAlbumDetail } from '../composables/useAlbumDetail'

const route = useRoute()
const router = useRouter()
const albumId = computed(() => String(route.params.albumId))

async function returnToCatalog() {
  await router.push({ name: 'catalog' })
}

const {
  album,
  currentUserId,
  deletingAlbum,
  editingReviewId,
  error,
  loadMoreReviews,
  loading,
  loadingMore,
  nextPageToken,
  ownReview,
  ownsAlbum,
  publishReview,
  removeAlbum,
  removeReview,
  reviewFieldErrors,
  reviews,
  saveReview,
  savingReview,
  startEditingReview,
} = useAlbumDetail(albumId, returnToCatalog)
</script>

<template>
  <section class="page-shell detail-page">
    <RouterLink class="back-link" :to="{ name: 'catalog' }">← Back to catalog</RouterLink>
    <AppNotice v-if="error" kind="error" :message="error" />
    <div v-if="loading" class="detail-loading">Loading album…</div>
    <template v-else-if="album">
      <AlbumHero :album="album" :editable="ownsAlbum" :deleting="deletingAlbum" @remove="removeAlbum" />

      <div class="reviews-layout">
        <section aria-labelledby="reviews-heading">
          <div class="section-heading compact-heading">
            <div>
              <p class="kicker">Listener notes</p>
              <h2 id="reviews-heading">Reviews</h2>
            </div>
          </div>
          <div v-if="reviews.length" class="review-list">
            <ReviewCard
              v-for="review in reviews"
              :key="review.id"
              :review="review"
              :editing="editingReviewId === review.id"
              :busy="savingReview"
              :editable="review.author.id === currentUserId"
              :errors="reviewFieldErrors"
              @edit="startEditingReview(review.id)"
              @save="saveReview(review, $event)"
              @cancel="editingReviewId = undefined"
              @remove="removeReview(review)"
            />
          </div>
          <div v-else class="empty-state compact-empty">
            <h3>No reviews yet</h3>
            <p>There is still time to form the first opinion.</p>
          </div>
          <button
            v-if="nextPageToken"
            class="button button-ghost"
            type="button"
            :disabled="loadingMore"
            @click="loadMoreReviews"
          >
            {{ loadingMore ? 'Loading…' : 'Load more reviews' }}
          </button>
        </section>

        <aside class="review-aside">
          <div class="panel sticky-panel">
            <template v-if="currentUserId && !ownReview">
              <p class="kicker">Your turn</p>
              <h2>Write a review</h2>
              <ReviewForm :busy="savingReview" :errors="reviewFieldErrors" @submit="publishReview" />
            </template>
            <template v-else-if="ownReview">
              <p class="kicker">Opinion recorded</p>
              <h2>You reviewed this album.</h2>
              <p class="muted">You can edit or delete your review in the list.</p>
            </template>
            <template v-else>
              <p class="kicker">Have a take?</p>
              <h2>Join the conversation.</h2>
              <p class="muted">Sign in to score this album and publish a review.</p>
              <RouterLink class="button button-accent" :to="{ name: 'login', query: { returnTo: route.fullPath } }"
                >Sign in to review</RouterLink
              >
            </template>
          </div>
        </aside>
      </div>
    </template>
  </section>
</template>
