import iconMotiva from '../assets/icon_motiva.svg'
import './Card.css'

function Card({ label, value, hint }) {
  return (
    <article className="kpi-card">
      <img className="kpi-card__mark" src={iconMotiva} alt="" aria-hidden="true" />
      <div className="kpi-card__content">
        <p>{label}</p>
        <div className="kpi-card__value">
          <strong>{value}</strong>
          <i />
        </div>
        {hint ? <small>{hint}</small> : null}
      </div>
    </article>
  )
}

export default Card
