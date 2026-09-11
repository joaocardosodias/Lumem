import { normalizeMaintenance } from '../data/stations'

const TOKEN_KEY = 'motiva.token'
const API_ORIGIN = String(import.meta.env.VITE_API_URL || '').replace(/\/$/, '')

let onUnauthorized = null

export function setUnauthorizedHandler(handler) {
  onUnauthorized = handler
}

export function getToken() {
  if (typeof window === 'undefined') return null
  return localStorage.getItem(TOKEN_KEY) || sessionStorage.getItem(TOKEN_KEY)
}

export function setToken(token) {
  if (typeof window === 'undefined') return
  if (token) {
    localStorage.setItem(TOKEN_KEY, token)
    sessionStorage.setItem(TOKEN_KEY, token)
  } else {
    localStorage.removeItem(TOKEN_KEY)
    sessionStorage.removeItem(TOKEN_KEY)
  }
}

export class ApiError extends Error {
  constructor(status, body) {
    const detail =
      (body && typeof body === 'object' && (body.detail || body.title)) ||
      (typeof body === 'string' && body) ||
      `Erro ${status}`
    super(detail)
    this.name = 'ApiError'
    this.status = status
    this.body = body
    this.code = body?.code
  }
}

export function errorMessage(error) {
  if (error instanceof ApiError) return error.message
  return error?.message || 'Não foi possível concluir a operação.'
}

function queryString(params = {}) {
  const search = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      search.set(key, String(value))
    }
  })
  const text = search.toString()
  return text ? `?${text}` : ''
}

async function parseBody(response) {
  const text = await response.text()
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    return text
  }
}

export async function request(path, { method = 'GET', body, formData, auth = true } = {}) {
  const headers = {}
  if (auth) {
    const token = getToken()
    if (token) headers.Authorization = `Bearer ${token}`
  }
  if (body !== undefined && !formData) {
    headers['Content-Type'] = 'application/json'
  }

  const url = import.meta.env.DEV || !API_ORIGIN ? path : `${API_ORIGIN}${path}`
  const response = await fetch(url, {
    method,
    headers,
    body: formData ?? (body !== undefined ? JSON.stringify(body) : undefined),
  })

  const data = await parseBody(response)

  if (!response.ok) {
    if (response.status === 401 && auth) onUnauthorized?.()
    throw new ApiError(
      response.status,
      typeof data === 'object' && data ? data : { detail: String(data || `Erro ${response.status}`) },
    )
  }

  return { data, headers: response.headers, status: response.status }
}

export function asList(payload) {
  if (Array.isArray(payload)) return payload
  if (!payload || typeof payload !== 'object') return []

  for (const key of [
    'items',
    'data',
    'results',
    'doors',
    'predictions',
    'maintenance',
    'maintenances',
    'inspections',
    'alarms',
  ]) {
    if (Array.isArray(payload[key])) return payload[key]
  }

  if (payload.data && typeof payload.data === 'object') {
    const nested = asList(payload.data)
    if (nested.length) return nested
  }

  const values = Object.values(payload)
  if (values.length && values.every((value) => Array.isArray(value))) {
    return values.flat()
  }

  return []
}

export function listTotal(payload, loaded = 0) {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) return null
  for (const key of ['total', 'total_count', 'totalCount']) {
    const value = Number(payload[key])
    if (Number.isFinite(value) && value > loaded) return value
  }
  return null
}

function pickNumber(source, ...keys) {
  if (!source || typeof source !== 'object') return 0
  for (const key of keys) {
    const value = source[key]
    if (typeof value === 'number' && Number.isFinite(value)) return value
    if (typeof value === 'string' && value.trim() !== '' && Number.isFinite(Number(value))) {
      return Number(value)
    }
  }
  return 0
}

