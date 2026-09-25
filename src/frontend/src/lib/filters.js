export const PAGE_SIZE = 50

export const PERIODS = [
  { id: '1d', label: '1 dia', days: 1 },
  { id: '7d', label: '7 dias', days: 7 },
  { id: '30d', label: '30 dias', days: 30 },
  { id: '6m', label: '6 meses', months: 6 },
  { id: '1y', label: '1 ano', years: 1 },
  { id: 'all', label: 'Tudo', days: null },
]

export const DEFAULT_PERIOD = 'all'

export function dateKey(value) {
  if (!value) return ''
  const text = String(value).trim()
  const iso = text.slice(0, 10)
  if (/^\d{4}-\d{2}-\d{2}$/.test(iso)) return iso

  const parsed = Date.parse(text.includes('T') ? text : text.replace(' ', 'T'))
  if (!Number.isFinite(parsed)) return ''
  const date = new Date(parsed)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function localDateKey(date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export function periodCutoff(periodId) {
  const period = PERIODS.find((item) => item.id === periodId)
  if (!period || period.id === 'all') return null

  const start = new Date()
  start.setHours(0, 0, 0, 0)
  if (period.years) start.setFullYear(start.getFullYear() - period.years)
  else if (period.months) start.setMonth(start.getMonth() - period.months)
  else start.setDate(start.getDate() - (period.days - 1))
  return localDateKey(start)
}

export function inPeriod(value, periodId) {
  const cutoff = periodCutoff(periodId)
  if (!cutoff) return true
  const key = dateKey(value)
  if (!key) return false
  return key >= cutoff
}

export function periodHint(periodId) {
  switch (periodId) {
    case '1d':
      return 'último dia'
    case '7d':
      return 'últimos 7 dias'
    case '30d':
      return 'últimos 30 dias'
    case '6m':
      return 'últimos 6 meses'
    case '1y':
      return 'último ano'
    default:
      return ''
  }
}

export function inDateRange(value, from, to) {
  if (!from && !to) return true
  const key = dateKey(value)
  if (!key) return false
  if (from && key < from) return false
  if (to && key > to) return false
  return true
}

export function inDurationRange(hours, min, max) {
  if (min === '' && max === '') return true
  const value = Number(hours)
  if (!Number.isFinite(value)) return false
  if (min !== '' && value < Number(min)) return false
  if (max !== '' && value > Number(max)) return false
  return true
}
