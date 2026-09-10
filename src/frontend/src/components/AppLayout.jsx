import BarraNav from './BarraNav'
import DashboardGrainient from './DashboardGrainient'
import './AppLayout.css'

function AppLayout({ current, user, onNavigate, onLogout, children }) {
  return (
    <div className="app-shell">
      <BarraNav current={current} user={user} onNavigate={onNavigate} onLogout={onLogout} />
      <main className="app-main">
        <DashboardGrainient />
        <div className="app-main__body">{children}</div>
      </main>
    </div>
  )
}

export default AppLayout
