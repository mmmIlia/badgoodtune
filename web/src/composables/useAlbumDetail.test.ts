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
  vi.resetAllMocks()
})

describe('useAlbumDetail', () => {
  it('ignores stale pagination after leaving and returning to the same album', async () => {
    const stalePage = deferred<{ data: { reviews: []; nextPageToken: string } }>()
    api.getAlbum.mockResolvedValue({ data: album('first') })
    api.listReviews
      .mockResolvedValue({ data: { reviews: [] } })
      .mockResolvedValueOnce({ data: { reviews: [], nextPageToken: 'page-2' } })
      .mockReturnValueOnce(stalePage.promise)

    const albumId = ref('first')
    const scope = effectScope()
    try {
      const state = scope.run(() =>
        useAlbumDetail(
          computed(() => albumId.value),
          vi.fn(),
        ),
      )!
      await flushPromises()
      const pagination = state.loadMoreReviews()
      albumId.value = 'second'
      await nextTick()
      albumId.value = 'first'
      await nextTick()
      await flushPromises()
      expect(state.nextPageToken.value).toBeUndefined()

      stalePage.resolve({ data: { reviews: [], nextPageToken: 'stale-page' } })
      await pagination
      expect(state.nextPageToken.value).toBeUndefined()
    } finally {
      scope.stop()
    }
  })

  it('ignores an old response even after returning to the same album', async () => {
    const staleAlbum = deferred<{ data: Album }>()
    const staleReviews = deferred<{ data: { reviews: [] } }>()
    api.getAlbum.mockResolvedValue({ data: album('first') }).mockReturnValueOnce(staleAlbum.promise)
    api.listReviews.mockResolvedValue({ data: { reviews: [] } }).mockReturnValueOnce(staleReviews.promise)

    const albumId = ref('first')
    const scope = effectScope()
    try {
      const state = scope.run(() =>
        useAlbumDetail(
          computed(() => albumId.value),
          vi.fn(),
        ),
      )!
      albumId.value = 'second'
      await nextTick()
      albumId.value = 'first'
      await nextTick()
      await flushPromises()
      expect(state.album.value?.version).toBe(1)

      staleAlbum.resolve({ data: { ...album('first'), version: 0 } })
      staleReviews.resolve({ data: { reviews: [] } })
      await flushPromises()
      expect(state.album.value?.version).toBe(1)
    } finally {
      scope.stop()
    }
  })

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
