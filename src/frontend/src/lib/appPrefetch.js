import { lineStations } from '../data/stations'
import { api, asList, mergeDashboard } from './api'
import { invalidateAlarms, prefetchAlarms } from './alarmCache'
import { getCachedDashboard, invalidateDashboard, setCachedDashboard } from './dashboardCache'
import { invalidateDoorsCache, loadStationDoors } from './doorsCache'
import { invalidateInspections, loadInspections } from './inspectionCache'
import { invalidateMaintenance, loadMaintenance } from './maintenanceCache'
import {
  invalidatePredictionCache,
  normalizePrediction,
  predictionRun,
  setCachedPrediction,
} from './predictionCache'

let prefetching = null

function report(onProgress, loaded, total) {
  onProgress?.({ loaded, total, pages: loaded })
}

async function prefetchDashboard() {
  const [dashboardResult, predictionsResult] = await Promise.allSettled([
    api.dashboard(),
    api.predictionsLatest(),
  ])

  const predictions =
    predictionsResult.status === 'fulfilled'
      ? asList(predictionsResult.value.predictions || predictionsResult.value)
      : getCachedDashboard()?.predictions || []

  if (dashboardResult.status === 'fulfilled') {
    setCachedDashboard(mergeDashboard(dashboardResult.value, null), predictions)
  }

  if (predictionsResult.status === 'fulfilled') {
    const response = predictionsResult.value
    setCachedPrediction(
      predictionRun(response),
      asList(response.predictions || response).map(normalizePrediction),
    )
  }
}

async function prefetchStationDoors() {
  const queue = [...lineStations]
  const workers = Math.min(3, queue.length)

  async function work() {
    while (queue.length) {
      const station = queue.shift()
      try {
        await loadStationDoors(station.code, station.name)
      } catch {
        // Uma estação indisponível não deve impedir o acesso às demais telas.
      }
    }
  }

  await Promise.all(Array.from({ length: workers }, work))
}

export function prefetchApplicationData({ onProgress } = {}) {
  if (prefetching) return prefetching

  const tasks = [
    prefetchDashboard,
    prefetchAlarms,
    loadMaintenance,
    loadInspections,
    prefetchStationDoors,
  ]
  const total = tasks.length
  let loaded = 0
  report(onProgress, loaded, total)

  prefetching = Promise.all(
    tasks.map(async (task) => {
      try {
        await task()
      } catch {
        // A tela correspondente tentará novamente quando for aberta.
      } finally {
        loaded += 1
        report(onProgress, loaded, total)
      }
    }),
  ).finally(() => {
    prefetching = null
  })

  return prefetching
}

export function invalidateApplicationCache() {
  invalidateAlarms()
  invalidateDashboard()
  invalidateDoorsCache()
  invalidateInspections()
  invalidateMaintenance()
  invalidatePredictionCache()
}
