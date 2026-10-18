import { useEffect, useState } from 'react'
import AppLayout from './components/AppLayout'
import { PredictionLoadProvider } from './components/PredictionLoad'
import { AuthProvider, useAuth } from './context/Auth'
import { ToastProvider } from './context/Toast'
import { getCurrentPage, getCurrentParams, navigateTo, subscribeToNavigation } from './lib/router'
import Alarmes from './pages/Alarmes'
import Dashboard from './pages/Dashboard'
import Inspecoes from './pages/Inspecoes'
import Login from './pages/Login'
import Manutencoes from './pages/Manutencoes'
import Portas from './pages/Portas'
import Predicoes from './pages/Predicoes'
import Engenharia from './pages/Engenharia'

const applicationPages = ['dashboard', 'predicoes', 'portas', 'alarmes', 'manutencoes', 'inspecoes', 'engenharia']

function normalizePage(page) {
  if (page === 'alertas') return 'alarmes'
  return applicationPages.includes(page) ? page : 'dashboard'
}

function AppShell() {
  const { user, ready, logout } = useAuth()
  const [page, setPage] = useState(() => normalizePage(getCurrentPage()))
  const [mountedPages, setMountedPages] = useState(() => new Set([normalizePage(getCurrentPage())]))
  const [paramsByPage, setParamsByPage] = useState(() => ({
    [normalizePage(getCurrentPage())]: getCurrentParams(),
  }))

  function activatePage(target, targetParams = {}) {
    const normalized = normalizePage(target)
    setPage(normalized)
    setParamsByPage((current) => ({ ...current, [normalized]: targetParams }))
    setMountedPages((current) => {
      if (current.has(normalized)) return current
      const next = new Set(current)
      next.add(normalized)
      return next
    })
  }

  useEffect(() => {
    return subscribeToNavigation((nextPage, nextParams) => {
      activatePage(nextPage, nextParams)
    })
  }, [])

  useEffect(() => {
    const discardHiddenPages = () => setMountedPages(new Set([page]))
    window.addEventListener('motiva:cache-invalidated', discardHiddenPages)
    return () => window.removeEventListener('motiva:cache-invalidated', discardHiddenPages)
  }, [page])

  function handleNavigate(target, targetParams = {}, replace = false) {
    const normalized = normalizePage(target)
    navigateTo(normalized, targetParams, replace)
    activatePage(normalized, targetParams)
  }

  function handleLogout() {
    handleNavigate('dashboard')
    logout()
    setMountedPages(new Set(['dashboard']))
    setParamsByPage({ dashboard: {} })
  }

  function renderPage(target) {
    const params = paramsByPage[target] || {}
    if (target === 'dashboard') return <Dashboard onNavigate={handleNavigate} />
    if (target === 'predicoes') return <Predicoes onNavigate={handleNavigate} params={params} />
    if (target === 'portas') return <Portas onNavigate={handleNavigate} params={params} />
    if (target === 'alarmes') return <Alarmes onNavigate={handleNavigate} params={params} />
    if (target === 'manutencoes') return <Manutencoes onNavigate={handleNavigate} params={params} />
    if (target === 'inspecoes') return <Inspecoes onNavigate={handleNavigate} params={params} />
    if (target === 'engenharia') return <Engenharia onNavigate={handleNavigate} params={params} />
    return <Dashboard onNavigate={handleNavigate} />
  }

  if (!ready) return <div className="app-boot" />

  return (
    <PredictionLoadProvider>
      {!user ? (
        <Login />
      ) : (
        <AppLayout current={page} user={user} onNavigate={handleNavigate} onLogout={handleLogout}>
          {applicationPages.map((pageId) => {
            const isAllowed = pageId !== 'engenharia' || user.role === 'enginner'
            const isMounted = mountedPages.has(pageId) || pageId === page
            if (!isAllowed || !isMounted) return null
            const isActive = pageId === page
            return (
              <div
                key={pageId}
                className="app-page-cache"
                data-page={pageId}
                hidden={!isActive}
                aria-hidden={!isActive}
              >
                {renderPage(pageId)}
              </div>
            )
          })}
        </AppLayout>
      )}
    </PredictionLoadProvider>
  )
}

function App() {
  return (
    <AuthProvider>
      <ToastProvider>
        <AppShell />
      </ToastProvider>
    </AuthProvider>
  )
}

export default App
