import { useEffect, useState } from 'react'
import Card from '../components/Card'
import { useAuth } from '../context/Auth'
import { api, asList, errorMessage, mergeDashboard } from '../lib/api'
import { stationLabel } from '../data/stations'
import './Dashboard.css'

const emptySummary = {
  total_doors: 0,
  doors_at_risk: 0,
  pending_inspections: 0,
  completed_inspections: 0,
  critical_stations: [],
}

function Dashboard({ onNavigate }) {
  const { user } = useAuth()
  const [summary, setSummary] = useState(emptySummary)
  const [inspectionKpis, setInspectionKpis] = useState({ pending: 0, completed: 0 })
  const [predictions, setPredictions] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false

    async function load() {
      setLoading(true)
      setError('')
      try {
        const [dashboardResult, latestResult] = await Promise.allSettled([
          api.dashboard(),
          api.predictionsLatest(),
        ])
        if (cancelled) return

        if (dashboardResult.status === 'fulfilled') {
          const merged = mergeDashboard(dashboardResult.value, null)
          setSummary(merged)
          setInspectionKpis({
            pending: merged.pending_inspections,
            completed: merged.completed_inspections,
          })
        } else {
          setError(errorMessage(dashboardResult.reason))
        }

        if (latestResult.status === 'fulfilled') {
          setPredictions(asList(latestResult.value))
        }
      } catch (err) {
        if (!cancelled) setError(errorMessage(err))
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    load()
    return () => {
      cancelled = true
    }
  }, [])

  const maxRisk = Math.max(...summary.critical_stations.map((item) => item.doors_at_risk), 1)
  const topStation = summary.critical_stations[0]
  const lastRun = predictions[0]?.last_evaluated_at
  const prioritized = predictions.filter((item) => item.will_fail)
  const firstName = user?.name?.split(' ')[0] || 'Operador'

  return (
    <section className="dashboard">
      <div className="page-head">
        <div>
          <p className="dashboard__kicker">Linha 4 Amarela</p>
          <h1>Dashboard</h1>
          <p className="dashboard__lead">
            Olá, {firstName}. {summary.doors_at_risk} portas em risco na última rodada da IA
            {lastRun ? `, em ${lastRun}` : ''}.
          </p>
        </div>
      </div>

      {loading ? <p className="page-status">Carregando o resumo operacional…</p> : null}
      {error ? <p className="page-status is-error">{error}</p> : null}

      <div className="dashboard__hero">
        <button
          type="button"
          onClick={() => onNavigate?.('predicoes', { status: 'fail' })}
          style={{ padding: 0, border: 0, background: 'transparent', textAlign: 'left', cursor: 'pointer' }}
        >
          <Card
            label="Portas em risco nos próximos 7 dias"
            value={summary.doors_at_risk}
            hint={`de ${summary.total_doors} portas ativas na linha · clique para ver predições`}
          />
        </button>
        <div className="dashboard__tiles">
          <button type="button" onClick={() => onNavigate?.('inspecoes', { status: 'pending' })}>
            <span>Inspeções pendentes</span>
            <strong>{inspectionKpis.pending}</strong>
          </button>
          <button type="button" onClick={() => onNavigate?.('inspecoes', { status: 'completed' })}>
            <span>Inspeções concluídas</span>
            <strong>{inspectionKpis.completed}</strong>
          </button>
          <article>
            <span>Estação mais crítica</span>
            <strong>{topStation?.station_code || '—'}</strong>
            <em>
              {topStation
                ? `${stationLabel(topStation.station_code)} · ${topStation.doors_at_risk} portas em risco`
                : 'Sem estações críticas nesta rodada'}
            </em>
          </article>
        </div>
      </div>

      <div className="dashboard__split">
        <section className="dashboard__panel">
          <h2>Estações críticas</h2>
          {summary.critical_stations.length ? (
            summary.critical_stations.map((station) => (
              <div
                key={station.station_code}
                className="dashboard__station"
                onClick={() => onNavigate?.('predicoes', { station: station.station_code, status: 'fail' })}
                style={{ cursor: 'pointer' }}
              >
                <strong>{station.station_code}</strong>
                <div>
                  <span>{stationLabel(station.station_code)}</span>
                  <i style={{ width: `${(station.doors_at_risk / maxRisk) * 100}%` }} />
                </div>
                <em>{station.doors_at_risk}</em>
              </div>
            ))
          ) : (
            <p className="page-empty">Nenhuma estação crítica nesta rodada.</p>
          )}
        </section>

        <section className="dashboard__panel">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '18px' }}>
            <h2 style={{ margin: 0 }}>Portas priorizadas pela IA</h2>
            <button
              type="button"
              onClick={() => onNavigate?.('predicoes')}
              style={{
                border: 0,
                background: 'transparent',
                color: 'var(--brand)',
                fontWeight: 600,
                fontSize: '13px',
                cursor: 'pointer',
              }}
            >
              Ver todas as predições →
            </button>
          </div>
          {prioritized.length ? (
            prioritized.map((item) => (
              <div
                key={item.door_id || item.sap_tag}
                className="dashboard__prediction"
                onClick={() =>
                  onNavigate?.('portas', {
                    station: item.station_code,
                    door: item.door_number,
                  })
                }
                style={{ cursor: 'pointer' }}
              >
                <div>
                  <span>TAG</span>
                  <strong>{item.sap_tag || '—'}</strong>
                </div>
                <div>
                  <span>Estação</span>
                  <strong>{item.station_name || stationLabel(item.station_code)}</strong>
                </div>
                <div>
                  <span>Plataforma</span>
                  <strong>{item.platform || '—'}</strong>
                </div>
                <div>
                  <span>Diagnóstico</span>
                  <strong style={{ color: item.will_fail ? '#dc2626' : '#16a34a' }}>
                    {item.will_fail ? 'Risco de Falha' : 'Estável'}
                  </strong>
                </div>
              </div>
            ))
          ) : (
            <p className="page-empty">Nenhuma porta priorizada na última inferência.</p>
          )}
        </section>
      </div>
    </section>
  )
}

export default Dashboard
