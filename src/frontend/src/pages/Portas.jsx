import { portasResumo } from '../data/mock'
import './Portas.css'

function Portas() {
  return (
    <section className="portas">
      <div className="page-head">
        <div>
          <h1>Portas</h1>
          <p>Linha 4 Amarela</p>
        </div>
      </div>

      <div className="portas__cards">
        {portasResumo.map((item) => (
          <article key={item.label} className="card-metrica">
            <span>{item.label}</span>
            <strong>{item.value}</strong>
          </article>
        ))}
      </div>
    </section>
  )
}

export default Portas