export function normalizeInspectionStatus(status) {
  const value = String(status ?? '')
    .trim()
    .toLowerCase()
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/[\s-]+/g, '_')

  if (!value) return 'pending'
  if (['pending', 'pendente', 'open', 'todo', 'created', 'new', '0'].includes(value)) return 'pending'
  if (['in_progress', 'inprogress', 'em_andamento', 'andamento'].includes(value)) return 'in_progress'
  if (['completed', 'complete', 'concluida', 'done', 'closed', 'finalizada', '1'].includes(value)) {
    return 'completed'
  }
  return 'pending'
}

function normalizeInspectionItem(item) {
  const row = item && typeof item === 'object' ? item : { id: item }
  return {
    ...row,
    status: normalizeInspectionStatus(
      row.status ?? row.state ?? row.inspection_status ?? row.current_status,
    ),
  }
}

export function listInspections(payload) {
  if (Array.isArray(payload)) return payload.map(normalizeInspectionItem)
  if (!payload || typeof payload !== 'object') return []

  for (const key of ['items', 'data', 'results', 'inspections']) {
    if (Array.isArray(payload[key])) return payload[key].map(normalizeInspectionItem)
  }

  if (payload.data && typeof payload.data === 'object') {
    const nested = listInspections(payload.data)
    if (nested.length) return nested
  }

  const bucketed = []
  for (const [key, value] of Object.entries(payload)) {
    if (!Array.isArray(value) || !value.length) continue
    const bucketStatus = normalizeInspectionStatus(key)
    const fromStatusKey = ['pending', 'in_progress', 'completed'].includes(bucketStatus)
    for (const item of value) {
      const row = item && typeof item === 'object' ? item : { id: item }
      bucketed.push(
        normalizeInspectionItem({
          ...row,
          status: row.status ?? row.state ?? (fromStatusKey ? key : undefined),
        }),
      )
    }
  }
  return bucketed
}

export function countInspections(payload) {
  const items = listInspections(payload)
  const pending = items.filter((item) => item.status === 'pending').length
  const inProgress = items.filter((item) => item.status === 'in_progress').length
  const completed = items.filter((item) => item.status === 'completed').length
  const metaPending = pickNumber(
    payload,
    'pending',
    'pending_count',
    'pendingCount',
    'pending_inspections',
    'total_pending',
  )
  const metaCompleted = pickNumber(
    payload,
    'completed',
    'completed_count',
    'completedCount',
    'completed_inspections',
    'total_completed',
  )

  return {
    items,
    pending: pending || metaPending,
    in_progress: inProgress,
    completed: completed || metaCompleted,
  }
}

export function mergeDashboard(dashboard, inspectionCounts) {
  const source = dashboard && typeof dashboard === 'object' ? dashboard : {}
  const nested = source.summary && typeof source.summary === 'object' ? source.summary : source

  return {
    total_doors: pickNumber(nested, 'total_doors', 'totalDoors'),
    doors_at_risk: pickNumber(nested, 'doors_at_risk', 'doorsAtRisk'),
    pending_inspections: inspectionCounts
      ? inspectionCounts.pending
      : pickNumber(nested, 'pending_inspections', 'pendingInspections'),
    completed_inspections: inspectionCounts
      ? inspectionCounts.completed
      : pickNumber(nested, 'completed_inspections', 'completedInspections'),
    critical_stations: Array.isArray(nested.critical_stations)
      ? nested.critical_stations
      : Array.isArray(nested.criticalStations)
        ? nested.criticalStations
        : [],
  }
}

function formatNameFromEmail(email) {
  const clean = String(email || '').split('@')[0]
  if (!clean) return 'Operador'
  const parts = clean.split(/[._-]/).filter(Boolean)
  if (!parts.length) return 'Operador'
  return parts
    .map((p) => p.charAt(0).toUpperCase() + p.slice(1).toLowerCase())
    .join(' ')
}

export function unwrapUser(payload) {
  const source = payload?.user && typeof payload.user === 'object' ? payload.user : payload
  if (!source || typeof source !== 'object') {
    return { id: '', name: 'Operador', email: '', role: '' }
  }

  const email = source.email || ''
  const name = source.name || formatNameFromEmail(email)

  return {
    id: String(source.id ?? ''),
    name,
    email,
    role: source.role || '',
  }
}

