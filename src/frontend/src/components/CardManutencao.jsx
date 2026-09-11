function formatDuration(hours) {
  if (hours == null || hours === '') return '—'
  const value = Number(hours)
  if (!Number.isFinite(value)) return '—'
  return `${value.toLocaleString('pt-BR', { maximumFractionDigits: 2 })} h`
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
    <details className="maintenance-record">
      <summary className="card-manutencao">
        <div>
          <span>Ordem SAP</span>
          <strong>{sapOrder || '—'}</strong>
          <em>{orderType || 'Tipo não informado'}</em>
        </div>
        <div>
          <span>Estação e local</span>
          <strong>{station || '—'}</strong>
          <em>{location || 'Local não informado'}</em>
        </div>
        <div>
          <span>Período</span>
          <strong>{startedAt || '—'}</strong>
          <em>{endedAt ? `até ${endedAt}` : 'Fim não informado'}</em>
        </div>
        <div>
          <span>Duração</span>
          <strong>{formatDuration(durationHours)}</strong>
        </div>
        <div>
          <span>Ocorrência</span>
          <strong>{symptom || 'Sem sintoma informado'}</strong>
          <em>{description || 'Sem descrição'}</em>
        </div>
        <span className="maintenance-record__toggle">
          Detalhes
          <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
            <path d="m4 6 4 4 4-4" />
          </svg>
        </span>
      </summary>
      <dl className="maintenance-record__details">
        <div><dt>Ordem SAP</dt><dd>{sapOrder || 'Não informada'}</dd></div>
        <div><dt>Tipo</dt><dd>{orderType || 'Não informado'}</dd></div>
        <div><dt>Estação</dt><dd>{station || 'Não informada'}</dd></div>
        <div><dt>Origem</dt><dd>{source || 'Não informada'}</dd></div>
        <div><dt>Início</dt><dd>{startedAt || 'Não informado'}</dd></div>
        <div><dt>Fim</dt><dd>{endedAt || 'Não informado'}</dd></div>
        <div className="maintenance-record__wide"><dt>Local de instalação</dt><dd>{location || 'Não informado'}</dd></div>
        <div className="maintenance-record__wide"><dt>Sintoma</dt><dd>{symptom || 'Não informado'}</dd></div>
        <div className="maintenance-record__wide"><dt>Descrição</dt><dd>{description || 'Não informada'}</dd></div>
      </dl>
    </details>
  )
}

export default CardManutencao
