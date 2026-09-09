import iconMotiva from '../assets/icon_motiva.svg'
import './Card.css'

function Card({ label, value }) {
  return (
    <article className="kpi-card">
      <div
        className="kpi-card__mark"
        style={{
          maskImage: `url(${iconMotiva})`,
          WebkitMaskImage: `url(${iconMotiva})`,
        }}
        aria-hidden="true"
      />
      <div className="kpi-card__content">
        <p>{label}</p>
        <div className="kpi-card__value">
          <strong>{value}</strong>
          <i />
        </div>
      </div>
    </article>
  )
}

export default Card
