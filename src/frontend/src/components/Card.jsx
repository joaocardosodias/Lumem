import { useEffect, useRef } from 'react'
import gsap from 'gsap'
import iconMotiva from '../assets/icon_motiva.svg'
import './Card.css'

function CountValue({ value }) {
  const nodeRef = useRef(null)
  const fromRef = useRef(0)
  const numeric = typeof value === 'number' ? value : Number(value)
  const isNumeric = Number.isFinite(numeric)

  useEffect(() => {
    if (!isNumeric || !nodeRef.current) return

    const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    if (reduce) {
      nodeRef.current.textContent = Math.round(numeric).toLocaleString('pt-BR')
      fromRef.current = numeric
      return
    }

    const from = fromRef.current
    const proxy = { n: from }
    const duration = Math.min(1.4, 0.55 + Math.log10(Math.max(Math.abs(numeric - from), 1)) * 0.32)
    const tween = gsap.to(proxy, {
      n: numeric,
      duration,
      ease: 'power2.out',
      onUpdate: () => {
        if (nodeRef.current) {
          nodeRef.current.textContent = Math.round(proxy.n).toLocaleString('pt-BR')
        }
      },
    })
    let scaleTween
    if (from === 0 && numeric > 0 && nodeRef.current) {
      scaleTween = gsap.fromTo(
        nodeRef.current,
        { scale: 0.86 },
        { scale: 1, duration: duration * 0.9, ease: 'power2.out' },
      )
    }
    fromRef.current = numeric
    return () => {
      tween.kill()
      scaleTween?.kill()
    }
  }, [isNumeric, numeric])

  if (!isNumeric) return value ?? '—'
  return <span ref={nodeRef}>{Math.round(fromRef.current).toLocaleString('pt-BR')}</span>
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
              <i />
            </div>
          </>
        ) : (
          <>
            <p>{label}</p>
            <div className="kpi-card__value">
              <strong>
                <CountValue value={value} />
              </strong>
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
