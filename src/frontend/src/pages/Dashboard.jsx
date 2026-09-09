import { useEffect, useState } from 'react'
import Card from '../components/Card'
import { useAuth } from '../context/Auth'
import { api, asList, errorMessage, fetchInspectionCounts, mergeDashboard } from '../lib/api'
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
        const [dashboardResult, latestResult, inspectionsResult] = await Promise.allSettled([
          api.dashboard(),
          api.predictionsLatest(),
          fetchInspectionCounts(),
        ])
        if (cancelled) return

        if (dashboardResult.status === 'fulfilled') {
          setSummary(mergeDashboard(dashboardResult.value, null))
        } else {
          setError(errorMessage(dashboardResult.reason))
        }

        if (latestResult.status === 'fulfilled') {
          setPredictions(asList(latestResult.value))
        }

        if (inspectionsResult.status === 'fulfilled') {
          const counts = inspectionsResult.value
          setInspectionKpis({ pending: counts.pending, completed: counts.completed })
        } else if (dashboardResult.status === 'fulfilled') {
          setError(errorMessage(inspectionsResult.reason))
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
        <Card
          label="Portas em risco nos próximos 7 dias"
          value={summary.doors_at_risk}
          hint={`de ${summary.total_doors} portas ativas na linha`}
        />
        <div className="dashboard__tiles">
          <button type="button" onClick={() => onNavigate?.('inspecoes')}>
            <span>Inspeções pendentes</span>
            <strong>{inspectionKpis.pending}</strong>
          </button>
          <button type="button" onClick={() => onNavigate?.('inspecoes')}>
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
              <div key={station.station_code} className="dashboard__station">
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
          <h2>Portas priorizadas pela IA</h2>
          {prioritized.length ? (
            prioritized.map((item) => (
              <div key={item.door_id || item.sap_tag} className="dashboard__prediction">
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
                  <strong>{item.will_fail ? 'Vai falhar' : 'Estável'}</strong>
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
