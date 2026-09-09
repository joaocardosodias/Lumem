import './Paginacao.css'

function Paginacao({ page, pageSize, total, onPage }) {
  if (!total) return null

  const pageCount = Math.max(1, Math.ceil(total / pageSize))
  const current = Math.min(Math.max(page, 1), pageCount)
  const from = (current - 1) * pageSize + 1
  const to = Math.min(current * pageSize, total)

  return (
    <div className="paginacao">
      <button type="button" disabled={current <= 1} onClick={() => onPage(current - 1)}>
        Anterior
      </button>
      <span>
        {from}–{to} de {total}
      </span>
      <button type="button" disabled={current >= pageCount} onClick={() => onPage(current + 1)}>
        Próxima
      </button>
    </div>
  )
}

export default Paginacao
