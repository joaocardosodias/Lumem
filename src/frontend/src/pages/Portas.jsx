import { useEffect, useMemo, useRef, useState } from 'react'
import Button from '../components/Button'
import { CountValue } from '../components/Card'
import StationPlan from '../components/StationPlan'
import PredictionHistory from '../components/PredictionHistory'
import { usePredictionLoad } from '../components/PredictionLoad'
import { useAuth } from '../context/Auth'
import { useToast } from '../context/Toast'
import { lineStations, mapStation, stationLabel } from '../data/stations'
import { api, asList, errorMessage, mergeDashboard, relatedMaintenances } from '../lib/api'
import { getCachedDashboard, setCachedDashboard, invalidateDashboard } from '../lib/dashboardCache'
import {
  getCachedDoorDossier,
  getCachedStationDoors,
  getLastOpenedStation,
  invalidateDoorsCache,
  loadStationDoors as loadCachedStationDoors,
  setCachedDoorDossier,
  setCachedStationDoors,
  setLastOpenedStation,
} from '../lib/doorsCache'
import { invalidateInspections } from '../lib/inspectionCache'
import './Portas.css'

function padDoor(value) {
  return String(value).padStart(2, '0')
}

function matchesDoorFilter(door, filter, search) {
  if (filter === 'risk' && !door.will_fail) return false
  if (filter === 'ok' && door.will_fail) return false
  const query = search.toLowerCase().trim()
  if (!query) return true
  const matchesNumber = String(door.door_number).includes(query) || padDoor(door.door_number).includes(query)
  return /^\d+$/.test(query) ? matchesNumber : String(door.sap_tag || '').toLowerCase().includes(query)
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
  const initialStation = params.station || getLastOpenedStation() || null
  const initialDoors = initialStation ? getCachedStationDoors(initialStation) : null
  const [selectedCode, setSelectedCode] = useState(null)
  const [tooltipOpen, setTooltipOpen] = useState(false)
  const [openedCode, setOpenedCode] = useState(initialStation)
  const [doors, setDoors] = useState(() => initialDoors || [])
  const [summary, setSummary] = useState(() => getCachedDashboard()?.summary || emptySummary)
  const [lastPrediction, setLastPrediction] = useState(() => initialDoors?.[0]?.last_evaluated_at || null)
  const [activePlatform, setActivePlatform] = useState(() => initialDoors?.[0]?.platform || 'P1')
  const [openedDoor, setOpenedDoor] = useState(null)
  const [doorMaintenances, setDoorMaintenances] = useState([])
  const [doorAlarms, setDoorAlarms] = useState([])
  const [doorAlarmsError, setDoorAlarmsError] = useState('')
  const [doorPredictions, setDoorPredictions] = useState([])
  const [doorPredictionsLoading, setDoorPredictionsLoading] = useState(false)
  const [doorPredictionsError, setDoorPredictionsError] = useState('')
  const [doorSearch, setDoorSearch] = useState('')
  const [doorFilter, setDoorFilter] = useState('all')
  const [inspNotes, setInspNotes] = useState('')
  const [inspError, setInspError] = useState('')
  const [error, setError] = useState('')
  const [loadingDoors, setLoadingDoors] = useState(false)
  const { user } = useAuth()
  const role = String(user?.role || '').toLowerCase()
  const canRunPrediction = role === 'enginner'
  const { run, busy } = usePredictionLoad()
  const hideTimer = useRef(null)
  const pendingCode = useRef(null)
  const doorDialogRef = useRef(null)
  const predictionHistoryRequestRef = useRef(0)
  const doorDetailsOpen = Boolean(openedDoor)

  const openedStation = lineStations.find((item) => item.code === openedCode)
  const selectedStation = lineStations.find((item) => item.code === selectedCode)
  const topStation = summary.critical_stations[0]
  const doorsAtRisk = doors.filter((item) => item.will_fail).length
  const recordedDoorProbability = Number(doorPredictions[0]?.probability)
  const hasRecordedDoorProbability = doorPredictions[0]?.probability != null && Number.isFinite(recordedDoorProbability)
  const platforms = useMemo(() => {
    return [...new Set([...(openedStation?.platforms ?? []), ...doors.map((item) => item.platform)])]
      .sort((a, b) => a.localeCompare(b, 'pt-BR', { numeric: true }))
  }, [doors, openedStation])
  const visibleDoors = useMemo(
    () =>
      doors
        .filter((item) => item.platform === activePlatform)
        .sort((a, b) => a.door_number - b.door_number),
    [doors, activePlatform],
  )
  const filteredDoors = useMemo(() => {
    return visibleDoors.filter((item) => matchesDoorFilter(item, doorFilter, doorSearch))
  }, [visibleDoors, doorFilter, doorSearch])

  const selectedIndex = lineStations.findIndex((item) => item.code === selectedCode)

  useEffect(() => {
    if (!doorDetailsOpen) return
    const trigger = document.activeElement
    doorDialogRef.current?.querySelector('button')?.focus({ preventScroll: true })
    return () => {
      if (trigger?.isConnected) trigger.focus({ preventScroll: true })
    }
  }, [doorDetailsOpen])

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
        const merged = mergeDashboard(data, null)
        setSummary(merged)
        setCachedDashboard(merged)
      } catch (err) {
        if (!cancelled && !getCachedDashboard()?.summary) setError(errorMessage(err))
      }
    }

    loadSummary()
    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    if (initialStation && !doors.length) {
      loadStationDoors(initialStation)
    }
  }, [])

  useEffect(() => {
    const targetStation = params.station ? mapStation(params.station) : null
    if (targetStation && targetStation !== openedCode) {
      handleOpenStation(targetStation, false)
    } else if (!targetStation && openedCode && params.station !== undefined) {
      setOpenedCode(null)
      setOpenedDoor(null)
      setLastOpenedStation(null)
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

  async function loadStationDoors(code, { force = false } = {}) {
    const station = lineStations.find((item) => item.code === code)
    const cached = getCachedStationDoors(code)
    if (cached && cached.length) {
      setDoors(cached)
      setActivePlatform(cached[0]?.platform ?? station?.platforms?.[0] ?? 'P1')
      setLastPrediction(cached[0]?.last_evaluated_at || null)
      if (force) setLoadingDoors(true)
    } else {
      setLoadingDoors(true)
    }
    setError('')
    try {
      const next = await loadCachedStationDoors(code, station?.name, { force })
      setDoors(next)
      setCachedStationDoors(code, next)
      setActivePlatform(next[0]?.platform ?? station?.platforms?.[0] ?? 'P1')
      setLastPrediction(next[0]?.last_evaluated_at || null)
    } catch (err) {
      if (!cached) {
        setDoors([])
        setError(errorMessage(err))
      }
    } finally {
      setLoadingDoors(false)
    }
  }

  function handleOpenStation(code, updateUrl = true) {
    if (updateUrl) {
      onNavigate?.('portas', { station: code })
    }
    setLastOpenedStation(code)
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
      const config = await api.getMLConfig().catch(() => ({ default_horizon_days: 7 }))
      await run({
        variant: 'overlay',
        mode: 'predict',
        task: () => api.triggerPredictions(config.default_horizon_days || 7),
      })
      invalidateDoorsCache()
      invalidateDashboard()
      toast.success('Diagnóstico preditivo executado com sucesso.')
      await loadStationDoors(openedCode)
    } catch (err) {
      setError(errorMessage(err))
      toast.error(errorMessage(err))
    }
  }

  async function handleOpenDoor(door, updateUrl = true) {
    setActivePlatform(door.platform)
    if (updateUrl && openedCode) {
      onNavigate?.('portas', { station: openedCode, platform: door.platform, door: door.door_number })
    }
    const cachedDossier = getCachedDoorDossier(door.door_id)
    if (cachedDossier) {
      setOpenedDoor(cachedDossier.door)
      setDoorMaintenances(cachedDossier.maintenances)
      setDoorAlarms(cachedDossier.alarms)
    } else {
      setOpenedDoor(door)
      setDoorMaintenances([])
      setDoorAlarms([])
    }
    setInspNotes('')
    setInspError('')
    setDoorAlarmsError('')
    setDoorPredictions([])
    setDoorPredictionsError('')
    setDoorPredictionsLoading(true)
    const historyRequest = predictionHistoryRequestRef.current + 1
    predictionHistoryRequestRef.current = historyRequest

    api.doorPredictionHistory(door.door_id, 30)
      .then((payload) => {
        if (predictionHistoryRequestRef.current !== historyRequest) return
        const history = asList(payload?.history || payload)
        setDoorPredictions(history)
      })
      .catch((err) => {
        if (predictionHistoryRequestRef.current === historyRequest) setDoorPredictionsError(errorMessage(err))
      })
      .finally(() => {
        if (predictionHistoryRequestRef.current === historyRequest) setDoorPredictionsLoading(false)
      })

    let extraMaintenances = cachedDossier?.maintenances || []
    let extraAlarms = cachedDossier?.alarms || []
    let updatedDoor = cachedDossier?.door || door
    let timelineLoaded = false

    try {
      const dossier = await api.doorTimeline(door.door_id)
      timelineLoaded = true
      updatedDoor = { ...door, ...dossier, station_code: door.station_code }
      setOpenedDoor(updatedDoor)
      extraAlarms = asList(dossier?.recent_alarms)
      setDoorAlarms(extraAlarms)
      const extra = asList(dossier?.maintenances || dossier?.maintenance)
      if (extra.length) {
        extraMaintenances = extra
        setDoorMaintenances(extra)
      } else {
        const items = asList(await api.maintenance(100, 0))
        extraMaintenances = relatedMaintenances(items, door)
        setDoorMaintenances(extraMaintenances)
      }
    } catch (err) {
      if (!cachedDossier) {
        setError(errorMessage(err))
        try {
          const items = asList(await api.maintenance(100, 0))
          extraMaintenances = relatedMaintenances(items, door)
          setDoorMaintenances(extraMaintenances)
        } catch {
          setDoorMaintenances([])
        }
      }
    }

    if (!timelineLoaded) {
      try {
        const rawAlarms = await api.alarms({
          station_code: door.station_code || openedCode,
          platform: door.platform,
          door_number: door.door_number,
          limit: 20,
        })
        extraAlarms = asList(rawAlarms?.data || rawAlarms?.alarms || rawAlarms)
        setDoorAlarms(extraAlarms)
      } catch {
        setDoorAlarmsError(
          cachedDossier
            ? 'Não foi possível atualizar os alarmes. Exibindo os dados salvos.'
            : 'Não foi possível carregar os alarmes desta porta.',
        )
        if (!cachedDossier) setDoorAlarms([])
      }
    }

    setCachedDoorDossier(door.door_id, {
      door: updatedDoor,
      maintenances: extraMaintenances,
      alarms: extraAlarms,
    })
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
        notes: inspNotes.trim() || undefined,
      })
      invalidateInspections()
      toast.success(`Ordem de inspeção para a porta ${padDoor(openedDoor.door_number)} cadastrada com sucesso.`)
      onNavigate?.('inspecoes')
    } catch (err) {
      setInspError(errorMessage(err))
      toast.error(errorMessage(err))
    }
  }

  function handleBackFromDoor() {
    predictionHistoryRequestRef.current += 1
    setOpenedDoor(null)
    if (openedCode) {
      onNavigate?.('portas', { station: openedCode })
    }
  }

  function handleDoorDialogKeyDown(event) {
    if (event.key === 'Escape') {
      event.preventDefault()
      handleBackFromDoor()
      return
    }
    if (event.key !== 'Tab') return
    const controls = doorDialogRef.current?.querySelectorAll('button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), a[href]')
    if (!controls?.length) return
    const first = controls[0]
    const last = controls[controls.length - 1]
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault()
      last.focus()
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault()
      first.focus()
    }
  }

  function handleBackFromStation() {
    setOpenedCode(null)
    setOpenedDoor(null)
    setLastOpenedStation(null)
    onNavigate?.('portas', {})
  }

  const adjacentDoors = useMemo(() => {
    if (!openedDoor) return { prev: null, next: null }
    const currentPlatformDoors = doors
      .filter((d) => d.platform === openedDoor.platform)
      .sort((a, b) => a.door_number - b.door_number)
    const currentIndex = currentPlatformDoors.findIndex(
      (d) => d.door_id === openedDoor.door_id || d.door_number === openedDoor.door_number,
    )
    if (currentIndex === -1 || currentPlatformDoors.length <= 1) {
      return { prev: null, next: null }
    }
    const prev = currentPlatformDoors[(currentIndex - 1 + currentPlatformDoors.length) % currentPlatformDoors.length]
    const next = currentPlatformDoors[(currentIndex + 1) % currentPlatformDoors.length]
    return { prev, next }
  }, [openedDoor, doors])



  if (openedStation) {
    return (
      <section className="portas portas--station">
        <div className="page-head">
          <div className="portas__title">
            <button type="button" className="portas__back" onClick={handleBackFromStation} aria-label="Voltar para a Linha 4">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="m14.5 5.5-6.5 6.5 6.5 6.5" />
              </svg>
            </button>
            <div>
              <p className="portas__eyebrow">Linha 4 Amarela · Estação {openedStation.code}</p>
              <h1>{openedStation.name}</h1>
            </div>
          </div>
          <div className="page-head__actions">
            <Button variant="secondary" disabled={loadingDoors || busy} onClick={() => loadStationDoors(openedCode, { force: true })}>
              Atualizar dados
            </Button>
            {canRunPrediction ? (
              <Button disabled={loadingDoors || busy} onClick={handlePredict}>
                Executar diagnóstico
              </Button>
            ) : null}
          </div>
        </div>

        {loadingDoors ? <p className="page-status">Carregando telemetria e topologia das portas…</p> : null}
        {error ? <p className="page-status is-error">{error}</p> : null}

        <div className="station-summary" aria-label="Resumo da estação">
          <article>
            <p>Portas monitoradas</p>
            <strong>{loadingDoors || (error && !doors.length) ? '—' : <CountValue value={doors.length} />}</strong>
            <span>Distribuídas em {platforms.length} plataformas</span>
          </article>
          <article>
            <p>Em risco de falha</p>
            <strong>{loadingDoors || (error && !doors.length) ? '—' : <CountValue value={doorsAtRisk} />}</strong>
            <span>Classificação da última análise</span>
          </article>
        </div>

        <StationPlan
          station={openedStation}
          platforms={platforms}
          doors={doors}
          loading={loadingDoors}
          error={error}
          filter={doorFilter}
          onFilter={setDoorFilter}
          search={doorSearch}
          onSearch={setDoorSearch}
          matchesDoor={(door) => matchesDoorFilter(door, doorFilter, doorSearch)}
          selectedDoor={openedDoor}
          onSelectDoor={handleOpenDoor}
        />

        <article className="portas__panel portas__records-panel" aria-labelledby="door-records-title">
          <header className="portas__doors-header portas__records-head">
            <div>
              <h2 id="door-records-title">Registros de portas</h2>
              <p className="portas__panel-sub">Identificação, diagnóstico e última avaliação por plataforma.</p>
            </div>
            <div className="portas__records-controls">
              <span className="portas__record-count">
                {filteredDoors.length} {filteredDoors.length === 1 ? 'porta' : 'portas'}
              </span>
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
            </div>
          </header>
          <div className="portas__table-head" aria-hidden="true">
            <span>Porta</span>
            <span>Identificação</span>
            <span>Diagnóstico</span>
            <span>Última avaliação</span>
            <span>Ação</span>
          </div>
          <div className="portas__records">
            {filteredDoors.length ? (
              filteredDoors.map((item) => (
                <button
                  key={item.door_id}
                  type="button"
                  className="portas__record"
                  onClick={() => handleOpenDoor(item)}
                >
                  <div>
                    <span className="portas__record-label">Porta</span>
                    <strong>Porta {padDoor(item.door_number)}</strong>
                  </div>
                  <div>
                    <span className="portas__record-label">Identificação</span>
                    <strong title={item.sap_tag}>{item.sap_tag || '—'}</strong>
                    <em>Plataforma {item.platform}</em>
                  </div>
                  <div>
                    <span className="portas__record-label">Diagnóstico</span>
                    <span className={`portas-status${item.will_fail ? '' : ' is-ok'}`}>
                      {item.will_fail ? 'Em risco' : 'Estável'}
                    </span>
                  </div>
                  <div>
                    <span className="portas__record-label">Última avaliação</span>
                    <strong>{item.last_evaluated_at || '—'}</strong>
                  </div>
                  <div className="portas__record-action">
                    <span className="portas__record-label">Ação</span>
                    <em>
                      Ver detalhes
                      <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true">
                        <path d="M3 8h10m-4-4 4 4-4 4" />
                      </svg>
                    </em>
                  </div>
                </button>
              ))
            ) : (
              <p className="portas__empty">Nenhuma porta encontrada com os filtros selecionados.</p>
            )}
          </div>
        </article>

        {lastPrediction ? <p className="portas__eval">Última avaliação de modelo: {lastPrediction}</p> : null}
        {openedDoor ? (
          <div className="portas-modal-backdrop" onClick={handleBackFromDoor}>
            <div
              className="portas-modal-content"
              ref={doorDialogRef}
              onClick={(e) => e.stopPropagation()}
              onKeyDown={handleDoorDialogKeyDown}
              role="dialog"
              aria-modal="true"
              aria-labelledby="dossier-door-title"
            >
              <header className="portas-modal-head">
                <div>
                  <div className="portas-modal-eyebrow">
                    <span>Detalhes do ativo</span>
                    <span className="portas-modal-car-tag">Carro {Math.ceil(openedDoor.door_number / 4)}</span>
                  </div>
                  <h2 id="dossier-door-title">
                    Porta {padDoor(openedDoor.door_number)} · Plataforma {openedDoor.platform}
                  </h2>
                  <p className="portas-modal-sub">
                    {openedDoor.sap_tag || `SP04-6-PSD-${openedDoor.station_code}-${openedDoor.platform}-PP${padDoor(openedDoor.door_number)}`} · {openedStation.name} ({openedStation.code})
                  </p>
                </div>

                <div className="portas-modal-nav-actions">
                  {adjacentDoors.prev ? (
                    <button
                      type="button"
                      className="btn-door-nav"
                      onClick={() => handleOpenDoor(adjacentDoors.prev)}
                      title="Porta Anterior"
                    >
                      ‹ P{padDoor(adjacentDoors.prev.door_number)}
                    </button>
                  ) : null}
                  {adjacentDoors.next ? (
                    <button
                      type="button"
                      className="btn-door-nav"
                      onClick={() => handleOpenDoor(adjacentDoors.next)}
                      title="Próxima Porta"
                    >
                      P{padDoor(adjacentDoors.next.door_number)} ›
                    </button>
                  ) : null}
                  <button
                    type="button"
                    className="portas-modal-close"
                    onClick={handleBackFromDoor}
                    aria-label="Fechar detalhes"
                  >
                    ✕
                  </button>
                </div>
              </header>

              <div className="portas-modal-body">
                <div className={`door-status-banner ${openedDoor.will_fail ? 'is-risk' : 'is-ok'}`}>
                  <div className="door-status-banner__info">
                    <span className="banner-kicker">Diagnóstico preditivo</span>
                    <h3>{openedDoor.will_fail ? 'Risco de falha identificado' : 'Operação estável'}</h3>
                    <p>
                      {openedDoor.will_fail
                        ? 'Os modelos preditivos identificaram anomalia de corrente ou tempo de ciclo de manobra nesta porta. Recomendada inspeção preventiva.'
                        : 'Telemetria dentro dos padrões nominais de vibração, corrente do atuador e tempo de ciclo de manobra.'}
                    </p>
                  </div>
                  <div className="door-status-banner__metric">
                    <span>{hasRecordedDoorProbability ? 'Probabilidade de falha' : 'Classificação do modelo'}</span>
                    <strong>
                      {hasRecordedDoorProbability
                        ? `${Math.round(recordedDoorProbability * 100)}%`
                        : doorPredictionsLoading
                          ? '…'
                          : openedDoor.will_fail ? 'Em risco' : 'Estável'}
                    </strong>
                    <small>Avaliação: {openedDoor.last_evaluated_at ? String(openedDoor.last_evaluated_at).slice(0, 10) : 'Recente'}</small>
                  </div>
                </div>

                <div className="door-specs-grid">
                  <div className="spec-item">
                    <span>Tag SAP</span>
                    <strong>{openedDoor.sap_tag || '—'}</strong>
                  </div>
                  <div className="spec-item">
                    <span>Estação</span>
                    <strong>{openedStation.name} ({openedDoor.station_code})</strong>
                  </div>
                  <div className="spec-item">
                    <span>Plataforma</span>
                    <strong>{openedDoor.platform}</strong>
                  </div>
                  <div className="spec-item">
                    <span>Posição no trem</span>
                    <strong>Porta {padDoor(openedDoor.door_number)} · Carro {Math.ceil(openedDoor.door_number / 4)}</strong>
                  </div>
                  <div className="spec-item">
                    <span>Status operacional</span>
                    <strong>{openedDoor.status === 'active' ? 'Ativa' : (openedDoor.status ?? 'Ativa')}</strong>
                  </div>
                  <div className="spec-item">
                    <span>ID do ativo</span>
                    <strong>#{openedDoor.door_id}</strong>
                  </div>
                </div>

                <div className="door-modal-section door-modal-section--predictions">
                  <div className="door-modal-section__head">
                    <div>
                      <h4>Histórico preditivo</h4>
                      <p>Evolução do risco e janelas previstas em todas as rodadas registradas.</p>
                    </div>
                    <span className="section-counter">{doorPredictionsLoading ? '…' : doorPredictions.length}</span>
                  </div>
                  <PredictionHistory
                    history={doorPredictions}
                    loading={doorPredictionsLoading}
                    error={doorPredictionsError}
                  />
                </div>

                <div className="door-modal-section">
                  <div className="door-modal-section__head">
                    <h4>Alarmes recentes</h4>
                    <span className="section-counter">
                      {doorAlarmsError && !doorAlarms.length ? '—' : doorAlarms.length}
                    </span>
                  </div>
                  {doorAlarmsError ? (
                    <p className="door-empty-state is-error">{doorAlarmsError}</p>
                  ) : null}
                  {doorAlarms.length ? (
                    <div className="door-alarms-list">
                      {doorAlarms.map((item) => (
                        <div key={item.id} className="door-alarm-row">
                          <div className="alarm-main">
                            <strong>{item.alarm_type}</strong>
                            <span>{item.event_timestamp}</span>
                          </div>
                          <div className="alarm-meta">
                            <em>Duração: {item.duration_seconds}s</em>
                            <code>TAG: {item.tag}</code>
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : !doorAlarmsError ? (
                    <p className="door-empty-state">Nenhum alarme de telemetria registrado para esta porta.</p>
                  ) : null}
                </div>

                <div className="door-modal-section">
                  <div className="door-modal-section__head">
                    <h4>Histórico de manutenções</h4>
                    <span className="section-counter">{doorMaintenances.length}</span>
                  </div>
                  {doorMaintenances.length ? (
                    <div className="door-maintenances-list">
                      {doorMaintenances.map((item) => (
                        <div key={item.id} className="door-maintenance-row">
                          <div className="maint-badge">{item.order_type}</div>
                          <div className="maint-content">
                            <strong>{item.description || item.symptom || 'Manutenção preventiva programada'}</strong>
                            <span>Início: {item.start_timestamp}</span>
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="door-empty-state">Nenhuma ordem de manutenção ligada a esta porta.</p>
                  )}
                </div>

                <div className="door-modal-section door-modal-section--inspection">
                  <div className="door-modal-section__head">
                    <div>
                      <h4>Abrir ordem de inspeção</h4>
                      <p>Registre uma observação para encaminhar a equipe de campo.</p>
                    </div>
                  </div>
                  {inspError ? <p className="page-status is-error">{inspError}</p> : null}
                  <form className="door-insp-form" onSubmit={handleCreateInspection}>
                    <input
                      type="text"
                      placeholder="Descreva a observação ou justificativa técnica da inspeção..."
                      value={inspNotes}
                      onChange={(e) => setInspNotes(e.target.value)}
                    />
                    <Button type="submit">Abrir ordem</Button>
                  </form>
                </div>
              </div>

              <footer className="portas-modal-foot">
                <Button
                  variant="secondary"
                  onClick={() => {
                    const currentDoor = openedDoor
                    handleBackFromDoor()
                    onNavigate?.('inspecoes', {
                      station: currentDoor.station_code,
                      platform: currentDoor.platform,
                      door: currentDoor.door_number,
                    })
                  }}
                >
                  Ver inspeções desta porta
                </Button>
                <Button onClick={handleBackFromDoor}>
                  Fechar
                </Button>
              </footer>
            </div>
          </div>
        ) : null}
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
