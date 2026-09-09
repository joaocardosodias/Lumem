function formatDuration(hours) {
  if (hours == null || hours === '') return '—'
  const value = Number(hours)
  if (!Number.isFinite(value)) return '—'
  return `${value} h`
}

function CardManutencao({
  orderType,
  sapOrder,
  station,
  location,
  startedAt,
  endedAt,
  durationHours,
  symptom,
  description,
  source,
}) {
  return (
    <article className="card-manutencao">
      <div>
        <span>Tipo</span>
        <strong>{orderType}</strong>
      </div>
      <div>
        <span>Ordem SAP</span>
        <strong>{sapOrder || '—'}</strong>
      </div>
      <div>
        <span>Estação</span>
        <strong>{station}</strong>
      </div>
      <div>
        <span>Local</span>
        <strong>{location || '—'}</strong>
      </div>
      <div>
        <span>Início</span>
        <strong>{startedAt}</strong>
      </div>
      <div>
        <span>Fim</span>
        <strong>{endedAt || '—'}</strong>
      </div>
      <div>
        <span>Duração</span>
        <strong>{formatDuration(durationHours)}</strong>
      </div>
      <div>
        <span>Sintoma</span>
        <strong>{symptom || '—'}</strong>
      </div>
      <div>
        <span>Descrição</span>
        <strong>{description || '—'}</strong>
      </div>
      <div>
        <span>Origem</span>
        <strong>{source || '—'}</strong>
      </div>
      <button type="button" aria-label="Editar manutenção">
        ✎
      </button>
    </article>
  )
}

export default CardManutencao
