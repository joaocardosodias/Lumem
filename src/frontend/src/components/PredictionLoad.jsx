import { createContext, useContext, useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import gsap from 'gsap'
import logo from '../assets/logo.svg'
import metro from '../assets/metro.svg'
import Grainient from './Grainient'
import './PredictionLoad.css'

const PredictionLoadContext = createContext(null)

const STAGES = {
  predict: [
    [0, 'Disparando a inferência'],
    [28, 'Lendo o status das portas'],
    [58, 'Calculando risco para 7 dias'],
    [82, 'Gravando as predições'],
  ],
  retrain: [
    [0, 'Preparando o retreino'],
    [35, 'Ajustando o modelo'],
    [70, 'Validando a nova rodada'],
    [88, 'Publicando o modelo'],
  ],
  login: [
    [0, 'Autenticando o acesso'],
    [24, 'Disparando a inferência'],
    [52, 'Avaliando as portas da linha'],
    [78, 'Carregando as ordens de manutenção'],
  ],
}

function stageFor(mode, progress) {
  const steps = STAGES[mode] ?? STAGES.predict
  let label = steps[0][1]
  for (const [threshold, text] of steps) {
    if (progress >= threshold) label = text
  }
  return label
}

function LinhaTrain() {
  return (
    <div className="linha-train" aria-hidden="true">
      <span className="linha-train__rail">
        <i className="linha-train__progress" />
      </span>
      <div className="linha-train__car">
        <img src={metro} alt="" />
      </div>
    </div>
  )
}

function PredictionLoadOverlay({ sessionRef, onSettled }) {
  const rootRef = useRef(null)
  const sceneRef = useRef(null)
  const logoRef = useRef(null)
  const trainWrapRef = useRef(null)
  const stageRef = useRef(null)
  const percentRef = useRef(null)
  const settledRef = useRef(onSettled)
  settledRef.current = onSettled

  useLayoutEffect(() => {
    const root = rootRef.current
    const session = sessionRef.current
    if (!root || !session) return

    const source = session.source ?? {}
    const brand = source.brand
    const logoEl = source.logo
    const fromLogin = session.variant === 'login' && brand
    const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    const time = reduce ? 0.18 : 1
    const mode = session.mode ?? 'predict'

    document.body.style.overflow = 'hidden'

    const loginChrome = document.querySelectorAll('.login__panel, .login__intro')
    const logoFly = logoRef.current
    const scene = sceneRef.current
    const trainWrap = trainWrapRef.current
    const car = trainWrap?.querySelector('.linha-train__car')
    const railFill = trainWrap?.querySelector('.linha-train__progress')
    const progressProxy = { value: 0 }

    const applyProgress = (value) => {
      const next = Math.max(0, Math.min(100, Math.round(value)))
      if (percentRef.current) percentRef.current.textContent = `${next}%`
      if (stageRef.current) stageRef.current.textContent = stageFor(mode, next)
      if (car && trainWrap) {
        const travel = Math.max(0, trainWrap.offsetWidth - car.offsetWidth)
        gsap.set(car, { x: (next / 100) * travel })
      }
      if (railFill) gsap.set(railFill, { scaleX: next / 100 })
    }

    const vw = window.innerWidth
    const vh = window.innerHeight
    const cx = vw / 2
    const cy = vh / 2

    gsap.set(scene, { autoAlpha: 0, y: 24 })
    applyProgress(0)

    if (fromLogin) {
      const rect = brand.getBoundingClientRect()
      gsap.set(root, {
        top: rect.top,
        left: rect.left,
        width: rect.width,
        height: rect.height,
        autoAlpha: 1,
      })

      if (logoEl && logoFly) {
        const r = logoEl.getBoundingClientRect()
        gsap.set(logoFly, { top: r.top, left: r.left, width: r.width, autoAlpha: 1 })
        gsap.set(logoEl, { autoAlpha: 0 })
      }
    } else {
      gsap.set(root, { top: 0, left: 0, width: vw, height: vh, autoAlpha: 0 })
      gsap.set(logoFly, { top: 28, left: 32, width: 150, autoAlpha: 0 })
    }

    let cancelled = false
    const tl = gsap.timeline()

    if (fromLogin) {
      tl.to(loginChrome, { autoAlpha: 0, duration: 0.32 * time, ease: 'power2.out' }, 0)
      tl.to(
        root,
        {
          top: 0,
          left: 0,
          width: vw,
          height: vh,
          duration: 1.08 * time,
          ease: 'power3.inOut',
        },
        0,
      )
    } else {
      tl.to(root, { autoAlpha: 1, duration: 0.4 * time, ease: 'power2.out' }, 0)
    }

    tl.to(
      logoFly,
      {
        top: cy - 120,
        left: cx - 84,
        width: 168,
        autoAlpha: 1,
        duration: 0.95 * time,
        ease: 'power3.inOut',
      },
      fromLogin ? 0.28 : 0.05,
    )

    tl.to(scene, { autoAlpha: 1, y: 0, duration: 0.5 * time, ease: 'power2.out' }, '-=0.45')

    const finishOk = () => {
      session.onReady?.()
      gsap.to(root, {
        autoAlpha: 0,
        duration: 0.55 * time,
        ease: 'power2.inOut',
        onComplete: () => {
          document.body.style.overflow = ''
          settledRef.current?.()
        },
      })
    }

    const finishErr = (error) => {
      document.body.style.overflow = ''
      settledRef.current?.(error)
    }

    tl.add(() => {
      const runWork = async () => {
        try {
          if (session.task) {
            applyProgress(8)
            const crawl = gsap.to(progressProxy, {
              value: 72,
              duration: 14 * time,
              ease: 'none',
              onUpdate: () => applyProgress(progressProxy.value),
            })
            try {
              await Promise.resolve().then(() =>
                session.task((info = {}) => {
                  if (cancelled) return
                  const loaded = Number(info.loaded) || 0
                  const total = Number(info.total)
                  const pages = Number(info.pages) || 0
                  const ratio =
                    Number.isFinite(total) && total > 0
                      ? loaded / total
                      : 1 - Math.exp(-pages / 8)
                  const pct = Math.min(96, 10 + 86 * Math.min(1, ratio))
                  gsap.to(progressProxy, {
                    value: pct,
                    duration: 0.18,
                    overwrite: true,
                    ease: 'power1.out',
                    onUpdate: () => applyProgress(progressProxy.value),
                  })
                }),
              )
            } finally {
              crawl.kill()
            }
            await new Promise((resolve) => {
              gsap.to(progressProxy, {
                value: 100,
                duration: 0.35 * time,
                overwrite: true,
                ease: 'power2.out',
                onUpdate: () => applyProgress(progressProxy.value),
                onComplete: resolve,
              })
            })
          } else {
            await new Promise((resolve) => {
              gsap.to(progressProxy, {
                value: 100,
                duration: (mode === 'login' ? 3.5 : 3) * time,
                ease: 'power1.inOut',
                onUpdate: () => applyProgress(progressProxy.value),
                onComplete: resolve,
              })
            })
          }
          if (!cancelled) finishOk()
        } catch (error) {
          if (!cancelled) finishErr(error)
        }
      }

      runWork()
    })

    return () => {
      cancelled = true
      tl.kill()
      document.body.style.overflow = ''
    }
  }, [sessionRef])

  return createPortal(
    <div ref={rootRef} className="prediction-load" role="alert" aria-live="assertive" aria-busy="true">
      <div className="prediction-load__grainient">
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

      <img ref={logoRef} className="prediction-load__logo" src={logo} alt="motiva" />

      <div ref={sceneRef} className="prediction-load__scene">
        <div ref={trainWrapRef} className="prediction-load__train">
          <LinhaTrain />
        </div>
        <div className="prediction-load__copy">
          <span ref={stageRef}>Disparando a inferência</span>
          <strong ref={percentRef}>0%</strong>
        </div>
      </div>
    </div>,
    document.body,
  )
}

export function PredictionLoadProvider({ children }) {
  const [open, setOpen] = useState(false)
  const sessionRef = useRef(null)

  function run(options = {}) {
    return new Promise((resolve, reject) => {
      sessionRef.current = {
        variant: options.variant ?? 'overlay',
        mode: options.mode ?? 'predict',
        source: options.source ?? null,
        task: options.task,
        onReady: options.onReady,
        resolve,
        reject,
      }
      setOpen(true)
    })
  }

  function handleSettled(error) {
    const session = sessionRef.current
    sessionRef.current = null
    setOpen(false)
    if (error) session?.reject?.(error)
    else session?.resolve?.()
  }

  return (
    <PredictionLoadContext.Provider value={{ run, busy: open }}>
      {children}
      {open ? <PredictionLoadOverlay sessionRef={sessionRef} onSettled={handleSettled} /> : null}
    </PredictionLoadContext.Provider>
  )
}

export function usePredictionLoad() {
  const value = useContext(PredictionLoadContext)
  if (!value) {
    throw new Error('usePredictionLoad precisa do PredictionLoadProvider')
  }
  return value
}
