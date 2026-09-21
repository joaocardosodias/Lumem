import iconMotiva from '../assets/icon_motiva.svg'
import './Card.css'

export function CountValue({ value }) {
  const numeric = typeof value === 'number' ? value : Number(value)
  const isNumeric = Number.isFinite(numeric)
  if (!isNumeric) return value ?? '—'
  return <span>{Math.round(numeric).toLocaleString('pt-BR')}</span>
}

function Card({ label, value, hint, layout = 'hero' }) {
  return (
    <article className={`kpi-card${layout === 'banner' ? ' kpi-card--banner' : ''}`}>
      <img className="kpi-card__mark" src={iconMotiva} alt="" aria-hidden="true" />
      <div className="kpi-card__content">
        {layout === 'banner' ? (
          <>
            <div>
              <p>{label}</p>
              {hint ? <small>{hint}</small> : null}
            </div>
            <div className="kpi-card__value">
              <strong>
                <CountValue value={value} />
              </strong>
            </div>
          </>
        ) : (
          <>
            <p>{label}</p>
            <div className="kpi-card__value">
              <strong>
                <CountValue value={value} />
              </strong>
            </div>
            {hint ? <small>{hint}</small> : null}
          </>
        )}
      </div>
    </article>
  )
}

export default Card
