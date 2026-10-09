import type { Problem } from '../api/generated'

export function isProblem(value: unknown): value is Problem {
  if (!value || typeof value !== 'object') return false
  const candidate = value as Partial<Problem>
  return (
    typeof candidate.status === 'number' && typeof candidate.code === 'string' && typeof candidate.title === 'string'
  )
}

export function problemMessage(error: unknown): string {
  if (isProblem(error)) {
    if (error.violations?.length) {
      return error.violations.map((violation) => `${violation.field}: ${violation.description}`).join(' · ')
    }
    return error.detail ?? error.title
  }
  if (error instanceof Error && error.message) return error.message
  return 'Something went wrong. Please try again.'
}

export function problemFieldErrors(error: unknown): Record<string, string> {
  if (!isProblem(error) || !error.violations) return {}
  return Object.fromEntries(error.violations.map((violation) => [violation.field, violation.description]))
}
