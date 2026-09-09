const TOKEN_KEY = 'motiva.token'

let onUnauthorized = null

export function setUnauthorizedHandler(handler) {
  onUnauthorized = handler
}

export function getToken() {
  return sessionStorage.getItem(TOKEN_KEY)
}

export function setToken(token) {
  if (token) sessionStorage.setItem(TOKEN_KEY, token)
  else sessionStorage.removeItem(TOKEN_KEY)
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

  const response = await fetch(path, {
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

export function unwrapUser(payload) {
  const source = payload?.user && typeof payload.user === 'object' ? payload.user : payload
  if (!source || typeof source !== 'object') {
    return { id: '', name: 'Operador', email: '', role: '' }
  }

  return {
    id: String(source.id ?? ''),
    name: source.name || source.email || 'Operador',
    email: source.email || '',
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
  if (typeof item?.risk === 'number' && Number.isFinite(item.risk)) return item.risk
  return item?.will_fail ? 0.82 : 0.22
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
  const pad = String(door.door_number).padStart(2, '0')
  const platformDigit = String(door.platform || '').replace(/\D/g, '')
  const byLocation = items.filter((item) => {
    const location = String(item.installation_location || '')
    const sameStation = !item.station_code || item.station_code === door.station_code
    const matchesDoor = location.includes(`PP${pad}`)
    const matchesPlatform = !platformDigit || location.includes(`-P${platformDigit}-`)
    return sameStation && matchesDoor && matchesPlatform
  })
  if (byLocation.length) return byLocation
  return items.filter((item) => item.station_code === door.station_code).slice(0, 8)
}

export const api = {
  login(email, password) {
    return request('/api/v1/auth/login', {
      method: 'POST',
      body: { email, password },
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
  triggerPredictions(horizonDays = '7') {
    return request('/api/v1/predictions/trigger', {
      method: 'POST',
      body: { horizon_days: String(horizonDays) },
    }).then((result) => result.data)
  },
  maintenance(limit = 50, offset = 0) {
    return request(`/api/v1/maintenance${queryString({ limit, offset })}`).then((result) => result.data)
  },
  createMaintenance(payload) {
    return request('/api/v1/maintenance', { method: 'POST', body: payload }).then((result) => result.data)
  },
  inspections(status) {
    return request(`/api/v1/inspections${queryString({ status })}`).then((result) => result.data)
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
