import { PERIODS } from '../lib/filters'
import './Filtros.css'

function Filtros({ children, dirty, onClear, period, onPeriod }) {
  return (
    <div className="filtros">
      <div className="filtros__periodos">
        <span>Período</span>
        <div>
          {PERIODS.map((item) => (
            <button
              key={item.id}
              type="button"
              className={period === item.id ? 'is-active' : ''}
              onClick={() => onPeriod(item.id)}
            >
              {item.label}
            </button>
          ))}
        </div>
      </div>
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
