import { api, asList } from './api'
import { stationLabel } from '../data/stations'

const STORAGE_KEY = 'motiva:alarms_cache'

let memoryAlarms = null
let inflight = null

export function mapAlarm(item) {
  return {
    id: item.id,
    tag: item.tag || `SP04-6-PSD-${item.station_code || 'BUT'}-${item.platform || 'P1'}-PP${String(item.door_number || '01').padStart(2, '0')}`,
    event_timestamp: item.event_timestamp || '—',
    station_code: item.station_code || '',
    station_name: stationLabel(item.station_code) || item.station_code || '—',
    platform: item.platform || 'P1',
    door_number: item.door_number,
    alarm_type: item.alarm_type || 'Alarme de Telemetria',
    duration_seconds: item.duration_seconds || 0,
    normalization_timestamp: item.normalization_timestamp || '',
    source: item.source || 'telemetria',
  }
}

export function getCachedAlarms() {
  if (memoryAlarms && memoryAlarms.length) {
    return memoryAlarms
  }

  if (typeof window === 'undefined') return null

  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed) && parsed.length) {
      memoryAlarms = parsed
      return parsed
    }
  } catch {
    return null
  }

  return null
}

export function setCachedAlarms(items) {
  memoryAlarms = items

  if (typeof window === 'undefined' || !items) return

  try {
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(items))
  } catch {}
}

export function invalidateAlarms() {
  memoryAlarms = null
  inflight = null

  if (typeof window === 'undefined') return

  try {
    sessionStorage.removeItem(STORAGE_KEY)
  } catch {}
}

export async function prefetchAlarms() {
  if (getCachedAlarms()) return getCachedAlarms()
  if (inflight) return inflight

  inflight = api
    .alarms({ limit: 500 })
    .then((res) => {
      const list = asList(res?.data || res?.alarms || res)
      const mapped = list.map(mapAlarm)
      setCachedAlarms(mapped)
      inflight = null
      return mapped
    })
    .catch(() => {
      inflight = null
      return null
    })

  return inflight
}
