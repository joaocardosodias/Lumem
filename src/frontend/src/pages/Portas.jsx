import { useEffect, useMemo, useRef, useState } from 'react'
import Card from '../components/Card'
import CadastrarButton from '../components/CadastrarButton'
import Input from '../components/Input'
import { usePredictionLoad } from '../components/PredictionLoad'
import { lineStations, stationLabel } from '../data/stations'
import { api, asList, countInspections, errorMessage, mergeDashboard, normalizeDoors, relatedMaintenances } from '../lib/api'
import { invalidateInspections, loadInspections } from '../lib/inspectionCache'
import './Portas.css'

function padDoor(value) {
  return String(value).padStart(2, '0')
}

const emptySummary = {
  total_doors: 0,
  doors_at_risk: 0,
  pending_inspections: 0,
  completed_inspections: 0,
  critical_stations: [],
}

function Portas({ onNavigate }) {
  const [selectedCode, setSelectedCode] = useState(null)
  const [tooltipOpen, setTooltipOpen] = useState(false)
  const [openedCode, setOpenedCode] = useState(null)
  const [doors, setDoors] = useState([])
  const [summary, setSummary] = useState(emptySummary)
  const [lastTraining, setLastTraining] = useState('—')
  const [lastPrediction, setLastPrediction] = useState(null)
  const [activePlatform, setActivePlatform] = useState('P1')
  const [openedDoor, setOpenedDoor] = useState(null)
  const [doorMaintenances, setDoorMaintenances] = useState([])
  const [inspNotes, setInspNotes] = useState('')
  const [inspWhen, setInspWhen] = useState({ date: '', time: '' })
  const [inspError, setInspError] = useState('')
  const [error, setError] = useState('')
  const [loadingDoors, setLoadingDoors] = useState(false)
  const { run, busy } = usePredictionLoad()
  const hideTimer = useRef(null)
  const pendingCode = useRef(null)

  const timeline = openedDoor
  const openedStation = lineStations.find((item) => item.code === openedCode)
  const selectedStation = lineStations.find((item) => item.code === selectedCode)
  const topStation = summary.critical_stations[0]
  const doorsAtRisk = doors.filter((item) => item.will_fail).length
  const platforms = useMemo(() => {
    const fromDoors = [...new Set(doors.map((item) => item.platform))]
    if (fromDoors.length) return fromDoors
    return openedStation?.platforms ?? ['P1', 'P2']
  }, [doors, openedStation])
  const visibleDoors = useMemo(
    () =>
      doors
        .filter((item) => item.platform === activePlatform)
        .sort((a, b) => a.door_number - b.door_number),
    [doors, activePlatform],
  )
  const selectedIndex = lineStations.findIndex((item) => item.code === selectedCode)
  const maxRisk = useMemo(
    () => Math.max(...visibleDoors.map((item) => item.risk), 0.01),
    [visibleDoors],
  )

  useEffect(() => {
    return () => {
      if (hideTimer.current) clearTimeout(hideTimer.current)
    }
  }, [])

  useEffect(() => {
    let cancelled = false

    async function loadSummary() {
      try {
        const [data, items] = await Promise.all([api.dashboard(), loadInspections()])
        if (cancelled) return
        setSummary(mergeDashboard(data, countInspections(items)))
      } catch (err) {
        if (!cancelled) setError(errorMessage(err))
      }
    }

    loadSummary()
    return () => {
      cancelled = true
    }
  }, [])

  function clearHide() {
    if (hideTimer.current) {
      clearTimeout(hideTimer.current)
      hideTimer.current = null
    }
  }

  function openTooltip(code) {
    setSelectedCode(code)
    requestAnimationFrame(() => {
      requestAnimationFrame(() => setTooltipOpen(true))
    })
  }

  function closeTooltip() {
    pendingCode.current = null
    setTooltipOpen(false)
    clearHide()
    hideTimer.current = setTimeout(() => {
      setSelectedCode(null)
      hideTimer.current = null
    }, 220)
  }

  function handleSelectStation(code) {
    if (selectedCode === code && tooltipOpen) {
      closeTooltip()
      return
    }

    const switching = Boolean(selectedCode && selectedCode !== code && (tooltipOpen || hideTimer.current))

    if (switching) {
      pendingCode.current = code
      setTooltipOpen(false)
      clearHide()
      hideTimer.current = setTimeout(() => {
        const next = pendingCode.current
        pendingCode.current = null
        hideTimer.current = null
        if (next) openTooltip(next)
        else setSelectedCode(null)
      }, 220)
      return
    }

    pendingCode.current = null
    clearHide()
    openTooltip(code)
  }

  async function loadStationDoors(code) {
    const station = lineStations.find((item) => item.code === code)
    setLoadingDoors(true)
    setError('')
    try {
      const next = normalizeDoors(await api.stationDoors(code), code, station?.name)
      setDoors(next)
      setActivePlatform(next[0]?.platform ?? station?.platforms?.[0] ?? 'P1')
      setLastPrediction(next[0]?.last_evaluated_at || null)
    } catch (err) {
      setDoors([])
      setError(errorMessage(err))
    } finally {
      setLoadingDoors(false)
    }
  }

  function handleOpenStation(code) {
    setOpenedCode(code)
    setOpenedDoor(null)
    setSelectedCode(null)
    setTooltipOpen(false)
    loadStationDoors(code)
  }

  async function handlePredict() {
    if (busy || !openedCode) return
    setError('')
    try {
      await run({
        variant: 'overlay',
        mode: 'predict',
        task: () => api.triggerPredictions('7'),
      })
      await loadStationDoors(openedCode)
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  async function handleRetrain() {
    if (busy) return
    setError('')
    try {
      await run({
        variant: 'overlay',
        mode: 'retrain',
        task: () => api.triggerPredictions('7'),
      })
      const now = new Date()
      setLastTraining(
        `${String(now.getDate()).padStart(2, '0')}/${String(now.getMonth() + 1).padStart(2, '0')}/${now.getFullYear()}`,
      )
      if (openedCode) await loadStationDoors(openedCode)
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  async function handleOpenDoor(door) {
    setOpenedDoor(door)
    setDoorMaintenances([])
    setInspNotes('')
    setInspWhen({ date: '', time: '' })
    setInspError('')

    try {
      const dossier = await api.doorTimeline(door.door_id)
      setOpenedDoor({ ...door, ...dossier, station_code: door.station_code })
      const extra = asList(dossier?.maintenances || dossier?.maintenance)
      if (extra.length) {
        setDoorMaintenances(extra)
        return
      }
    } catch (err) {
      setError(errorMessage(err))
    }

    try {
      const items = asList(await api.maintenance(100, 0))
      setDoorMaintenances(relatedMaintenances(items, door))
    } catch {
      setDoorMaintenances([])
    }
  }

  async function handleCreateInspection(event) {
    event.preventDefault()
    if (!openedDoor) return
    setInspError('')
    try {
      await api.createInspection({
        station_code: openedDoor.station_code,
        platform: openedDoor.platform,
        door_number: String(openedDoor.door_number),
        priority: openedDoor.will_fail ? 'high' : 'normal',
        scheduled_for:
          inspWhen.date && inspWhen.time
            ? `${inspWhen.date} ${inspWhen.time.length === 5 ? `${inspWhen.time}:00` : inspWhen.time}`
            : undefined,
        notes: inspNotes.trim() || undefined,
      })
      invalidateInspections()
      onNavigate?.('inspecoes')
    } catch (err) {
      setInspError(errorMessage(err))
    }
  }

  if (openedStation && timeline) {
    return (
      <section className="portas">
        <div className="page-head">
          <div className="portas__title">
            <button type="button" className="portas__back" onClick={() => setOpenedDoor(null)} aria-label="Voltar">
              ‹
            </button>
            <div>
              <h1>{timeline.sap_tag || `Porta ${padDoor(timeline.door_number)}`}</h1>
              <p>
                {timeline.station_name} · {timeline.platform} · porta {padDoor(timeline.door_number)}
              </p>
            </div>
          </div>
        </div>

        {error ? <p className="page-status is-error">{error}</p> : null}

        <div className="portas__kpis">
          <Card
            layout="banner"
            label="Diagnóstico"
            value={timeline.will_fail ? 'Vai falhar' : 'Estável'}
            hint={`status ${timeline.status} · id ${timeline.door_id}`}
          />
          <Card
            layout="banner"
            label="Última avaliação"
            value={timeline.last_evaluated_at ? String(timeline.last_evaluated_at).slice(0, 10) : '—'}
            hint={timeline.last_evaluated_at || 'Sem avaliação gravada'}
          />
        </div>

        <article className="portas__panel">
          <h2>Dossiê da porta</h2>
          <div className="portas__dossier">
            <div>
              <span>Id</span>
              <strong>{timeline.door_id}</strong>
            </div>
            <div>
              <span>Estação</span>
              <strong>{timeline.station_name}</strong>
            </div>
            <div>
              <span>Plataforma</span>
              <strong>{timeline.platform}</strong>
            </div>
            <div>
              <span>Porta</span>
              <strong>{padDoor(timeline.door_number)}</strong>
            </div>
            <div>
              <span>TAG</span>
              <strong>{timeline.sap_tag || '—'}</strong>
            </div>
            <div>
              <span>Status</span>
              <strong>{timeline.status}</strong>
            </div>
            <div>
              <span>Vai falhar</span>
              <strong>{timeline.will_fail ? 'Sim' : 'Não'}</strong>
            </div>
          </div>
        </article>

        <article className="portas__panel">
          <h2>Manutenções relacionadas</h2>
          {doorMaintenances.length ? (
            doorMaintenances.map((item) => (
              <div key={item.id} className="portas__related">
                <strong>{item.order_type}</strong>
                <span>{item.start_timestamp}</span>
                <em>{item.description || item.symptom || item.installation_location || item.station_code}</em>
              </div>
            ))
          ) : (
            <p className="portas__empty">Nenhuma ordem de manutenção ligada a esta porta.</p>
          )}
        </article>

        <article className="portas__panel">
          <h2>Abrir inspeção</h2>
          {inspError ? <p className="page-status is-error">{inspError}</p> : null}
          <form className="portas__insp" onSubmit={handleCreateInspection}>
            <Input
              id="door-insp-date"
              label="Data"
              type="date"
              value={inspWhen.date}
              onChange={(event) => setInspWhen((current) => ({ ...current, date: event.target.value }))}
            />
            <Input
              id="door-insp-time"
              label="Hora"
              type="time"
              step="60"
              value={inspWhen.time}
              onChange={(event) => setInspWhen((current) => ({ ...current, time: event.target.value }))}
            />
            <Input
              id="door-insp-notes"
              label="Notas"
              value={inspNotes}
              onChange={(event) => setInspNotes(event.target.value)}
            />
            <CadastrarButton type="submit">Abrir ordem</CadastrarButton>
          </form>
        </article>
      </section>
    )
  }

  if (openedStation) {
    return (
      <section className="portas">
        <div className="page-head">
          <div className="portas__title">
            <button type="button" className="portas__back" onClick={() => setOpenedCode(null)} aria-label="Voltar">
              ‹
            </button>
            <div>
              <h1>{openedStation.name}</h1>
              <p>Portas</p>
            </div>
          </div>
        </div>

        {loadingDoors ? <p className="page-status">Carregando o mapa térmico da estação…</p> : null}
        {error ? <p className="page-status is-error">{error}</p> : null}

        <div className="portas__kpis">
          <Card
            layout="banner"
            label="Total de portas"
            value={doors.length}
            hint={`${visibleDoors.length} em ${activePlatform} · ${platforms.length} plataformas`}
          />
          <Card layout="banner" label="Portas em risco" value={doorsAtRisk} hint="próximos 7 dias" />
        </div>

        <div className="portas-tabs" role="tablist" aria-label="Plataformas da estação">
          {platforms.map((platform) => (
            <button
              key={platform}
              type="button"
              role="tab"
              aria-selected={activePlatform === platform}
              className={activePlatform === platform ? 'is-active' : ''}
              onClick={() => setActivePlatform(platform)}
            >
              {platform}
            </button>
          ))}
        </div>

        <article className="portas__panel">
          <h2>Riscos por porta · {activePlatform}</h2>
          {visibleDoors.length ? (
            <div className="portas-risk-chart">
              {visibleDoors.map((item) => (
                <button
                  key={item.door_id}
                  type="button"
                  className="portas-risk-chart__col"
                  onClick={() => handleOpenDoor(item)}
                >
                  <span>
                    <i style={{ height: `${(item.risk / maxRisk) * 100}%` }} />
                  </span>
                  <strong>{padDoor(item.door_number)}</strong>
                </button>
              ))}
            </div>
          ) : (
            <p className="portas__empty">Nenhuma porta retornada para esta plataforma.</p>
          )}
        </article>

        <article className="portas__panel">
          <h2>Portas de Estação · {activePlatform}</h2>
          <div className="portas__table">
            <div className="portas__row portas__row--head">
              <span>Porta</span>
              <span>TAG</span>
              <span>Risco</span>
              <span>Status</span>
              <span>Avaliado em</span>
            </div>
            {visibleDoors.map((item) => (
              <button key={item.door_id} type="button" className="portas__row" onClick={() => handleOpenDoor(item)}>
                <strong>{padDoor(item.door_number)}</strong>
                <strong>{item.sap_tag || '—'}</strong>
                <div className="portas-risk">
                  <b>
                    <i style={{ width: `${Math.round(item.risk * 100)}%` }} />
                  </b>
                  <em>{Math.round(item.risk * 100)}%</em>
                </div>
                <span className={`portas-status${item.will_fail ? '' : ' is-ok'}`}>
                  {item.will_fail ? 'Em risco' : 'Estável'}
                </span>
                <strong>{item.last_evaluated_at || '—'}</strong>
              </button>
            ))}
          </div>
        </article>

        <div className="portas__actions">
          <CadastrarButton onClick={handlePredict}>Realizar Previsão</CadastrarButton>
          <div>
            <CadastrarButton onClick={handleRetrain}>Re-treinar modelo</CadastrarButton>
            <small>Último disparo {lastTraining}</small>
          </div>
        </div>
        {lastPrediction ? <p className="portas__eval">Última predição {lastPrediction}</p> : null}
      </section>
    )
  }

  return (
    <section className="portas" onClick={closeTooltip}>
      <div className="page-head">
        <div>
          <h1>Portas</h1>
          <p>Linha 4 - Amarela</p>
        </div>
      </div>

      {error ? <p className="page-status is-error">{error}</p> : null}

      <div className="linha-map" onClick={(event) => event.stopPropagation()}>
        <div className="linha-map__rail" />
        <div className="linha-map__stops">
          {lineStations.map((station, index) => (
            <button
              key={station.code}
              type="button"
              className={`linha-map__stop${index % 2 ? ' is-up' : ' is-down'}${selectedCode === station.code ? ' is-active' : ''}`}
              onClick={() => handleSelectStation(station.code)}
            >
              <span className="linha-map__dot" />
              <span className="linha-map__name">{station.name}</span>
            </button>
          ))}
        </div>

        {selectedStation ? (
          <aside
            className={`portas-tooltip${tooltipOpen ? ' is-open' : ''}${selectedIndex % 2 ? ' is-up' : ' is-down'}${selectedIndex === 0 ? ' is-start' : ''}${selectedIndex === lineStations.length - 1 ? ' is-end' : ''}`}
            style={{ left: `${(selectedIndex / (lineStations.length - 1)) * 100}%` }}
          >
            <div>
              <div>
                <span>Estação</span>
                <strong>{selectedStation.name}</strong>
              </div>
              <img src={selectedStation.photo} alt="" />
            </div>
            <button type="button" onClick={() => handleOpenStation(selectedStation.code)}>
              Acessar
            </button>
          </aside>
        ) : null}
      </div>

      <div className="portas__cards">
        <article>
          <span>Estação mais crítica</span>
          <strong>{topStation ? stationLabel(topStation.station_code) : '—'}</strong>
        </article>
        <article>
          <span>Inspeções pendentes</span>
          <strong>{summary.pending_inspections}</strong>
        </article>
        <article>
          <span>Portas em risco nos próximos 7 dias</span>
          <strong>{summary.doors_at_risk}</strong>
        </article>
      </div>
    </section>
  )
}

export default Portas
