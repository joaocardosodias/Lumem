import { api } from './api'

let cache = null
let inflight = null

export function getCachedMaintenance() {
  return cache
}

export function invalidateMaintenance() {
  cache = null
  inflight = null
}

export function loadMaintenance({ force = false, onProgress } = {}) {
  if (force) {
    cache = null
    inflight = null
  }
  if (cache) return Promise.resolve(cache)
  if (inflight) return inflight

  inflight = api
    .listAllMaintenance({
      pageSize: 500,
      concurrency: 8,
      onProgress,
    })
    .then((items) => {
      cache = items
      inflight = null
      return items
    })
    .catch((error) => {
      inflight = null
      throw error
    })

  return inflight
}
