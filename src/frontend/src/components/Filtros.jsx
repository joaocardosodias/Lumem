import './Filtros.css'

function Filtros({ children, dirty, onClear }) {
  return (
    <div className="filtros">
      {children}
      {dirty ? (
        <button type="button" className="filtros__clear" onClick={onClear}>
          Limpar filtros
        </button>
      ) : null}
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