export function uploadIdFrom(result) {
  const payload = result?.data
  const id = payload?.id || payload?.upload_id || payload?.data?.id
  if (id) return String(id)

  const location = result?.headers?.get?.('Location') || result?.headers?.get?.('location')
  if (location) {
    const parts = String(location).split('/').filter(Boolean)
    return parts[parts.length - 1] || null
  }

  return null
}

export function doorRisk(item) {
  if (typeof item?.probability === 'number' && Number.isFinite(item.probability)) {
    return Math.max(0, Math.min(1, item.probability))
  }
  if (typeof item?.failure_probability === 'number' && Number.isFinite(item.failure_probability)) {
    return Math.max(0, Math.min(1, item.failure_probability))
  }
  if (typeof item?.risk === 'number' && Number.isFinite(item.risk)) {
    return Math.max(0, Math.min(1, item.risk))
  }
  return item?.will_fail ? 1.0 : 0.0
}

export function normalizeDoors(payload, stationCode, stationName) {
  return asList(payload).map((item, index) => {
    const willFail = Boolean(item.will_fail ?? item.status === 1)
    const doorNumber = Number(item.door_number ?? index + 1)

    return {
      door_id: item.door_id ?? item.id ?? `${stationCode}-${item.platform || 'P1'}-${doorNumber}`,
      station_name: item.station_name || stationName || stationCode,
      platform: item.platform || 'P1',
      door_number: Number.isFinite(doorNumber) ? doorNumber : index + 1,
      sap_tag: item.sap_tag || '',
      status: item.status ?? (willFail ? 1 : 0),
      will_fail: willFail,
      last_evaluated_at: item.last_evaluated_at || '',
      station_code: item.station_code || stationCode,
      risk: doorRisk({ ...item, will_fail: willFail }),
    }
  })
}

export function relatedMaintenances(items, door) {
  if (!door) return []
  const list = (Array.isArray(items) ? items : asList(items)).map(normalizeMaintenance)
  const pad = String(door.door_number).padStart(2, '0')
  const platformDigit = String(door.platform || '').replace(/\D/g, '')
  return list.filter((item) => {
    const location = String(item.installation_location || '')
    const sameStation = !item.station_code || item.station_code === door.station_code
    const matchesDoor = location.includes(`PP${pad}`)
    const matchesPlatform = !platformDigit || location.includes(`-P${platformDigit}-`)
    return sameStation && matchesDoor && matchesPlatform
  })
}

function maintenanceKey(item) {
  if (item?.id != null && item.id !== '') return `id:${item.id}`
  return `row:${item.sap_order_number}|${item.start_timestamp}|${item.end_timestamp}|${item.station_code}|${item.installation_location}|${item.order_type}`
}

function inspectionKey(item) {
  if (item?.id != null && item.id !== '') return `id:${item.id}`
  return `row:${item.station_code}|${item.platform}|${item.door_number}|${item.status}|${item.notes}`
}

function mergeInspectionLists(...lists) {
  const seen = new Set()
  const all = []
  for (const list of lists) {
    for (const item of list || []) {
      const row = normalizeInspectionItem(item)
      const id = inspectionKey(row)
      if (seen.has(id)) continue
      seen.add(id)
      all.push(row)
    }
  }
  return all
}

