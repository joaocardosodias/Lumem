import logo from '../assets/logo.svg'
import './BarraNav.css'

const items = [
  { id: 'dashboard', label: 'Dashboard' },
  { id: 'alertas', label: 'Alertas' },
  { id: 'manutencoes', label: 'Manutenções' },
  { id: 'portas', label: 'Portas' },
  { id: 'analise', label: 'Analise' },
  { id: 'configuracoes', label: 'Configurações' },
  { id: 'usuarios', label: 'Usuários' },
]

function BarraNav({ current, onNavigate, onLogout }) {
  return (
    <aside className="barra-nav">
      <img className="barra-nav__logo" src={logo} alt="motiva" />

      <nav>
        <ul>
          {items.map((item) => (
            <li key={item.id}>
              <button
                type="button"
                className={current === item.id ? 'is-active' : undefined}
                onClick={() => onNavigate(item.id)}
              >
                <NavIcon id={item.id} />
                {item.label}
              </button>
            </li>
          ))}
        </ul>
      </nav>

      <button type="button" className="barra-nav__sair" onClick={onLogout}>
        <NavIcon id="sair" />
        Sair
      </button>
    </aside>
  )
}

function NavIcon({ id }) {
  if (id === 'dashboard') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
        <rect x="3" y="4" width="18" height="12" rx="2" />
        <path d="M8 20h8M12 16v4" />
      </svg>
    )
  }

  if (id === 'alertas') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
        <path d="M7 10a5 5 0 0 1 10 0c0 5 2 6 2 7H5c0-1 2-2 2-7Z" />
        <path d="M10 20a2 2 0 0 0 4 0" />
      </svg>
    )
  }

  if (id === 'manutencoes') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
        <path d="M14.7 6.3a4 4 0 0 0-5.7 5.7L4 17l3 3 5-5a4 4 0 0 0 5.7-5.7L15 12l-3-3z" />
      </svg>
    )
  }

  if (id === 'portas') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
        <path d="M6 21V5a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v16H6Z" />
        <path d="M14 12h.01" />
      </svg>
    )
  }

  if (id === 'analise') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
        <path d="M12 3a9 9 0 1 0 9 9h-9V3Z" />
        <path d="M13.5 3.2A9 9 0 0 1 20.8 10.5H13.5V3.2Z" />
      </svg>
    )
  }

  if (id === 'configuracoes') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
        <circle cx="12" cy="12" r="3" />
        <path d="M12 4v2M12 18v2M4 12h2M18 12h2M6.3 6.3l1.4 1.4M16.3 16.3l1.4 1.4M17.7 6.3l-1.4 1.4M7.7 16.3l-1.4 1.4" />
      </svg>
    )
  }

  if (id === 'usuarios') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
        <circle cx="12" cy="8" r="3" />
        <path d="M5 19c1.3-3 3.8-4.5 7-4.5s5.7 1.5 7 4.5" />
      </svg>
    )
  }

  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
      <path d="M10 7V5a2 2 0 0 1 2-2h7v18h-7a2 2 0 0 1-2-2v-2" />
      <path d="M4 12h11M12 8l4 4-4 4" />
    </svg>
  )
}

export default BarraNav
