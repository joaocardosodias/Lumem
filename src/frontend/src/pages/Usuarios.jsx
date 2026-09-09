import Button from '../components/Button'
import { users } from '../data/mock'
import './Usuarios.css'

function Usuarios() {
  return (
    <section>
      <div className="page-head">
        <h1>Usuários</h1>
        <Button>Cadastrar Usuários +</Button>
      </div>

      <div className="usuarios__list">
        {users.map((user) => (
          <article key={user.id} className="card-usuario">
            <div className="card-usuario__avatar" />
            <div>
              <strong>{user.name}</strong>
              <span>{user.role}</span>
            </div>
            <div className="card-usuario__actions">
              <button type="button" aria-label="Excluir">
                ⌫
              </button>
              <button type="button" aria-label="Editar">
                ✎
              </button>
            </div>
          </article>
        ))}
      </div>
    </section>
  )
}

export default Usuarios
