import BarraNav, { NavIcon } from './BarraNav'
import logo from '../assets/logo.svg'
import './AppLayout.css'

const pageLabels = {
  dashboard: 'Visão geral', predicoes: 'Predições', portas: 'Portas',
  alarmes: 'Alarmes', alertas: 'Alarmes', manutencoes: 'Manutenções', inspecoes: 'Inspeções',
  engenharia: 'Engenharia de dados',
}

function AppLayout({ current, user, onNavigate, onLogout, children }) {
  return (
    <div className="app-shell">
      <a className="app-skip-link" href="#conteudo-principal">Ir para o conteúdo</a>
      <BarraNav current={current} user={user} onNavigate={onNavigate} onLogout={onLogout} />
      <main className="app-main">
        <header className="app-topbar">
          <div className="app-topbar__mobile-brand">
            <img src={logo} alt="Motiva" />
          </div>
          <div className="app-topbar__breadcrumb">
            <span>Manutenção preditiva</span>
            <span aria-hidden="true">/</span>
            <strong>{pageLabels[current] || 'Visão geral'}</strong>
          </div>
          <span className="app-topbar__line"><i aria-hidden="true" />Linha 4 · Amarela</span>
          <div className="app-topbar__mobile-account">
            <span className="app-topbar__user" title={user?.name}>{user?.name}</span>
            <button type="button" onClick={onLogout} aria-label="Sair da conta" title="Sair da conta">
              <NavIcon id="sair" />
            </button>
          </div>
        </header>
        <div className="app-main__body" id="conteudo-principal" tabIndex={-1}>{children}</div>
      </main>
    </div>
  )
}

export default AppLayout
