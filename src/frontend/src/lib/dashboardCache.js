const STORAGE_KEY = 'motiva:dashboard_cache'

let memorySummary = null
let memoryPredictions = null

export function getCachedDashboard() {
  if (memorySummary || (memoryPredictions && memoryPredictions.length)) {
    return { summary: memorySummary, predictions: memoryPredictions || [] }
  }

  if (typeof window === 'undefined') return null

  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw)
    if (parsed && (parsed.summary || parsed.predictions)) {
      memorySummary = parsed.summary || null
      memoryPredictions = parsed.predictions || []
      return { summary: memorySummary, predictions: memoryPredictions }
    }
  } catch {
    return null
  }

  return null
}

export function setCachedDashboard(summary, predictions) {
  if (summary) memorySummary = summary
  if (predictions) memoryPredictions = predictions

  if (typeof window === 'undefined') return

  try {
    sessionStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ summary: memorySummary, predictions: memoryPredictions }),
    )
  } catch {}
}

export function invalidateDashboard() {
  memorySummary = null
  memoryPredictions = null

  if (typeof window === 'undefined') return

  try {
    sessionStorage.removeItem(STORAGE_KEY)
  } catch {}
}
