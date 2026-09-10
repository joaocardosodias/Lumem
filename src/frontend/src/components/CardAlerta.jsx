function CardAlerta({
  tag,
  startedAt,
  station,
  alarm,
  normalizedAt,
  willFail,
  onViewDoor,
  onOpenInspection,
}) {
  return (
    <article className="card-alerta">
      <div>
        <span>Tag</span>
        <strong>{tag}</strong>
      </div>
      <div>
        <span>Data e Hora</span>
        <strong>{startedAt}</strong>
      </div>
      <div>
        <span>Estação</span>
        <strong>{station}</strong>
      </div>
      <div>
        <span>Diagnóstico</span>
        <strong className={willFail ? 'card-alerta__status--risk' : 'card-alerta__status--ok'}>
          {alarm}
        </strong>
      </div>
      <div>
        <span>Plataforma</span>
        <strong>{normalizedAt}</strong>
      </div>
      <div className="card-alerta__actions">
        {onViewDoor ? (
          <button type="button" className="card-alerta__btn" onClick={onViewDoor}>
            Ver Porta
          </button>
        ) : null}
        {onOpenInspection && willFail ? (
          <button
            type="button"
            className="card-alerta__btn card-alerta__btn--primary"
            onClick={onOpenInspection}
          >
            Inspecionar
          </button>
        ) : null}
      </div>
    </article>
  )
}

export default CardAlerta
