import { useEffect, useState } from 'react'
import { CountValue } from '../components/Card'
import { NavIcon } from '../components/BarraNav'
import iconMotiva from '../assets/icon_motiva.svg'
import { useAuth } from '../context/Auth'
import { api, asList, errorMessage, mergeDashboard } from '../lib/api'
import { stationLabel } from '../data/stations'
import { getCachedDashboard, setCachedDashboard } from '../lib/dashboardCache'
import { getCachedPrediction, setCachedPrediction } from '../lib/predictionCache'
import './Dashboard.css'

function ArrowIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M5 12h14m-5-5 5 5-5 5" />
    </svg>
  )
}

function MetricValue({ value, loading }) {
  if (loading) return <span className="overview-skeleton overview-skeleton--value" aria-label="Carregando" />
  return <strong className="overview-metric__value">{value == null ? '—' : <CountValue value={value} />}</strong>
}

function PanelState({ loading, error, children }) {
  if (loading) {
    return (
      <div className="overview-placeholder" role="status" aria-label="Carregando dados">
        <span className="overview-skeleton" /><span className="overview-skeleton" /><span className="overview-skeleton" />
      </div>
    )
  }
  return (
    <div className={`overview-empty${error ? ' overview-empty--error' : ''}`} role={error ? 'alert' : 'status'}>
      <span className="overview-empty__icon" aria-hidden="true"><NavIcon id={error ? 'alarmes' : 'inspecoes'} /></span>
      <p>{error || children}</p>
      {!error && <span>O resumo acompanha os dados da última análise.</span>}
    </div>
  )
}

function evaluationLabel(value) {
  if (!value) return 'Sem data de avaliação'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return 'Data de avaliação indisponível'
  return new Intl.DateTimeFormat('pt-BR', { dateStyle: 'short', timeStyle: 'short' }).format(date)
}

