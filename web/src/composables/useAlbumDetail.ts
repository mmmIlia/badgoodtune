import { computed, ref, watch, type ComputedRef } from 'vue'

import { createReview, deleteAlbum, deleteReview, getAlbum, listReviews, updateReview } from '../api/generated'
import type { Album, CreateReviewRequest, Review } from '../api/generated'
import { authState } from '../state/auth'
import { problemFieldErrors, problemMessage } from '../utils/problem'

const reviewsPageSize = 10

export function useAlbumDetail(albumId: ComputedRef<string>, onAlbumDeleted: () => void | Promise<void>) {
  const album = ref<Album>()
  const reviews = ref<Review[]>([])
  const nextPageToken = ref<string>()
  const loading = ref(true)
  const loadingMore = ref(false)
  const savingReview = ref(false)
  const deletingAlbum = ref(false)
  const editingReviewId = ref<string>()
  const error = ref('')
  const reviewFieldErrors = ref<Record<string, string>>({})
  let reviewOperation = 0
  let albumOperation = 0

  const currentUserId = computed(() => authState.user?.id)
  const ownsAlbum = computed(() => album.value?.owner.id === currentUserId.value)
  const ownReview = computed(() => reviews.value.find((review) => review.author.id === currentUserId.value))

  function isCurrentAlbum(requestedAlbumId: string) {
    return requestedAlbumId === albumId.value
  }

  function isCurrentReviewOperation(operation: number, requestedAlbumId: string) {
    return operation === reviewOperation && isCurrentAlbum(requestedAlbumId)
  }

  function clearReviewErrors() {
    error.value = ''
    reviewFieldErrors.value = {}
  }

  function setReviewError(cause: unknown) {
    reviewFieldErrors.value = problemFieldErrors(cause)
    error.value = Object.keys(reviewFieldErrors.value).length ? '' : problemMessage(cause)
  }

  async function refreshAlbum(operation: number, requestedAlbumId: string, failureMessage: string) {
    try {
      const response = await getAlbum({ path: { albumId: requestedAlbumId } })
      if (isCurrentReviewOperation(operation, requestedAlbumId)) album.value = response.data
    } catch (cause) {
      if (isCurrentReviewOperation(operation, requestedAlbumId)) {
        error.value = `${failureMessage} ${problemMessage(cause)}`
      }
    }
  }

  async function loadMoreReviews() {
    if (loadingMore.value || !nextPageToken.value) return
    const requestedAlbumId = albumId.value
    const pageToken = nextPageToken.value
    loadingMore.value = true
    error.value = ''
    try {
      const response = await listReviews({
        path: { albumId: requestedAlbumId },
        query: { pageSize: reviewsPageSize, pageToken, orderBy: 'createdAt desc' },
      })
      if (!isCurrentAlbum(requestedAlbumId)) return
      reviews.value = [...reviews.value, ...response.data.reviews]
      nextPageToken.value = response.data.nextPageToken
    } catch (cause) {
      if (isCurrentAlbum(requestedAlbumId)) error.value = problemMessage(cause)
    } finally {
      if (isCurrentAlbum(requestedAlbumId)) loadingMore.value = false
    }
  }

  async function loadPage() {
    const requestedAlbumId = albumId.value
    reviewOperation += 1
    albumOperation += 1
    loading.value = true
    loadingMore.value = false
    savingReview.value = false
    deletingAlbum.value = false
    editingReviewId.value = undefined
    clearReviewErrors()
    album.value = undefined
    reviews.value = []
    nextPageToken.value = undefined
    try {
      const [albumResponse, reviewsResponse] = await Promise.all([
        getAlbum({ path: { albumId: requestedAlbumId } }),
        listReviews({
          path: { albumId: requestedAlbumId },
          query: { pageSize: reviewsPageSize, orderBy: 'createdAt desc' },
        }),
      ])
      if (!isCurrentAlbum(requestedAlbumId)) return
      album.value = albumResponse.data
      reviews.value = reviewsResponse.data.reviews
      nextPageToken.value = reviewsResponse.data.nextPageToken
    } catch (cause) {
      if (isCurrentAlbum(requestedAlbumId)) error.value = problemMessage(cause)
    } finally {
      if (isCurrentAlbum(requestedAlbumId)) loading.value = false
    }
  }

  async function publishReview(body: CreateReviewRequest) {
    const operation = ++reviewOperation
    const requestedAlbumId = albumId.value
    savingReview.value = true
    clearReviewErrors()
    try {
      const response = await createReview({ path: { albumId: requestedAlbumId }, body })
      if (!isCurrentReviewOperation(operation, requestedAlbumId)) return
      reviews.value = [response.data, ...reviews.value]
      await refreshAlbum(
        operation,
        requestedAlbumId,
        'The review was published, but the album rating was not refreshed.',
      )
    } catch (cause) {
      if (isCurrentReviewOperation(operation, requestedAlbumId)) setReviewError(cause)
    } finally {
      if (operation === reviewOperation) savingReview.value = false
    }
  }

  async function saveReview(review: Review, body: CreateReviewRequest) {
    const operation = ++reviewOperation
    const requestedAlbumId = albumId.value
    savingReview.value = true
    clearReviewErrors()
    try {
      const response = await updateReview({
        path: { albumId: requestedAlbumId, reviewId: review.id },
        body: { ...body, version: review.version },
      })
      if (!isCurrentReviewOperation(operation, requestedAlbumId)) return
      reviews.value = reviews.value.map((item) => (item.id === review.id ? response.data : item))
      editingReviewId.value = undefined
      await refreshAlbum(operation, requestedAlbumId, 'The review was saved, but the album rating was not refreshed.')
    } catch (cause) {
      if (isCurrentReviewOperation(operation, requestedAlbumId)) setReviewError(cause)
    } finally {
      if (operation === reviewOperation) savingReview.value = false
    }
  }

  function startEditingReview(reviewId: string) {
    clearReviewErrors()
    editingReviewId.value = reviewId
  }

  async function removeReview(review: Review) {
    if (!window.confirm('Delete this review? This cannot be undone.')) return
    const operation = ++reviewOperation
    const requestedAlbumId = albumId.value
    savingReview.value = true
    clearReviewErrors()
    try {
      await deleteReview({
        path: { albumId: requestedAlbumId, reviewId: review.id },
        query: { version: review.version },
      })
      if (!isCurrentReviewOperation(operation, requestedAlbumId)) return
      reviews.value = reviews.value.filter((item) => item.id !== review.id)
      await refreshAlbum(operation, requestedAlbumId, 'The review was deleted, but the album rating was not refreshed.')
    } catch (cause) {
      if (isCurrentReviewOperation(operation, requestedAlbumId)) error.value = problemMessage(cause)
    } finally {
      if (operation === reviewOperation) savingReview.value = false
    }
  }

  async function removeAlbum() {
    if (!album.value || !window.confirm('Delete this album from the catalog?')) return
    const operation = ++albumOperation
    const requestedAlbum = album.value
    deletingAlbum.value = true
    error.value = ''
    try {
      await deleteAlbum({ path: { albumId: requestedAlbum.id }, query: { version: requestedAlbum.version } })
      if (operation !== albumOperation || requestedAlbum.id !== album.value?.id) return
      await onAlbumDeleted()
    } catch (cause) {
      if (operation === albumOperation) error.value = problemMessage(cause)
    } finally {
      if (operation === albumOperation) deletingAlbum.value = false
    }
  }

  watch(albumId, loadPage, { immediate: true })

  return {
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
  }
}
