import { useRef, useState } from 'react'
import iconMotiva from '../assets/icon_motiva.svg'
import logo from '../assets/logo.svg'
import Button from '../components/Button'
import Grainient from '../components/Grainient'
import Input from '../components/Input'
import { usePredictionLoad } from '../components/PredictionLoad'
import { useAuth } from '../context/Auth'
import { errorMessage } from '../lib/api'
import './Login.css'

function Login() {
  const { authenticate, acceptUser } = useAuth()
  const { run, busy } = usePredictionLoad()
  const brandRef = useRef(null)
  const logoRef = useRef(null)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(event) {
    event.preventDefault()
    if (busy || submitting) return

    setSubmitting(true)
    setError('')

    try {
      const profile = await authenticate(email.trim(), password)
      await run({
        variant: 'login',
        mode: 'login',
        source: {
          brand: brandRef.current,
          logo: logoRef.current,
        },
        onReady: () => acceptUser(profile),
      })
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <section className={`login${busy ? ' is-loading' : ''}`}>
      <aside ref={brandRef} className="login__brand">
        <div className="login__grainient" aria-hidden="true">
          <Grainient
            color1="#1f03ff"
            color2="#0e2697"
            color3="#597dfd"
            timeSpeed={1.25}
            colorBalance={0}
            warpStrength={1}
            warpFrequency={5}
            warpSpeed={2}
            warpAmplitude={50}
            blendAngle={0}
            blendSoftness={0.05}
            rotationAmount={500}
            noiseScale={2}
            grainAmount={0.1}
            grainScale={2}
            grainAnimated={false}
            contrast={1.5}
            gamma={1}
            saturation={1}
            centerX={0}
            centerY={0}
            zoom={0.9}
          />
        </div>
        <img ref={logoRef} className="login__logo" src={logo} alt="motiva" />

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
            required
            autoComplete="username"
            onChange={(event) => setEmail(event.target.value)}
          />
          <Input
            id="password"
            label="Senha"
            type="password"
            value={password}
            required
            autoComplete="current-password"
            onChange={(event) => setPassword(event.target.value)}
          />

          {error ? <p className="login__error">{error}</p> : null}

          <Button type="submit">{submitting || busy ? 'Entrando…' : 'Logar'}</Button>
        </form>
      </div>
    </section>
  )
}

export default Login