async function paginateCollection({
  fetchPage,
  normalizeItem,
  itemKey,
  toList = asList,
  pageSize = 500,
  concurrency = 8,
  onProgress,
  isCancelled,
}) {
  const all = []
  const seen = new Set()
  let pages = 0
  let stalled = false

  const ingest = (payload) => {
    const batch = toList(payload).map(normalizeItem)
    let added = 0
    batch.forEach((item) => {
      const id = itemKey(item)
      if (seen.has(id)) return
      seen.add(id)
      all.push(item)
      added += 1
    })
    pages += 1
    onProgress?.({ loaded: all.length, pages, total: null })
    return { batch, added }
  }

  const firstPayload = await fetchPage(pageSize, 0)
  if (isCancelled?.()) return { items: all, stalled }
  const first = ingest(firstPayload)
  const chunk = first.batch.length
  const knownTotal = listTotal(firstPayload, chunk)
  if (knownTotal) onProgress?.({ loaded: all.length, pages, total: knownTotal })
  if (!chunk) return { items: all, stalled: true }
  if (knownTotal != null && all.length >= knownTotal) return { items: all, stalled: false }

  let nextOffset = chunk
  let finished = false
  const maxRounds = 80

  for (let round = 0; round < maxRounds && !finished; round += 1) {
    if (isCancelled?.()) break

    const offsets = []
    for (let i = 0; i < concurrency; i += 1) {
      const offset = nextOffset + i * chunk
      if (knownTotal != null && offset >= knownTotal) break
      offsets.push(offset)
    }
    nextOffset += offsets.length * chunk
    if (!offsets.length) break

    const results = await Promise.all(
      offsets.map(async (offset) => ({ offset, payload: await fetchPage(pageSize, offset) })),
    )
    results.sort((a, b) => a.offset - b.offset)

    for (const { payload } of results) {
      if (isCancelled?.()) {
        finished = true
        break
      }
      const { batch, added } = ingest(payload)
      if (knownTotal) onProgress?.({ loaded: all.length, pages, total: knownTotal })
      if (batch.length && added === 0) {
        stalled = true
        finished = true
        break
      }
      if (!batch.length || batch.length < chunk) {
        finished = true
        break
      }
      if (knownTotal != null && all.length >= knownTotal) {
        finished = true
        break
      }
    }
  }

  return { items: all, stalled }
}

