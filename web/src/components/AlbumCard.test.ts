import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import type { Album } from '../api/generated'
import AlbumCard from './AlbumCard.vue'

const album: Album = {
  id: '6ba7b811-9dad-11d1-80b4-00c04fd430c8',
  owner: {
    id: '6ba7b810-9dad-11d1-80b4-00c04fd430c8',
    username: 'listener_1',
    createdAt: '2026-10-02T10:00:00Z',
  },
  artist: 'Massive Attack',
  title: 'Mezzanine',
  releaseYear: 1998,
  genre: 'Trip-hop',
  reviewCount: 0,
  version: 1,
  createdAt: '2026-10-02T10:00:00Z',
  updatedAt: '2026-10-02T10:00:00Z',
}

describe('AlbumCard', () => {
  it('renders unrated albums without inventing a score', () => {
    const wrapper = mount(AlbumCard, {
      props: { album },
      global: {
        stubs: {
          RouterLink: { template: '<a><slot /></a>' },
        },
      },
    })

    expect(wrapper.text()).toContain('Mezzanine')
    expect(wrapper.text()).toContain('Massive Attack')
    expect(wrapper.text()).toContain('—/10')
    expect(wrapper.text()).toContain('0 reviews')
  })
})
