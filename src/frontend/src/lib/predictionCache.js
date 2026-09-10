const STORAGE_KEY = 'motiva:latest_prediction'

let memoryRunData = null
let memoryItems = null

export function getCachedPrediction() {
  if (memoryRunData && memoryItems && memoryItems.length) {
    return { runData: memoryRunData, items: memoryItems }
  }

  if (typeof window === 'undefined') return null

  try {
    const raw = localStorage.getItem(STORAGE_KEY) || sessionStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw)
    if (parsed && parsed.runData && Array.isArray(parsed.items) && parsed.items.length) {
      memoryRunData = parsed.runData
      memoryItems = parsed.items
      return { runData: parsed.runData, items: parsed.items }
    }
  } catch {
    return null
  }

  return null
}

export function setCachedPrediction(runData, items) {
  memoryRunData = runData
  memoryItems = items

  if (typeof window === 'undefined') return

  try {
    const payload = JSON.stringify({ runData, items })
    localStorage.setItem(STORAGE_KEY, payload)
    sessionStorage.setItem(STORAGE_KEY, payload)
  } catch {
    try {
      sessionStorage.setItem(STORAGE_KEY, JSON.stringify({ runData, items }))
    } catch {}
  }
}

export function invalidatePredictionCache() {
  memoryRunData = null
  memoryItems = null

  if (typeof window === 'undefined') return

  try {
    localStorage.removeItem(STORAGE_KEY)
    sessionStorage.removeItem(STORAGE_KEY)
  } catch {}
}
