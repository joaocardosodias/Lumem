import iconMotiva from '../assets/icon_motiva.svg'
import BarraNav from './BarraNav'
import './AppLayout.css'

function AppLayout({ current, onNavigate, onLogout, children }) {
  return (
    <div className="app-shell">
      <BarraNav current={current} onNavigate={onNavigate} onLogout={onLogout} />
      <main className="app-main">
        <div
          className="app-main__mark"
          style={{
            maskImage: `url(${iconMotiva})`,
            WebkitMaskImage: `url(${iconMotiva})`,
          }}
          aria-hidden="true"
        />
        {children}
      </main>
    </div>
  )
}

export default AppLayout
