import Button from '../components/Button'
import { maintenances } from '../data/mock'
import './Manutencoes.css'

function Manutencoes() {
  return (
    <section>
      <div className="page-head">
        <h1>Manutenções</h1>
        <Button>Cadastrar Manutenções +</Button>
      </div>

      <div className="manutencoes__list">
        {maintenances.map((item) => (
          <article key={item.ordem} className="card-manutencao">
            <div>
              <span>Tipo</span>
              <strong>{item.tipo}</strong>
            </div>
            <div>
              <span>Nota</span>
              <strong>{item.nota}</strong>
            </div>
            <div>
              <span>Ordem</span>
              <strong>{item.ordem}</strong>
            </div>
            <div>
              <span>Indisponibilidade</span>
              <strong>{item.indisponibilidade}</strong>
            </div>
            <div>
              <span>Falha</span>
              <strong>{item.falha}</strong>
            </div>
            <div>
              <span>Analisada</span>
              <strong>{item.analisada}</strong>
            </div>
            <div>
              <span>CCE</span>
              <strong>{item.cce}</strong>
            </div>
            <div>
              <span>Perfil do Catálogo</span>
              <strong>{item.perfil}</strong>
            </div>
          </article>
        ))}
      </div>
    </section>
  )
}

export default Manutencoes
