import { flushPromises } from '@vue/test-utils'
import { computed, effectScope, nextTick, ref } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { Album } from '../api/generated'

const api = vi.hoisted(() => ({
  createReview: vi.fn(),
  deleteAlbum: vi.fn(),
  deleteReview: vi.fn(),
  getAlbum: vi.fn(),
  listReviews: vi.fn(),
  updateReview: vi.fn(),
}))

vi.mock('../api/generated', () => api)

import { useAlbumDetail } from './useAlbumDetail'

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((resolvePromise) => {
    resolve = resolvePromise
  })
  return { promise, resolve }
}

function album(id: string): Album {
  return {
    id,
    owner: { id: `owner-${id}`, username: `owner-${id}`, createdAt: '2026-10-02T10:00:00Z' },
    artist: `Artist ${id}`,
    title: `Album ${id}`,
    releaseYear: 2026,
    genre: 'Test',
    reviewCount: 0,
    version: 1,
    createdAt: '2026-10-02T10:00:00Z',
    updatedAt: '2026-10-02T10:00:00Z',
  }
}

afterEach(() => {
  vi.clearAllMocks()
})

describe('useAlbumDetail', () => {
  it('ignores a page response after the route changes', async () => {
    const firstAlbum = deferred<{ data: Album }>()
    const secondAlbum = deferred<{ data: Album }>()
    const firstReviews = deferred<{ data: { reviews: [] } }>()
    const secondReviews = deferred<{ data: { reviews: [] } }>()

    api.getAlbum.mockImplementation(({ path }: { path: { albumId: string } }) =>
      path.albumId === 'first' ? firstAlbum.promise : secondAlbum.promise,
    )
    api.listReviews.mockImplementation(({ path }: { path: { albumId: string } }) =>
      path.albumId === 'first' ? firstReviews.promise : secondReviews.promise,
    )

    const albumId = ref('first')
    const scope = effectScope()
    const state = scope.run(() =>
      useAlbumDetail(
        computed(() => albumId.value),
        vi.fn(),
      ),
    )!

    albumId.value = 'second'
    await nextTick()
    secondAlbum.resolve({ data: album('second') })
    secondReviews.resolve({ data: { reviews: [] } })
    await flushPromises()
    expect(state.album.value?.id).toBe('second')

    firstAlbum.resolve({ data: album('first') })
    firstReviews.resolve({ data: { reviews: [] } })
    await flushPromises()
    expect(state.album.value?.id).toBe('second')

    scope.stop()
  })
})
