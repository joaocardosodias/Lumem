import test from 'node:test'
import assert from 'node:assert/strict'
import { dashboardHighlights } from '../src/lib/dashboardHighlights.js'

test('RF10 ranks critical stations by descending risk and selects the top', () => {
  const summary = { critical_stations: [
    { station_code: 'PIN', doors_at_risk: 1 },
    { station_code: 'BUT', doors_at_risk: 2 },
    { station_code: 'FAL', doors_at_risk: 3 },
  ] }
  const before = structuredClone(summary)
  const result = dashboardHighlights(summary)
  assert.deepEqual(result.stations.map((s) => s.station_code), ['FAL', 'BUT', 'PIN'])
  assert.equal(result.topStation.station_code, 'FAL')
  assert.equal(result.maxRisk, 3)
  assert.deepEqual(summary, before)
})

test('RF10 empty dashboard has no critical station or prioritized doors', () => {
  assert.deepEqual(dashboardHighlights(null), {
    stations: [], maxRisk: 1, topStation: undefined, prioritized: [], visiblePredictions: [],
  })
})

test('RF10 tied station counts preserve the received order', () => {
  const stations = [{ station_code: 'PIN', doors_at_risk: 2 }, { station_code: 'FAL', doors_at_risk: 2 }]
  assert.deepEqual(dashboardHighlights({ critical_stations: stations }).stations, stations)
})

test('RF11 attention panel selects only risky predictions and preserves identity', () => {
  const items = [
    { id: 'stable', station_code: 'FAL', platform: 'P1', door_number: 1, will_fail: false },
    { id: 'risk', station_code: 'FAL', platform: 'P2', door_number: 1, will_fail: true },
  ]
  const before = structuredClone(items)
  const result = dashboardHighlights({}, items)
  assert.deepEqual(result.prioritized, [items[1]])
  assert.deepEqual(result.visiblePredictions, [items[1]])
  assert.deepEqual(items, before)
})

test('RF11 attention panel shows five doors while preserving total risk count', () => {
  const items = Array.from({ length: 7 }, (_, i) => ({ id: i, will_fail: true }))
  const result = dashboardHighlights({}, items)
  assert.equal(result.prioritized.length, 7)
  assert.deepEqual(result.visiblePredictions.map((p) => p.id), [0, 1, 2, 3, 4])
})

test('RF11 all stable predictions leave the attention panel empty', () => {
  const result = dashboardHighlights({}, [{ id: 1, will_fail: false }])
  assert.deepEqual(result.prioritized, [])
  assert.deepEqual(result.visiblePredictions, [])
})
