const stationDoorsCache = new Map()
const doorDossierCache = new Map()
let lastOpenedStationCode = null

export function getCachedStationDoors(code) {
  if (!code) return null
  return stationDoorsCache.get(code) || null
}

export function setCachedStationDoors(code, doors) {
  if (!code || !doors) return
  stationDoorsCache.set(code, doors)
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
}