export const api = {
  login(email, password) {
    return request('/api/v1/auth/login', {
      method: 'POST',
      body: {
        email: String(email ?? '').trim(),
        password: String(password ?? ''),
      },
      auth: false,
    }).then((result) => result.data)
  },
  me() {
    return request('/api/v1/auth/me').then((result) => result.data)
  },
  dashboard() {
    return request('/api/v1/dashboard/summary').then((result) => result.data)
  },
  predictionsLatest() {
    return request('/api/v1/predictions/latest').then((result) => result.data)
  },
  triggerPredictions(horizonDays = 7) {
    return request('/api/v1/predictions/trigger', {
      method: 'POST',
      body: { horizon_days: Number(horizonDays) || 7 },
    }).then((result) => result.data)
  },
  alarms(limit = 50, offset = 0, filters = {}) {
    const query = typeof limit === 'object' ? limit : { limit, offset, ...filters }
    return request(`/api/v1/alarms${queryString(query)}`).then((result) => result.data)
  },
  maintenance(limit = 50, offset = 0) {
    return request(`/api/v1/maintenance${queryString({ limit, offset })}`).then((result) => result.data)
  },
  async listAllMaintenance({
    pageSize = 500,
    concurrency = 8,
    onBatch,
    onProgress,
    isCancelled,
  } = {}) {
    const all = []
    const seen = new Set()
    let pages = 0

    const ingest = (payload) => {
      const batch = asList(payload)
      let added = 0
      batch.forEach((item) => {
        const id = maintenanceKey(item)
        if (seen.has(id)) return
        seen.add(id)
        all.push(normalizeMaintenance(item))
        added += 1
      })
      pages += 1
      onBatch?.(all.slice())
      onProgress?.({ loaded: all.length, pages, total: null })
      return { batch, added }
    }

    const firstPayload = await this.maintenance(pageSize, 0)
    if (isCancelled?.()) return all
    const first = ingest(firstPayload)
    const chunk = first.batch.length
    const knownTotal = listTotal(firstPayload, chunk)
    if (knownTotal) onProgress?.({ loaded: all.length, pages, total: knownTotal })
    if (!chunk) return all
    if (knownTotal != null && all.length >= knownTotal) return all

    let nextOffset = chunk
    let finished = false
    const maxRounds = 80

    for (let round = 0; round < maxRounds && !finished; round += 1) {
      if (isCancelled?.()) break

      const offsets = []
      for (let i = 0; i < concurrency; i += 1) {
        const offset = nextOffset + i * chunk
        if (knownTotal != null && offset >= knownTotal) break
        offsets.push(offset)
      }
      nextOffset += offsets.length * chunk
      if (!offsets.length) break

      const results = await Promise.all(
        offsets.map(async (offset) => ({ offset, payload: await this.maintenance(pageSize, offset) })),
      )
      results.sort((a, b) => a.offset - b.offset)

      for (const { payload } of results) {
        if (isCancelled?.()) {
          finished = true
          break
        }
        const { batch, added } = ingest(payload)
        if (knownTotal) onProgress?.({ loaded: all.length, pages, total: knownTotal })
        if (!batch.length || added === 0 || batch.length < chunk) {
          finished = true
          break
        }
        if (knownTotal != null && all.length >= knownTotal) {
          finished = true
          break
        }
      }
    }

    return all
  },
  createMaintenance(payload) {
    return request('/api/v1/maintenance', { method: 'POST', body: payload }).then((result) => result.data)
  },
  inspections({ status, limit, offset } = {}) {
    if (typeof arguments[0] === 'string') {
      status = arguments[0]
      limit = undefined
      offset = undefined
    }
    return request(`/api/v1/inspections${queryString({ status, limit, offset })}`).then(
      (result) => result.data,
    )
  },
  async fetchInspectionPage(status, limit, offset) {
    try {
      return await this.inspections({ status, limit, offset })
    } catch (err) {
      if (err instanceof ApiError && (err.status === 400 || err.status === 422) && (limit != null || offset != null)) {
        return this.inspections({ status })
      }
      throw err
    }
  },
  async listAllInspections({
    pageSize = 500,
    concurrency = 8,
    onProgress,
    isCancelled,
  } = {}) {
    const paginate = (status) =>
      paginateCollection({
        fetchPage: (limit, offset) => this.fetchInspectionPage(status, limit, offset),
        normalizeItem: normalizeInspectionItem,
        itemKey: inspectionKey,
        toList: listInspections,
        pageSize,
        concurrency,
        onProgress: status ? undefined : onProgress,
        isCancelled,
      })

    const unfiltered = await paginate()
    if (!unfiltered.stalled && unfiltered.items.length > 50) return unfiltered.items

    const buckets = await Promise.all([
      paginate('pending'),
      paginate('in_progress'),
      paginate('completed'),
    ])
    return mergeInspectionLists(
      unfiltered.items,
      buckets[0].items,
      buckets[1].items,
      buckets[2].items,
    )
  },
  createInspection(payload) {
    return request('/api/v1/inspections', { method: 'POST', body: payload }).then((result) => result.data)
  },
  updateInspectionStatus(id, payload) {
    return request(`/api/v1/inspections/${encodeURIComponent(id)}/status`, {
      method: 'PATCH',
      body: payload,
    }).then((result) => result.data)
  },
  stationDoors(code) {
    return request(`/api/v1/stations/${encodeURIComponent(code)}/doors/status`).then((result) => result.data)
  },
  doorTimeline(id) {
    return request(`/api/v1/doors/${encodeURIComponent(id)}/timeline`).then((result) => result.data)
  },
  uploadAlarms(file) {
    const formData = new FormData()
    formData.append('file', file)
    return request('/api/v1/uploads/alarms', { method: 'POST', formData })
  },
  uploadMaintenance(file, orderType) {
    const formData = new FormData()
    formData.append('file', file)
    if (orderType) formData.append('order_type', orderType)
    return request('/api/v1/uploads/maintenance', { method: 'POST', formData })
  },
  uploadStatus(id) {
    return request(`/api/v1/uploads/${encodeURIComponent(id)}`).then((result) => result.data)
  },
}

export async function fetchInspectionCounts() {
  const items = await api.listAllInspections()
  return countInspections(items)
}
