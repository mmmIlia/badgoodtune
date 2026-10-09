import { describe, expect, it } from 'vitest'

import { returnPath } from './navigation'

describe('returnPath', () => {
  it('accepts local paths', () => {
    expect(returnPath('/albums/example')).toBe('/albums/example')
  })

  it('rejects external and protocol-relative locations', () => {
    expect(returnPath('https://example.com')).toBe('/')
    expect(returnPath('//example.com')).toBe('/')
  })
})
