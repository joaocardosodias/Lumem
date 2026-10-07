import { mapStation, resolveStation } from '../data/stationCatalog.js'
import { inDurationRange, inPeriod } from './filters.js'

const contains = (values, search) => {
  const query = search.trim().toLowerCase()
  return !query || values.some((value) => String(value ?? '').toLowerCase().includes(query))
}
const stationMatches = (value, station) => !station || mapStation(value) === mapStation(station)

export function filterMaintenance(items, filters, search = '', now) {
  return items.filter((item) =>
    stationMatches(item.station_code, filters.station) &&
    (!filters.orderType || item.order_type === filters.orderType) &&
    (!filters.source || item.source === filters.source) &&
    inPeriod(item.start_timestamp, filters.period, now) &&
    inDurationRange(item.duration_hours, filters.durationMin ?? '', filters.durationMax ?? '') &&
    contains([item.sap_order_number, item.installation_location, item.symptom, item.description, item.station_code], search),
  )
}

export function filterAlarms(items, filters, now) {
  return items.filter((item) =>
    stationMatches(item.station_code, filters.station) &&
    (!filters.platform || item.platform === filters.platform) &&
    (!filters.alarmType || item.alarm_type === filters.alarmType) &&
    (filters.normalization !== 'open' || !item.normalization_timestamp) &&
    (filters.normalization !== 'normalized' || Boolean(item.normalization_timestamp)) &&
    inPeriod(item.event_timestamp, filters.period, now) &&
    contains([item.tag, item.alarm_type, item.station_name, item.station_code, item.door_number], filters.search || ''),
  )
}

export function isAiInspection(item) {
  if (item.prediction_id) return true
  const notes = String(item.notes || '').toLowerCase()
  return notes.includes('automaticamente pela ia') || notes.includes('rodada #')
}

export function inspectionStation(item) {
  return resolveStation(item) || String(item.station_code || '').toUpperCase()
}

export function filterInspections(items, filters, search = '', now) {
  return items.filter((item) =>
    stationMatches(inspectionStation(item), filters.station) &&
    (!filters.platform || item.platform === filters.platform) &&
    (filters.origin !== 'ai' || isAiInspection(item)) &&
    (filters.origin !== 'manual' || !isAiInspection(item)) &&
    inPeriod(item.created_at, filters.period, now) &&
    contains([item.door_number, item.notes, item.station_code, item.platform], search),
  )
}

export function filterInspectionStatus(items, status) {
  return status ? items.filter((item) => item.status === status) : items
}

export function filterPredictions(items, filters, status = 'all') {
  return items.filter((item) =>
    (status !== 'fail' || item.will_fail) &&
    (status !== 'ok' || !item.will_fail) &&
    stationMatches(item.station_code, filters.station) &&
    (!filters.platform || item.platform === filters.platform) &&
    contains([item.sap_tag, item.station_name, item.station_code, item.door_number], filters.search || ''),
  )
}
