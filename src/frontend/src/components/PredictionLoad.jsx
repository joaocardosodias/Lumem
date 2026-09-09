import { createContext, useContext, useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import gsap from 'gsap'
import iconMotiva from '../assets/icon_motiva.svg'
import logo from '../assets/logo.svg'
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
    [78, 'Carregando o painel'],
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
      <span className="linha-train__glow" />
      <div className="linha-train__car">
        <svg viewBox="0 0 220 86" fill="none">
          <path d="M18 58h184c6 0 10-4 10-9V30c0-12-9-20-22-20H54L18 28v30Z" fill="#ffd000" />
          <path d="M18 38h194v6H18V38Z" fill="#3f00ff" />
          <rect x="58" y="22" width="28" height="16" rx="3" fill="#1a1464" opacity="0.85" />
          <rect x="92" y="22" width="28" height="16" rx="3" fill="#1a1464" opacity="0.85" />
          <rect x="126" y="22" width="28" height="16" rx="3" fill="#1a1464" opacity="0.85" />
          <rect x="160" y="22" width="28" height="16" rx="3" fill="#1a1464" opacity="0.85" />
          <path d="M18 28 54 10h8v28H18V28Z" fill="#fff59a" />
          <g className="linha-train__wheel" transform="translate(58 66)">
            <circle r="10" fill="#1a1464" />
            <circle r="4" fill="#ffd000" />
          </g>
          <g className="linha-train__wheel" transform="translate(96 66)">
            <circle r="10" fill="#1a1464" />
            <circle r="4" fill="#ffd000" />
          </g>
          <g className="linha-train__wheel" transform="translate(158 66)">
            <circle r="10" fill="#1a1464" />
            <circle r="4" fill="#ffd000" />
          </g>
          <g className="linha-train__wheel" transform="translate(196 66)">
            <circle r="10" fill="#1a1464" />
            <circle r="4" fill="#ffd000" />
          </g>
          <circle cx="34" cy="24" r="4" fill="#fff" />
        </svg>
      </div>
    </div>
  )
}

function PredictionLoadOverlay({ sessionRef, onSettled }) {
  const rootRef = useRef(null)
  const sceneRef = useRef(null)
  const logoRef = useRef(null)
  const marksRef = useRef(null)
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
    const markEl = source.mark
    const fromLogin = session.variant === 'login' && brand
    const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    const time = reduce ? 0.18 : 1
    const mode = session.mode ?? 'predict'

    document.body.style.overflow = 'hidden'

    const loginChrome = document.querySelectorAll('.login__panel, .login__intro')
    const logoFly = logoRef.current
    const markNodes = [...(marksRef.current?.children ?? [])]
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

      if (markEl && markNodes[0]) {
        const r = markEl.getBoundingClientRect()
        gsap.set(markNodes[0], {
          top: r.top,
          left: r.left,
          width: r.width,
          height: r.height,
          opacity: 0.12,
        })
        gsap.set(markEl, { autoAlpha: 0 })
      }

      markNodes.slice(1).forEach((node, index) => {
        gsap.set(node, {
          top: rect.top + rect.height * (0.2 + index * 0.18),
          left: rect.left + rect.width * (0.15 + index * 0.22),
          width: 64,
          height: 54,
          autoAlpha: 0,
          scale: 0.55,
        })
      })
    } else {
      gsap.set(root, { top: 0, left: 0, width: vw, height: vh, autoAlpha: 0 })
      gsap.set(logoFly, { top: 28, left: 32, width: 150, autoAlpha: 0 })
      const corners = [
        { top: -48, left: -36 },
        { top: -36, left: vw - 40 },
        { top: vh - 80, left: -40 },
        { top: vh - 64, left: vw - 48 },
      ]
      markNodes.forEach((node, index) => {
        const spot = corners[index] ?? corners[0]
        gsap.set(node, { top: spot.top, left: spot.left, width: 80, height: 68, autoAlpha: 0 })
      })
    }

    const tl = gsap.timeline({
      onComplete: () => {
        document.body.style.overflow = ''
        settledRef.current?.()
      },
    })

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
        top: cy - 168,
        left: cx - 84,
        width: 168,
        autoAlpha: 1,
        duration: 0.95 * time,
        ease: 'power3.inOut',
      },
      fromLogin ? 0.28 : 0.05,
    )

    markNodes.forEach((node, index) => {
      const spread = (index - (markNodes.length - 1) / 2) * 62
      tl.to(
        node,
        {
          top: cy - 92,
          left: cx + spread - 28,
          width: 56,
          height: 46,
          autoAlpha: 0.95,
          opacity: 0.95,
          scale: 1,
          duration: 0.9 * time,
          ease: 'power3.inOut',
        },
        (fromLogin ? 0.34 : 0.08) + index * 0.05,
      )
    })

    tl.to(scene, { autoAlpha: 1, y: 0, duration: 0.5 * time, ease: 'power2.out' }, '-=0.4')

    tl.to(
      progressProxy,
      {
        value: 100,
        duration: (mode === 'login' ? 3.5 : 3) * time,
        ease: 'power1.inOut',
        onUpdate: () => applyProgress(progressProxy.value),
      },
      '-=0.15',
    )

    tl.add(() => {
      session.onReady?.()
    })

    tl.to(root, { autoAlpha: 0, duration: 0.55 * time, ease: 'power2.inOut' }, '+=0.16')

    return () => {
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

      <div ref={marksRef} className="prediction-load__marks">
        {Array.from({ length: 4 }, (_, index) => (
          <img key={index} src={iconMotiva} alt="" />
        ))}
      </div>

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
    return new Promise((resolve) => {
      sessionRef.current = {
        variant: options.variant ?? 'overlay',
        mode: options.mode ?? 'predict',
        source: options.source ?? null,
        onReady: options.onReady,
        resolve,
      }
      setOpen(true)
    })
  }

  function handleSettled() {
    const session = sessionRef.current
    sessionRef.current = null
    setOpen(false)
    session?.resolve()
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
