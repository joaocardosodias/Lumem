import './Card.css'

function Card({ label, value }) {
  return (
    <article className="kpi-card">
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
