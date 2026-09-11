import { stationLabel } from '../data/stations'

const STORAGE_KEY = 'motiva:latest_prediction'

let memoryRunData = null
let memoryItems = null

function formatTag(stationCode, platform, doorNumber) {
  const pad = String(doorNumber || '').padStart(2, '0')
  return `SP04-6-PSD-${stationCode}-${platform || 'P1'}-PP${pad}`
}

export function predictionRun(response, fallbackHorizonDays = 7) {
  return {
    id: response?.id || '',
    horizon_days: response?.horizon_days || fallbackHorizonDays,
    total_doors_evaluated: response?.total_doors_evaluated || 0,
    total_doors_at_risk: response?.total_doors_at_risk || 0,
    created_at: response?.created_at || '',
  }
}

export function normalizePrediction(item) {
  return {
    door_id: item.door_id ?? item.id,
    run_id: item.run_id || '',
    station_code: item.station_code || '',
    station_name: stationLabel(item.station_code) || item.station_name || item.station_code || '',
    platform: item.platform || 'P1',
    door_number: item.door_number,
    sap_tag: item.sap_tag || formatTag(item.station_code, item.platform, item.door_number),
    will_fail: Boolean(item.will_fail),
  }
}

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
