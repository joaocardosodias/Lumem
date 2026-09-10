import { useEffect, useMemo, useRef, useState } from 'react'
import Card from '../components/Card'
import CadastrarButton from '../components/CadastrarButton'
import Input from '../components/Input'
import { usePredictionLoad } from '../components/PredictionLoad'
import { useToast } from '../context/Toast'
import { lineStations, mapStation, stationLabel } from '../data/stations'
import { api, asList, errorMessage, mergeDashboard, normalizeDoors, relatedMaintenances } from '../lib/api'
import { invalidateInspections } from '../lib/inspectionCache'
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

function Portas({ onNavigate, params = {} }) {
  const toast = useToast()
  const [selectedCode, setSelectedCode] = useState(null)
  const [tooltipOpen, setTooltipOpen] = useState(false)
  const [openedCode, setOpenedCode] = useState(params.station || null)
  const [doors, setDoors] = useState([])
  const [summary, setSummary] = useState(emptySummary)
  const [lastPrediction, setLastPrediction] = useState(null)
  const [activePlatform, setActivePlatform] = useState('P1')
  const [openedDoor, setOpenedDoor] = useState(null)
  const [doorMaintenances, setDoorMaintenances] = useState([])
  const [doorAlarms, setDoorAlarms] = useState([])
  const [doorSearch, setDoorSearch] = useState('')
  const [doorFilter, setDoorFilter] = useState('all')
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
  const filteredDoors = useMemo(() => {
    return visibleDoors.filter((item) => {
      if (doorFilter === 'risk' && !item.will_fail) return false
      if (doorFilter === 'ok' && item.will_fail) return false
      if (doorSearch) {
        const query = doorSearch.toLowerCase().trim()
        const tag = String(item.sap_tag || '').toLowerCase()
        const num = String(item.door_number || '')
        if (!tag.includes(query) && !num.includes(query)) return false
      }
      return true
    })
  }, [visibleDoors, doorFilter, doorSearch])

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
        const data = await api.dashboard()
        if (cancelled) return
        setSummary(mergeDashboard(data, null))
      } catch (err) {
        if (!cancelled) setError(errorMessage(err))
      }
    }

    loadSummary()
    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    const targetStation = params.station ? mapStation(params.station) : null
    if (targetStation && targetStation !== openedCode) {
      handleOpenStation(targetStation, false)
    } else if (!targetStation && openedCode) {
      setOpenedCode(null)
      setOpenedDoor(null)
    }
  }, [params.station])

  useEffect(() => {
    if (params.door && doors.length && (!openedDoor || String(openedDoor.door_number) !== String(params.door) || (params.platform && openedDoor.platform !== params.platform))) {
      const match = doors.find(
        (d) => String(d.door_number) === String(params.door) && (!params.platform || d.platform === params.platform),
      )
      if (match) {
        if (match.platform) setActivePlatform(match.platform)
        handleOpenDoor(match, false)
      }
    }
  }, [params.door, params.platform, doors])

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

  function handleOpenStation(code, updateUrl = true) {
    if (updateUrl) {
      onNavigate?.('portas', { station: code })
    }
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
      toast.success('Diagnóstico preditivo executado com sucesso!')
      await loadStationDoors(openedCode)
    } catch (err) {
      setError(errorMessage(err))
      toast.error(errorMessage(err))
    }
  }

  async function handleOpenDoor(door, updateUrl = true) {
    if (updateUrl && openedCode) {
      onNavigate?.('portas', { station: openedCode, door: door.door_number })
    }
    setOpenedDoor(door)
    setDoorMaintenances([])
    setDoorAlarms([])
    setInspNotes('')
    setInspWhen({ date: '', time: '' })
    setInspError('')

    try {
      const dossier = await api.doorTimeline(door.door_id)
      setOpenedDoor({ ...door, ...dossier, station_code: door.station_code })
      const extra = asList(dossier?.maintenances || dossier?.maintenance)
      if (extra.length) {
        setDoorMaintenances(extra)
      } else {
        const items = asList(await api.maintenance(100, 0))
        setDoorMaintenances(relatedMaintenances(items, door))
      }
    } catch (err) {
      setError(errorMessage(err))
      try {
        const items = asList(await api.maintenance(100, 0))
        setDoorMaintenances(relatedMaintenances(items, door))
      } catch {
        setDoorMaintenances([])
      }
    }

    try {
      const rawAlarms = await api.alarms({
        station_code: door.station_code || openedCode,
        door_number: door.door_number,
        limit: 10,
      })
      const alarms = asList(rawAlarms?.data || rawAlarms?.alarms || rawAlarms)
      setDoorAlarms(alarms)
    } catch {
      setDoorAlarms([])
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
      toast.success(`Ordem de inspeção para a porta ${padDoor(openedDoor.door_number)} cadastrada com sucesso!`)
      onNavigate?.('inspecoes')
    } catch (err) {
      setInspError(errorMessage(err))
      toast.error(errorMessage(err))
    }
  }

  function handleBackFromDoor() {
    setOpenedDoor(null)
    if (openedCode) {
      onNavigate?.('portas', { station: openedCode })
    }
  }

  function handleBackFromStation() {
    setOpenedCode(null)
    setOpenedDoor(null)
    onNavigate?.('portas', {})
  }

  if (openedStation && timeline) {
    return (
      <section className="portas">
        <div className="page-head">
          <div className="portas__title">
            <button type="button" className="portas__back" onClick={handleBackFromDoor} aria-label="Voltar">
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
            label="Diagnóstico Preditivo"
            value={timeline.will_fail ? 'Em risco de falha' : 'Estável'}
            hint={`status ${timeline.status} · porta ${timeline.station_code}-${timeline.platform}-P${padDoor(timeline.door_number)}`}
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
              <span>Diagnóstico IA</span>
              <strong style={{ color: timeline.will_fail ? '#dc2626' : '#16a34a' }}>
                {timeline.will_fail ? 'Risco de Falha' : 'Operação Estável'}
              </strong>
            </div>
          </div>
          {timeline.will_fail ? (
            <div style={{ marginTop: 14, display: 'flex', gap: 10 }}>
              <button
                type="button"
                className="portas-link-insp"
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 8,
                  padding: '8px 16px',
                  borderRadius: 8,
                  background: '#7c3aed',
                  color: '#fff',
                  border: 'none',
                  fontWeight: 600,
                  fontSize: 13,
                  cursor: 'pointer',
                }}
                onClick={() =>
                  onNavigate?.('inspecoes', {
                    station: timeline.station_code,
                    door: timeline.door_number,
                  })
                }
              >
                Ver Ordens de Inspeção Desta Porta
              </button>
            </div>
          ) : null}
        </article>

        <article className="portas__panel">
          <h2>Alarmes de telemetria da porta</h2>
          {doorAlarms.length ? (
            doorAlarms.map((item) => (
              <div key={item.id} className="portas__related">
                <strong>{item.alarm_type}</strong>
                <span>{item.event_timestamp}</span>
                <em>Duração: {item.duration_seconds}s · TAG: {item.tag}</em>
              </div>
            ))
          ) : (
            <p className="portas__empty">Nenhum alarme de telemetria registrado para esta porta.</p>
          )}
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
            <button type="button" className="portas__back" onClick={handleBackFromStation} aria-label="Voltar">
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
          <div style={{ display: 'flex', gap: 12, alignItems: 'center', justifyContent: 'space-between', marginBottom: 16, flexWrap: 'wrap' }}>
            <h2>Portas de Estação · {activePlatform}</h2>
            <div style={{ display: 'flex', gap: 10, alignItems: 'center', flexWrap: 'wrap' }}>
              <input
                type="text"
                placeholder="Buscar porta ou TAG…"
                value={doorSearch}
                onChange={(e) => setDoorSearch(e.target.value)}
                style={{
                  padding: '8px 12px',
                  borderRadius: 8,
                  border: '1px solid rgba(63,0,255,0.2)',
                  fontSize: '0.86rem',
                  outline: 'none',
                }}
              />
              <select
                value={doorFilter}
                onChange={(e) => setDoorFilter(e.target.value)}
                style={{
                  padding: '8px 12px',
                  borderRadius: 8,
                  border: '1px solid rgba(63,0,255,0.2)',
                  fontSize: '0.86rem',
                  background: '#ffffff',
                }}
              >
                <option value="all">Todas as portas</option>
                <option value="risk">Em risco</option>
                <option value="ok">Estáveis</option>
              </select>
            </div>
          </div>
          <div className="portas__table">
            <div className="portas__row portas__row--head">
              <span>Porta</span>
              <span>TAG</span>
              <span>Risco</span>
              <span>Status</span>
              <span>Avaliado em</span>
            </div>
            {filteredDoors.length ? (
              filteredDoors.map((item) => (
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
              ))
            ) : (
              <p className="portas__empty">Nenhuma porta encontrada com esses filtros.</p>
            )}
          </div>
        </article>

        <div className="portas__actions">
          <div>
            <CadastrarButton onClick={handlePredict}>Executar Diagnóstico IA</CadastrarButton>
          </div>
          <div>
            <CadastrarButton onClick={() => loadStationDoors(openedCode)}>Atualizar Dados</CadastrarButton>
          </div>
        </div>
        {lastPrediction ? <p className="portas__eval">Última avaliação: {lastPrediction}</p> : null}
      </section>
    )
  }

  return (
    <section className="portas" onClick={closeTooltip}>
      <div className="page-head">
        <div>
          <p className="dashboard__kicker">Linha 4 Amarela</p>
          <h1>Portas de Plataforma</h1>
          <p className="dashboard__lead">
            Visão topográfica, status operacional e dossiê detalhado das portas por estação.
          </p>
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
