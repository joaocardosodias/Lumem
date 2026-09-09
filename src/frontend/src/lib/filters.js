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
