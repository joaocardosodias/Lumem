import { useState } from 'react'
import iconMotiva from '../assets/icon_motiva.svg'
import logo from '../assets/logo.svg'
import Button from '../components/Button'
import Input from '../components/Input'
import { mockAuth } from '../data/mock'
import './Login.css'

function Login({ onLogin }) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  function handleSubmit(event) {
    event.preventDefault()

    const validEmail = email.trim().toLowerCase() === mockAuth.email.toLowerCase()
    const validPassword = password === mockAuth.password

    if (!validEmail || !validPassword) {
      setError('E-mail ou senha inválidos.')
      return
    }

    setError('')
    onLogin()
  }

  return (
    <section className="login">
      <aside className="login__brand">
        <img className="login__logo" src={logo} alt="motiva" />

        <div className="login__intro">
          <p>Manutenção preditiva das portas de plataforma</p>
          <strong>Linha 4 Amarela</strong>
        </div>
      </aside>

      <div className="login__panel">
        <img className="login__panel-mark" src={iconMotiva} alt="" aria-hidden="true" />

        <form className="login__form" onSubmit={handleSubmit}>
          <div className="login__form-head">
            <span>Acesso à plataforma</span>
            <h1>Entrar</h1>
          </div>

          <Input
            id="email"
            label="Endereço de e-mail"
            type="email"
            value={email}
            autoComplete="username"
            onChange={(event) => setEmail(event.target.value)}
          />
          <Input
            id="password"
            label="Senha"
            type="password"
            value={password}
            autoComplete="current-password"
            onChange={(event) => setPassword(event.target.value)}
          />

          {error ? <p className="login__error">{error}</p> : null}

          <Button type="submit">Logar</Button>
        </form>
      </div>
    </section>
  )
}

export default Login
