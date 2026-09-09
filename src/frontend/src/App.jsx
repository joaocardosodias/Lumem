import { useState } from 'react'
import AppLayout from './components/AppLayout'
import { PredictionLoadProvider } from './components/PredictionLoad'
import Alertas from './pages/Alertas'
import Dashboard from './pages/Dashboard'
import Inspecoes from './pages/Inspecoes'
import Login from './pages/Login'
import Manutencoes from './pages/Manutencoes'
import Placeholder from './pages/Placeholder'
import Portas from './pages/Portas'
import Usuarios from './pages/Usuarios'

function App() {
  const [loggedIn, setLoggedIn] = useState(false)
  const [page, setPage] = useState('dashboard')

  function handleLogin() {
    setPage('dashboard')
    setLoggedIn(true)
  }

  function handleLogout() {
    setLoggedIn(false)
    setPage('dashboard')
  }

  function renderPage() {
    if (page === 'dashboard') return <Dashboard onNavigate={setPage} />
    if (page === 'alertas') return <Alertas />
    if (page === 'manutencoes') return <Manutencoes />
    if (page === 'portas') return <Portas onNavigate={setPage} />
    if (page === 'inspecoes') return <Inspecoes />
    if (page === 'usuarios') return <Usuarios />
    return <Placeholder title="Configurações" />
  }

  return (
    <PredictionLoadProvider>
      {!loggedIn ? (
        <Login onLogin={handleLogin} />
      ) : (
        <AppLayout current={page} onNavigate={setPage} onLogout={handleLogout}>
          {renderPage()}
        </AppLayout>
      )}
    </PredictionLoadProvider>
  )
}

export default App
