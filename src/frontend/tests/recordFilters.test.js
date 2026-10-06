import test from 'node:test'
import assert from 'node:assert/strict'
import { filterMaintenance, filterAlarms, filterInspections, filterInspectionStatus, filterPredictions } from '../src/lib/recordFilters.js'
import { inPeriod, inDateRange, inDurationRange } from '../src/lib/filters.js'

const now = new Date(2026, 9, 5, 12)
const ids = (items) => items.map((item) => item.id)
const maintenance = [
  { id: 'a', station_code: 'FAL', order_type: 'ZM01', source: 'manual_form', start_timestamp: '2026-10-05', duration_hours: 1.5, description: 'Troca do SENSOR' },
  { id: 'b', station_code: 'PIN', order_type: 'ZM03', source: 'csv', start_timestamp: '2026-09-29', duration_hours: 3, description: 'Limpeza' },
  { id: 'c', station_code: 'FAL', order_type: 'ZM03', source: 'csv', start_timestamp: '2026-09-28', duration_hours: 0.5 },
]
const alarms = [
  { id: 'a', station_code: 'FAL', platform: 'P1', alarm_type: 'OBS', event_timestamp: '2026-10-05', normalization_timestamp: null, tag: 'PORTA-4' },
  { id: 'b', station_code: 'FAL', platform: 'P2', alarm_type: 'CA', event_timestamp: '2026-09-29', normalization_timestamp: '2026-09-29', tag: 'PORTA-8' },
  { id: 'c', station_code: 'PIN', platform: 'P1', alarm_type: 'OBS', event_timestamp: '2026-09-28', normalization_timestamp: null },
]
const inspections = [
  { id: 'a', station_code: 'FAL', platform: 'P1', status: 'pending', prediction_id: 'prediction-1', created_at: '2026-10-05', door_number: 4 },
  { id: 'b', station_code: 'FAL', platform: 'P2', status: 'completed', notes: 'Verificar sensor', created_at: '2026-09-29', door_number: 8 },
  { id: 'c', station_code: 'PIN', platform: 'P1', status: 'in_progress', created_at: '2026-09-28', door_number: 2 },
]
const predictions = [
  { id: 'a', station_code: 'FAL', platform: 'P1', will_fail: true, sap_tag: 'PORTA-4' },
  { id: 'b', station_code: 'FAL', platform: 'P2', will_fail: false, sap_tag: 'PORTA-8' },
  { id: 'c', station_code: 'PIN', platform: 'P1', will_fail: true, sap_tag: 'PORTA-2' },
]

const cases = [
  ['maintenance station', () => filterMaintenance(maintenance, { station: 'FAL' }), ['a', 'c']],
  ['maintenance period boundary', () => filterMaintenance(maintenance, { period: '7d' }, '', now), ['a', 'b']],
  ['maintenance type', () => filterMaintenance(maintenance, { orderType: 'ZM03' }), ['b', 'c']],
  ['maintenance source', () => filterMaintenance(maintenance, { source: 'manual_form' }), ['a']],
  ['maintenance duration inclusive', () => filterMaintenance(maintenance, { durationMin: '1.5', durationMax: '3' }), ['a', 'b']],
  ['maintenance combined', () => filterMaintenance(maintenance, { station: 'FAL', period: '7d', orderType: 'ZM01', source: 'manual_form' }, ' sensor ', now), ['a']],
  ['maintenance no results', () => filterMaintenance(maintenance, { station: 'PIN', orderType: 'ZM01' }), []],
  ['maintenance reset', () => filterMaintenance(maintenance, { period: 'all' }), ['a', 'b', 'c']],
  ['alarm station', () => filterAlarms(alarms, { station: 'FAL' }), ['a', 'b']],
  ['alarm period', () => filterAlarms(alarms, { period: '7d' }, now), ['a', 'b']],
  ['alarm open', () => filterAlarms(alarms, { normalization: 'open' }), ['a', 'c']],
  ['alarm normalized', () => filterAlarms(alarms, { normalization: 'normalized' }), ['b']],
  ['alarm type', () => filterAlarms(alarms, { alarmType: 'OBS' }), ['a', 'c']],
  ['alarm combined', () => filterAlarms(alarms, { station: 'FAL', platform: 'P2', period: '7d', normalization: 'normalized', search: 'porta-8' }, now), ['b']],
  ['alarm no results', () => filterAlarms(alarms, { station: 'PIN', platform: 'P2' }), []],
  ['alarm reset', () => filterAlarms(alarms, {}), ['a', 'b', 'c']],
  ['inspection station', () => filterInspections(inspections, { station: 'FAL' }), ['a', 'b']],
  ['inspection period', () => filterInspections(inspections, { period: '7d' }, '', now), ['a', 'b']],
  ['inspection AI', () => filterInspections(inspections, { origin: 'ai' }), ['a']],
  ['inspection manual', () => filterInspections(inspections, { origin: 'manual' }), ['b', 'c']],
  ['inspection status', () => filterInspectionStatus(inspections, 'completed'), ['b']],
  ['inspection combined', () => filterInspectionStatus(filterInspections(inspections, { station: 'FAL', platform: 'P1', origin: 'ai', period: '7d' }, '4', now), 'pending'), ['a']],
  ['inspection no results', () => filterInspectionStatus(filterInspections(inspections, { station: 'PIN' }), 'completed'), []],
  ['inspection reset', () => filterInspectionStatus(filterInspections(inspections, {}), ''), ['a', 'b', 'c']],
  ['predictive alerts risk', () => filterPredictions(predictions, {}, 'fail'), ['a', 'c']],
  ['predictive alerts stable', () => filterPredictions(predictions, {}, 'ok'), ['b']],
  ['predictive alerts combined', () => filterPredictions(predictions, { station: 'FAL', platform: 'P1', search: 'porta-4' }, 'fail'), ['a']],
  ['predictive alerts no results', () => filterPredictions(predictions, { station: 'PIN' }, 'ok'), []],
  ['predictive alerts reset', () => filterPredictions(predictions, {}), ['a', 'b', 'c']],
]
for (const [name, select, expected] of cases) test(`RF06 ${name}`, () => assert.deepEqual(ids(select()), expected))

test('RF06 station aliases apply to all record filters', () => {
  for (const select of [filterMaintenance, filterAlarms, filterInspections, filterPredictions]) {
    assert.deepEqual(ids(select([{ id: 'alias', station_code: 'PTA' }], { station: 'PAU' })), ['alias'])
  }
})
test('RF06 filtering does not mutate the source collection', () => {
  const before = structuredClone(maintenance)
  filterMaintenance(maintenance, { station: 'FAL', period: '7d' }, '', now)
  assert.deepEqual(maintenance, before)
})
test('RF06 custom date range includes both endpoints', () => {
  for (const value of ['2026-09-29', '2026-10-05']) assert.equal(inDateRange(value, '2026-09-29', '2026-10-05'), true)
  for (const value of ['2026-09-28', '2026-10-06', 'invalid']) assert.equal(inDateRange(value, '2026-09-29', '2026-10-05'), false)
})
test('RF06 recent period excludes future and missing dates', () => {
  for (const value of ['2026-10-06', '', null, 'invalid']) assert.equal(inPeriod(value, '7d', now), false)
})
test('RF06 duration filter excludes missing values when a bound is set', () => {
  for (const value of [null, undefined, '', 'invalid']) assert.equal(inDurationRange(value, '0', '2'), false)
})
