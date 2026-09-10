const ROUTES = {
  dashboard: '/dashboard',
  predicoes: '/predicoes',
  portas: '/portas',
  alarmes: '/alarmes',
  manutencoes: '/manutencoes',
  inspecoes: '/inspecoes',
  alertas: '/alarmes',
}

const PATH_TO_PAGE = {
  '/': 'dashboard',
  '/dashboard': 'dashboard',
  '/predicoes': 'predicoes',
  '/predictions': 'predicoes',
  '/alarmes': 'alarmes',
  '/alertas': 'alarmes',
  '/alerts': 'alarmes',
  '/manutencoes': 'manutencoes',
  '/maintenance': 'manutencoes',
  '/portas': 'portas',
  '/doors': 'portas',
  '/inspecoes': 'inspecoes',
  '/inspections': 'inspecoes',
}

export function getCurrentPage() {
  if (typeof window === 'undefined') return 'dashboard'
  const path = window.location.pathname.replace(/\/$/, '') || '/'
  return PATH_TO_PAGE[path] || 'dashboard'
}

export function getCurrentParams() {
  if (typeof window === 'undefined') return {}
  const search = new URLSearchParams(window.location.search)
  const params = {}
  for (const [key, value] of search.entries()) {
    params[key] = value
  }
  return params
}

export function getPagePath(page, params = {}) {
  const base = ROUTES[page] || '/dashboard'
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value != null && value !== '') {
      search.set(key, String(value))
    }
  }
  const query = search.toString()
  return query ? `${base}?${query}` : base
}

export function navigateTo(page, params = {}, replace = false) {
  if (typeof window === 'undefined') return
  const path = getPagePath(page, params)
  const currentFull = window.location.pathname + window.location.search
  if (currentFull === path) return

  if (replace) {
    window.history.replaceState(null, '', path)
  } else {
    window.history.pushState(null, '', path)
  }
  window.dispatchEvent(new Event('app:navigate'))
}

export function subscribeToNavigation(callback) {
  if (typeof window === 'undefined') return () => {}

  const handler = () => {
    callback(getCurrentPage(), getCurrentParams())
  }

  window.addEventListener('popstate', handler)
  window.addEventListener('app:navigate', handler)

  return () => {
    window.removeEventListener('popstate', handler)
    window.removeEventListener('app:navigate', handler)
  }
}
