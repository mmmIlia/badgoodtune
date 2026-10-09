import { describe, expect, it } from 'vitest'

import type { Problem } from '../api/generated'
import { isProblem, problemMessage } from './problem'

describe('problem helpers', () => {
  it('formats field violations without exposing transport internals', () => {
    const problem: Problem = {
      type: 'about:blank',
      title: 'Invalid request',
      status: 400,
      code: 'INVALID_ARGUMENT',
      requestId: 'request-1',
      violations: [
        { field: 'artist', description: 'must not be empty' },
        { field: 'releaseYear', description: 'is invalid' },
      ],
    }

    expect(isProblem(problem)).toBe(true)
    expect(problemMessage(problem)).toBe('artist: must not be empty · releaseYear: is invalid')
  })

  it('uses a safe fallback for unknown thrown values', () => {
    expect(isProblem(null)).toBe(false)
    expect(problemMessage(Symbol('failure'))).toBe('Something went wrong. Please try again.')
  })
})
