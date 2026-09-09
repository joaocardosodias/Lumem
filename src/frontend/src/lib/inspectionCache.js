import { api } from './api'

let cache = null
let inflight = null

export function getCachedInspections() {
  return cache
}

export function invalidateInspections() {
  cache = null
  inflight = null
}

export function patchCachedInspection(id, updates) {
  if (!cache) return
  cache = cache.map((item) => (item.id === id ? { ...item, ...updates } : item))
}

export function loadInspections({ force = false, onProgress } = {}) {
  if (force) {
    cache = null
    inflight = null
  }
  if (cache) return Promise.resolve(cache)
  if (inflight) return inflight

  inflight = api
    .listAllInspections({
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
