export function formatDate(value: string): string {
  return new Intl.DateTimeFormat('en', { day: 'numeric', month: 'short', year: 'numeric' }).format(new Date(value))
}

export function formatScore(value?: number): string {
  return value === undefined ? '—' : value.toFixed(1)
}
