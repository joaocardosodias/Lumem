import { useEffect, useState } from 'react'
import AppLayout from './components/AppLayout'
import { PredictionLoadProvider } from './components/PredictionLoad'
import { AuthProvider, useAuth } from './context/Auth'
import { ToastProvider } from './context/Toast'
import { getCurrentPage, getCurrentParams, navigateTo, subscribeToNavigation } from './lib/router'
import Alertas from './pages/Alertas'
import Dashboard from './pages/Dashboard'
import Inspecoes from './pages/Inspecoes'
import Login from './pages/Login'
import Manutencoes from './pages/Manutencoes'
import Portas from './pages/Portas'

function AppShell() {
  const { user, ready, logout } = useAuth()
  const [page, setPage] = useState(getCurrentPage)
  const [params, setParams] = useState(getCurrentParams)

  useEffect(() => {
    return subscribeToNavigation((nextPage, nextParams) => {
      setPage(nextPage)
      setParams(nextParams)
    })
  }, [])

  function handleNavigate(target, targetParams = {}, replace = false) {
    navigateTo(target, targetParams, replace)
    setPage(target)
    setParams(targetParams)
  }

  function handleLogout() {
    handleNavigate('dashboard')
    logout()
  }

  function renderPage() {
    if (page === 'dashboard') return <Dashboard onNavigate={handleNavigate} />
    if (page === 'alertas') return <Alertas onNavigate={handleNavigate} params={params} />
    if (page === 'manutencoes') return <Manutencoes onNavigate={handleNavigate} params={params} />
    if (page === 'portas') return <Portas onNavigate={handleNavigate} params={params} />
    return <Inspecoes onNavigate={handleNavigate} params={params} />
  }

  if (!ready) return <div className="app-boot" />

  return (
    <PredictionLoadProvider>
      {!user ? (
        <Login />
      ) : (
        <AppLayout current={page} user={user} onNavigate={handleNavigate} onLogout={handleLogout}>
          {renderPage()}
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
