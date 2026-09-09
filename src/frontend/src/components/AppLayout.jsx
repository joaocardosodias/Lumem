import BarraNav from './BarraNav'
import DashboardGrainient from './DashboardGrainient'
import './AppLayout.css'

function AppLayout({ current, onNavigate, onLogout, children }) {
  return (
    <div className="app-shell">
      <BarraNav current={current} onNavigate={onNavigate} onLogout={onLogout} />
      <main className="app-main">
        <DashboardGrainient />
        <div className="app-main__body">{children}</div>
      </main>
    </div>
  )
}

export default AppLayout
