function CardAlerta({ tag, startedAt, station, alarm, normalizedAt }) {
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
        <span>Alarme</span>
        <strong>{alarm}</strong>
      </div>
      <div>
        <span>Data e Hora de Normalização</span>
        <strong>{normalizedAt}</strong>
      </div>
      <button type="button" aria-label="Editar alerta">
        ✎
      </button>
    </article>
  )
}

export default CardAlerta
