import { useState } from 'react'
import AppLayout from './components/AppLayout'
import { PredictionLoadProvider } from './components/PredictionLoad'
import { AuthProvider, useAuth } from './context/Auth'
import Alertas from './pages/Alertas'
import Dashboard from './pages/Dashboard'
import Inspecoes from './pages/Inspecoes'
import Login from './pages/Login'
import Manutencoes from './pages/Manutencoes'
import Portas from './pages/Portas'

function AppShell() {
  const { user, ready, logout } = useAuth()
  const [page, setPage] = useState('dashboard')

  function handleLogout() {
    setPage('dashboard')
    logout()
  }

  function renderPage() {
    if (page === 'dashboard') return <Dashboard onNavigate={setPage} />
    if (page === 'alertas') return <Alertas />
    if (page === 'manutencoes') return <Manutencoes />
    if (page === 'portas') return <Portas onNavigate={setPage} />
    return <Inspecoes />
  }

  if (!ready) return <div className="app-boot" />

  return (
    <PredictionLoadProvider>
      {!user ? (
        <Login />
      ) : (
        <AppLayout current={page} onNavigate={setPage} onLogout={handleLogout}>
          {renderPage()}
        </AppLayout>
      )}
    </PredictionLoadProvider>
  )
}

function App() {
  return (
    <AuthProvider>
      <AppShell />
    </AuthProvider>
  )
}

export default App
