export const lineStations = [
  { code: 'VSO', name: 'Vila Sônia', platforms: ['P1', 'P2'] },
  { code: 'MBI', name: 'São Paulo-Morumbi', platforms: ['P1', 'P2'] },
  { code: 'BUT', name: 'Butantã', platforms: ['P1', 'P2'] },
  { code: 'PIN', name: 'Pinheiros', platforms: ['P1', 'P2'] },
  { code: 'FAL', name: 'Faria Lima', platforms: ['P1', 'P2'] },
  { code: 'FRA', name: 'Fradique Coutinho', platforms: ['P1', 'P2'] },
  { code: 'OFR', name: 'Oscar Freire', platforms: ['P1', 'P2'] },
  { code: 'PAU', name: 'Paulista', platforms: ['P1', 'P2'] },
  { code: 'HIG', name: 'Higienópolis', platforms: ['P1', 'P2'] },
  { code: 'REP', name: 'República', platforms: ['P1', 'P2'] },
  { code: 'LUZ', name: 'Luz', platforms: ['P1', 'P2'] },
]

const knownStations = new Set(lineStations.map((item) => item.code))

const stationAliases = {
  PTA: 'PAU',
  OSC: 'OFR',
  FRE: 'OFR',
}

const genericCenters = new Set(['PVS', 'ENC', 'GER'])

export function mapStation(code) {
  const mapped = stationAliases[code] || code
  return genericCenters.has(mapped) ? '' : mapped
}

function extractStationCodes(value) {
  const text = String(value || '').toUpperCase().trim()
  if (!text) return []

  const codes = []
  const push = (code) => {
    const mapped = mapStation(code)
    if (mapped) codes.push(mapped)
  }

  if (/^[A-Z]{3}$/.test(text)) push(text)
  for (const match of text.matchAll(/PSD[-_]?([A-Z]{3})/g)) push(match[1])
  for (const match of text.matchAll(/SP04I([A-Z]{3})/g)) push(match[1])
  return codes
}

export function resolveStation(item) {
  if (!item || typeof item !== 'object') return ''
  const fromLocation = extractStationCodes(item.installation_location)
  const fromCode = extractStationCodes(item.station_code)
  const candidates = [...fromLocation, ...fromCode]
  return candidates.find((code) => knownStations.has(code)) || candidates[0] || ''
}

export function stationLabel(code) {
  const resolved = mapStation(String(code || '').toUpperCase()) || code
  return lineStations.find((item) => item.code === resolved)?.name ?? resolved
}

export function normalizeMaintenance(item) {
  if (!item || typeof item !== 'object') return item
  return { ...item, station_code: resolveStation(item) || item.station_code || '' }
}