function Dashboard({ onNavigate }) {
  const { user } = useAuth()
  const cached = getCachedDashboard()
  const [summary, setSummary] = useState(() => cached?.summary || null)
  const [predictions, setPredictions] = useState(() => cached?.predictions || [])
  const [loading, setLoading] = useState(() => !cached?.summary)
  const [error, setError] = useState('')
  const [predictionsError, setPredictionsError] = useState('')

  useEffect(() => {
    let cancelled = false

    async function load() {
      if (!getCachedDashboard()?.summary) {
        setLoading(true)
      }
      try {
        const [dashboardResult, latestResult] = await Promise.allSettled([
          api.dashboard(),
          api.predictionsLatest(),
        ])
        if (cancelled) return

        let nextSummary = null
        let nextPredictions = []

        if (dashboardResult.status === 'fulfilled') {
          nextSummary = mergeDashboard(dashboardResult.value, null)
          setSummary(nextSummary)
        } else if (!cached?.summary) {
          setError(errorMessage(dashboardResult.reason))
        }

        if (latestResult.status === 'fulfilled') {
          nextPredictions = asList(latestResult.value)
          setPredictions(nextPredictions)

          if (!getCachedPrediction()?.items?.length) {
            const list = asList(latestResult.value)
            const nextRunData = {
              id: latestResult.value?.id || '',
              horizon_days: latestResult.value?.horizon_days || 7,
              total_doors_evaluated: latestResult.value?.total_doors_evaluated || 0,
              total_doors_at_risk: latestResult.value?.total_doors_at_risk || 0,
              created_at: latestResult.value?.created_at || '',
            }
            const nextItems = list.map((item) => ({
              door_id: item.door_id ?? item.id,
              run_id: item.run_id || '',
              station_code: item.station_code || '',
              station_name: stationLabel(item.station_code) || item.station_name || item.station_code || '',
              platform: item.platform || 'P1',
              door_number: item.door_number,
              sap_tag: item.sap_tag || `SP04-6-PSD-${item.station_code || 'BUT'}-${item.platform || 'P1'}-PP${String(item.door_number || '01').padStart(2, '0')}`,
              will_fail: Boolean(item.will_fail),
            }))
            setCachedPrediction(nextRunData, nextItems)
          }
        } else if (!cached?.predictions?.length) {
          setPredictionsError(errorMessage(latestResult.reason))
        }

        if (nextSummary) {
          setCachedDashboard(nextSummary, nextPredictions)
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    load()
    return () => { cancelled = true }
  }, [])

  const stations = [...(summary?.critical_stations || [])].sort((a, b) => b.doors_at_risk - a.doors_at_risk)
  const maxRisk = Math.max(...stations.map((item) => item.doors_at_risk), 1)
  const topStation = stations[0]
  const lastRun = predictions[0]?.last_evaluated_at
  const prioritized = predictions.filter((item) => item.will_fail)
  const firstName = user?.name?.split(' ')[0] || 'Operador'

  return (
    <section className="overview" aria-labelledby="overview-title" aria-busy={loading}>
      <header className="overview-head">
        <div>
          <p className="overview-eyebrow"><span aria-hidden="true" />Operação e manutenção</p>
          <h1 id="overview-title">Visão geral</h1>
          <p className="overview-lead">Olá, {firstName}. Estas são as prioridades da linha.</p>
        </div>
        <button className="overview-action" type="button" onClick={() => onNavigate?.('portas')}>
          Explorar portas <ArrowIcon />
        </button>
      </header>

      {error && <p className="overview-error" role="alert">Não foi possível carregar o resumo: {error}</p>}

      <div className="overview-metrics">
        <button className="overview-metric overview-metric--primary" type="button" onClick={() => onNavigate?.('predicoes', { status: 'fail' })}>
          <img className="overview-metric__watermark" src={iconMotiva} alt="" aria-hidden="true" />
          <span className="overview-metric__top"><span>Portas em risco</span><ArrowIcon /></span>
          <MetricValue value={summary?.doors_at_risk} loading={loading} />
          <span className="overview-metric__description">Horizonte de 7 dias</span>
          <span className="overview-metric__foot">
            <span>{loading ? 'Carregando…' : summary ? `${summary.total_doors.toLocaleString('pt-BR')} portas na linha` : 'Resumo indisponível'}</span>
            <ArrowIcon />
          </span>
        </button>

        <button className="overview-metric" type="button" onClick={() => onNavigate?.('inspecoes', { status: 'pending' })}>
          <span className="overview-metric__top"><span>Inspeções pendentes</span><span aria-hidden="true"><NavIcon id="inspecoes" /></span></span>
          <MetricValue value={summary?.pending_inspections} loading={loading} />
          <span className="overview-metric__description">Aguardando execução</span>
          <span className="overview-metric__foot">Ver pendentes <ArrowIcon /></span>
        </button>

        <button className="overview-metric" type="button" onClick={() => onNavigate?.('inspecoes', { status: 'completed' })}>
          <span className="overview-metric__top"><span>Inspeções concluídas</span><span aria-hidden="true"><NavIcon id="manutencoes" /></span></span>
          <MetricValue value={summary?.completed_inspections} loading={loading} />
          <span className="overview-metric__description">Atividades finalizadas</span>
          <span className="overview-metric__foot">Ver concluídas <ArrowIcon /></span>
        </button>

        <article className="overview-metric overview-metric--station">
          <span className="overview-metric__top"><span>Estação crítica</span><span aria-hidden="true"><NavIcon id="portas" /></span></span>
          <MetricValue value={topStation?.station_code} loading={loading} />
          <span className="overview-metric__description" title={topStation ? stationLabel(topStation.station_code) : undefined}>{loading ? 'Consultando…' : topStation ? stationLabel(topStation.station_code) : error ? 'Resumo indisponível' : 'Sem estação crítica'}</span>
          <span className="overview-metric__foot">{loading ? 'Aguarde a análise' : topStation ? `${topStation.doors_at_risk} portas em risco` : error ? 'Tente novamente mais tarde' : 'Última análise disponível'}</span>
        </article>
      </div>

      <div className="overview-panels">
        <section className="overview-panel" aria-labelledby="critical-stations-title">
          <header className="overview-panel__head">
            <div><p className="overview-eyebrow">Distribuição por estação</p><h2 id="critical-stations-title">Estações prioritárias</h2></div>
            <span className="overview-panel__icon" aria-hidden="true"><NavIcon id="alarmes" /></span>
          </header>
          <div className="overview-panel__legend"><span>Estação</span><span>Portas em risco</span></div>
          {loading || error || !stations.length ? (
            <PanelState loading={loading} error={error}>Nenhuma estação crítica nesta rodada.</PanelState>
          ) : (
            <div className="overview-stations">
              {stations.map((station) => (
                <button key={station.station_code} className="overview-station" type="button" title={`${stationLabel(station.station_code)} · ${station.doors_at_risk} portas em risco`} onClick={() => onNavigate?.('predicoes', { station: station.station_code, status: 'fail' })}>
                  <span className="overview-station__code">{station.station_code}</span>
                  <span className="overview-station__body">
                    <span>{stationLabel(station.station_code)}</span>
                    <span className="overview-station__track" aria-hidden="true"><span style={{ width: `${(station.doors_at_risk / maxRisk) * 100}%` }} /></span>
                  </span>
                  <strong>{station.doors_at_risk}</strong><ArrowIcon />
                </button>
              ))}
            </div>
          )}
          <footer className="overview-panel__footer">Ordenado pelo total de portas em risco.</footer>
        </section>

        <section className="overview-panel" aria-labelledby="priority-doors-title">
          <header className="overview-panel__head">
            <div><p className="overview-eyebrow">Última análise preditiva</p><h2 id="priority-doors-title">Portas em atenção</h2></div>
            <button className="overview-text-action" type="button" onClick={() => onNavigate?.('predicoes')}>Ver todas <ArrowIcon /></button>
          </header>
          {loading || predictionsError || !prioritized.length ? (
            <PanelState loading={loading} error={predictionsError}>Nenhuma porta priorizada nesta rodada.</PanelState>
          ) : (
            <div className="overview-predictions">
              {prioritized.slice(0, 5).map((item) => (
                <button
                  key={item.door_id || item.sap_tag}
                  className="overview-prediction"
                  type="button"
                  title={`${item.sap_tag || `Porta ${item.door_number ?? '—'}`} · ${item.station_name || stationLabel(item.station_code)} · ${item.platform || 'Plataforma não informada'}`}
                  onClick={() => onNavigate?.('portas', { station: item.station_code, door: item.door_number })}
                >
                  <span className="overview-prediction__icon" aria-hidden="true"><NavIcon id="portas" /></span>
                  <span className="overview-prediction__body">
                    <strong>{item.sap_tag || `Porta ${item.door_number ?? '—'}`}</strong>
                    <span>{item.station_name || stationLabel(item.station_code)} · {item.platform || 'Plataforma não informada'}</span>
                  </span>
                  <span className="overview-risk"><i aria-hidden="true" />Risco de falha</span>
                  <ArrowIcon />
                </button>
              ))}
            </div>
          )}
          <footer className="overview-panel__footer">
            <span>{loading ? 'Carregando…' : predictionsError ? 'Predições indisponíveis' : prioritized.length > 5 ? `5 de ${prioritized.length} portas priorizadas` : `${prioritized.length} portas priorizadas`}</span>
            <button className="overview-text-action" type="button" onClick={() => onNavigate?.('predicoes')}>Ver predições <ArrowIcon /></button>
          </footer>
        </section>
      </div>

      <footer className="overview-footer">
        <span>Linha 4-Amarela · Gestão de portas de plataforma</span>
        <span>{loading ? 'Consultando última avaliação…' : predictionsError ? 'Avaliação indisponível' : `Última avaliação · ${evaluationLabel(lastRun)}`}</span>
      </footer>
    </section>
  )
}

export default Dashboard
