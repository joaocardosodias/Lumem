import { useState } from 'react'
import iconMotiva from '../assets/icon_motiva.svg'
import logo from '../assets/logo.svg'
import './BarraNav.css'

const items = [
  { id: 'dashboard', label: 'Visão geral' },
  { id: 'predicoes', label: 'Predições' },
  { id: 'portas', label: 'Portas' },
  { id: 'alarmes', label: 'Alarmes' },
  { id: 'manutencoes', label: 'Manutenções' },
  { id: 'inspecoes', label: 'Inspeções' },
]

function BarraNav({ current, user, onNavigate, onLogout }) {
  const [collapsed, setCollapsed] = useState(false)
  const activeIndex = Math.max(
    items.findIndex((item) => item.id === current),
    0,
  )

  return (
    <aside className={`barra-nav${collapsed ? ' is-collapsed' : ''}`}>
      <div className="barra-nav__brand">
        <img
          className="barra-nav__logo"
          src={collapsed ? iconMotiva : logo}
          alt="motiva"
        />
        <span className="barra-nav__brand-caption">Manutenção preditiva</span>
      </div>

      <p className="barra-nav__section-label">Operação</p>
      <nav aria-label="Navegação principal">
        <div className="barra-nav__indicator" style={{ '--i': String(activeIndex) }} aria-hidden="true" />
        <ul>
          {items.map((item) => (
            <li key={item.id}>
              <button
                type="button"
                className={current === item.id ? 'is-active' : undefined}
                onClick={() => onNavigate(item.id)}
                title={collapsed ? item.label : undefined}
                aria-label={item.label}
                aria-current={current === item.id ? 'page' : undefined}
              >
                <NavIcon id={item.id} />
                <span className="barra-nav__label">{item.label}</span>
              </button>
            </li>
          ))}
        </ul>
      </nav>

      {user ? (
        <div className="barra-nav__user" title={`${user.name} (${user.email || user.role || 'Operador'})`}>
          <div className="barra-nav__user-avatar">{user.name ? user.name.charAt(0).toUpperCase() : 'O'}</div>
          {!collapsed ? (
            <div className="barra-nav__user-info">
              <strong>{user.name}</strong>
              <small>{user.role === 'operator' ? 'Operador' : user.role || 'Operador Linha 4'}</small>
            </div>
          ) : null}
        </div>
      ) : null}

      <div className="barra-nav__footer">
        <button
          type="button"
          className="barra-nav__toggle"
          onClick={() => setCollapsed((value) => !value)}
          aria-label={collapsed ? 'Expandir menu' : 'Minimizar menu'}
          aria-expanded={!collapsed}
          title={collapsed ? 'Expandir menu' : 'Minimizar menu'}
        >
          <ToggleIcon collapsed={collapsed} />
        </button>
        <button
          type="button"
          className="barra-nav__sair"
          onClick={onLogout}
          title={collapsed ? 'Sair' : undefined}
          aria-label="Sair"
        >
          <NavIcon id="sair" />
          <span className="barra-nav__label">Sair</span>
        </button>
      </div>
    </aside>
  )
}

function ToggleIcon({ collapsed }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      {collapsed ? (
        <path d="M9 6l6 6-6 6" />
      ) : (
        <path d="M15 6l-6 6 6 6" />
      )}
    </svg>
  )
}

export function NavIcon({ id }) {
  if (id === 'completed') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <circle cx="12" cy="12" r="8.5" />
        <path d="m8 12 3 3 5-6" />
      </svg>
    )
  }

  if (id === 'station') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <path d="M3 12h5m8 0h5M12 3v5m0 8v5" />
        <circle cx="12" cy="12" r="4" />
      </svg>
    )
  }

  if (id === 'dashboard') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8">
        <rect x="3" y="4" width="18" height="12" rx="2" />
        <path d="M8 20h8M12 16v4" />
      </svg>
    )
  }

  if (id === 'predicoes') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" />
      </svg>
    )
  }

  if (id === 'alarmes' || id === 'alertas') {
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
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <path d="M4 21V5a2 2 0 0 1 2-2h7v18H6a2 2 0 0 1-2-2Z" />
        <path d="M13 3.2 20 5v14l-7 1.8" />
        <path d="M16.5 12h.01" />
      </svg>
    )
  }

  if (id === 'inspecoes') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <path d="M9 5H7a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2h-2" />
        <rect x="9" y="3" width="6" height="4" rx="1" />
        <path d="M9 12h6M9 16h4" />
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
