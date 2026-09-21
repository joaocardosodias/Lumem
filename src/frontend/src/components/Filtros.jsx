import { PERIODS } from '../lib/filters'
import './Filtros.css'

function Filtros({ children, dirty, onClear, period, onPeriod, showPeriod = true }) {
  return (
    <div className="filtros">
      {showPeriod && onPeriod ? (
        <div className="filtros__periodos">
          <div className="filtros__periodos-copy">
            <span>Período</span>
            <small>Janela de análise</small>
          </div>
          <div className="filtros__periodos-control" role="group" aria-label="Período de análise">
          {PERIODS.map((item) => (
            <button
              key={item.id}
              type="button"
              className={period === item.id ? 'is-active' : ''}
              aria-pressed={period === item.id}
              onClick={() => onPeriod(item.id)}
            >
              {item.label}
            </button>
          ))}
          </div>
        </div>
      ) : null}
      <div className="filtros__campos">
        {children}
        {dirty ? (
          <button type="button" className="filtros__clear" onClick={onClear}>
            Limpar filtros
          </button>
        ) : null}
      </div>
    </div>
  )
}

export function FiltroCampo({ label, children }) {
  return (
    <label className="filtros__campo">
      <span>{label}</span>
      {children}
    </label>
  )
}

export default Filtros
