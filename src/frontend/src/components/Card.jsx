import iconMotiva from '../assets/icon_motiva.svg'
import './Card.css'

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
              <strong>{value}</strong>
              <i />
            </div>
          </>
        ) : (
          <>
            <p>{label}</p>
            <div className="kpi-card__value">
              <strong>{value}</strong>
              <i />
            </div>
            {hint ? <small>{hint}</small> : null}
          </>
        )}
      </div>
    </article>
  )
}

export default Card
