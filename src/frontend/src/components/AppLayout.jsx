import BarraNav from './BarraNav'
import './AppLayout.css'

function AppLayout({ current, onNavigate, onLogout, children }) {
  return (
    <div className="app-shell">
      <BarraNav current={current} onNavigate={onNavigate} onLogout={onLogout} />
      <main className="app-main">{children}</main>
    </div>
  )
}

export default AppLayout
