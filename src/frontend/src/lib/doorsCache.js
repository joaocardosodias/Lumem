import { api, normalizeDoors } from './api'

const stationDoorsCache = new Map()
const doorDossierCache = new Map()
const stationDoorsInflight = new Map()
let lastOpenedStationCode = null

export function getCachedStationDoors(code) {
  if (!code) return null
  return stationDoorsCache.get(code) || null
}

export function setCachedStationDoors(code, doors) {
  if (!code || !doors) return
  stationDoorsCache.set(code, doors)
}

export function loadStationDoors(code, stationName, { force = false } = {}) {
  if (!code) return Promise.resolve([])

  const cached = getCachedStationDoors(code)
  if (cached && !force) return Promise.resolve(cached)

  const pending = stationDoorsInflight.get(code)
  if (pending) return pending

  const request = api
    .stationDoors(code)
    .then((payload) => {
      const doors = normalizeDoors(payload, code, stationName)
      setCachedStationDoors(code, doors)
      return doors
    })
    .finally(() => {
      stationDoorsInflight.delete(code)
    })

  stationDoorsInflight.set(code, request)
  return request
}

export function getCachedDoorDossier(doorId) {
  if (!doorId) return null
  return doorDossierCache.get(doorId) || null
}

export function setCachedDoorDossier(doorId, dossier) {
  if (!doorId || !dossier) return
  doorDossierCache.set(doorId, dossier)
}

export function getLastOpenedStation() {
  return lastOpenedStationCode
}

export function setLastOpenedStation(code) {
  lastOpenedStationCode = code
}

export function invalidateDoorsCache() {
  stationDoorsCache.clear()
  doorDossierCache.clear()
  stationDoorsInflight.clear()
  lastOpenedStationCode = null
}
