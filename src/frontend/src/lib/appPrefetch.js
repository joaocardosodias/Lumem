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
  setCachedModelConfig,
  setCachedPrediction,
} from './predictionCache'

let prefetchingEssential = null
let prefetchingBackground = null

function report(onProgress, loaded, total) {
  onProgress?.({ loaded, total, pages: loaded })
}

export async function prefetchDashboard({ onProgress } = {}) {
  let loaded = 0
  const total = 4
  report(onProgress, 0, total)

  const [dashboardResult, predictionsResult, configResult, activeModelResult] = await Promise.allSettled([
    api.dashboard().then((res) => {
      loaded += 1
      report(onProgress, loaded, total)
      return res
    }),
    api.predictionsLatest().then((res) => {
      loaded += 1
      report(onProgress, loaded, total)
      return res
    }),
    api.getMLConfig().then((res) => {
      loaded += 1
      report(onProgress, loaded, total)
      return res
    }).catch(() => null),
    api.getActiveModel().then((res) => {
      loaded += 1
      report(onProgress, loaded, total)
      return res
    }).catch(() => null),
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

  const cfg = configResult.status === 'fulfilled' ? configResult.value : null
  const active = activeModelResult.status === 'fulfilled' ? activeModelResult.value : null
  if (cfg || active) {
    setCachedModelConfig({
      threshold: cfg?.threshold ?? 0.28,
      defaultHorizon: cfg?.default_horizon_days ?? 7,
      modelName: 'Random Forest',
      version: active?.version || 'v1.0.0',
      artifact: active?.artifact_path ? 'modelo_5.joblib' : 'modelo_5.joblib',
      recall: active?.test_metrics?.recall ? `${(active.test_metrics.recall * 100).toFixed(1)}%` : '55,3%',
      precision: active?.test_metrics?.precision ? `${(active.test_metrics.precision * 100).toFixed(1)}%` : '15,9%',
    })
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
      }
    }
  }

  await Promise.all(Array.from({ length: workers }, work))
}

export async function prefetchEssentialData({ onProgress } = {}) {
  if (prefetchingEssential) return prefetchingEssential

  prefetchingEssential = prefetchDashboard({ onProgress }).finally(() => {
    prefetchingEssential = null
  })

  return prefetchingEssential
}

export function prefetchBackgroundData() {
  if (prefetchingBackground) return prefetchingBackground

  const tasks = [
    prefetchAlarms,
    loadMaintenance,
    loadInspections,
    prefetchStationDoors,
  ]

  prefetchingBackground = Promise.all(
    tasks.map(async (task) => {
      try {
        await task()
      } catch {
      }
    }),
  ).finally(() => {
    prefetchingBackground = null
  })

  return prefetchingBackground
}

export async function prefetchApplicationData({ onProgress } = {}) {
  await prefetchEssentialData({ onProgress })
  prefetchBackgroundData()
}

export function invalidateApplicationCache() {
  invalidateAlarms()
  invalidateDashboard()
  invalidateDoorsCache()
  invalidateInspections()
  invalidateMaintenance()
  invalidatePredictionCache()
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new Event('motiva:cache-invalidated'))
  }
}
