import Card from '../components/Card'
import { currentUser, dashboardSummary, latestPredictions, stationLabel } from '../data/mock'
import './Dashboard.css'

function Dashboard({ onNavigate }) {
  const maxRisk = Math.max(...dashboardSummary.critical_stations.map((item) => item.doors_at_risk), 1)
  const topStation = dashboardSummary.critical_stations[0]
  const lastRun = latestPredictions[0]?.last_evaluated_at
  const prioritized = latestPredictions.filter((item) => item.will_fail)

  return (
    <section className="dashboard">
      <div className="page-head">
        <div>
          <p className="dashboard__kicker">Linha 4 Amarela</p>
          <h1>Dashboard</h1>
          <p className="dashboard__lead">
            Olá, {currentUser.name.split(' ')[0]}. {dashboardSummary.doors_at_risk} portas em
            risco na última rodada da IA
            {lastRun ? `, em ${lastRun}` : ''}.
          </p>
        </div>
      </div>

      <div className="dashboard__hero">
        <Card
          label="Portas em risco nos próximos 7 dias"
          value={dashboardSummary.doors_at_risk}
          hint={`de ${dashboardSummary.total_doors} portas ativas na linha`}
        />
        <div className="dashboard__tiles">
          <button type="button" onClick={() => onNavigate?.('inspecoes')}>
            <span>Inspeções pendentes</span>
            <strong>{dashboardSummary.pending_inspections}</strong>
          </button>
          <button type="button" onClick={() => onNavigate?.('inspecoes')}>
            <span>Inspeções concluídas</span>
            <strong>{dashboardSummary.completed_inspections}</strong>
          </button>
          <article>
            <span>Estação mais crítica</span>
            <strong>{topStation.station_code}</strong>
            <em>
              {stationLabel(topStation.station_code)} · {topStation.doors_at_risk} portas em risco
            </em>
          </article>
        </div>
      </div>

      <div className="dashboard__split">
        <section className="dashboard__panel">
          <h2>Estações críticas</h2>
          {dashboardSummary.critical_stations.map((station) => (
            <div key={station.station_code} className="dashboard__station">
              <strong>{station.station_code}</strong>
              <div>
                <span>{stationLabel(station.station_code)}</span>
                <i style={{ width: `${(station.doors_at_risk / maxRisk) * 100}%` }} />
              </div>
              <em>{station.doors_at_risk}</em>
            </div>
          ))}
        </section>

        <section className="dashboard__panel">
          <h2>Portas priorizadas pela IA</h2>
          {prioritized.map((item) => (
            <div key={item.door_id} className="dashboard__prediction">
              <div>
                <span>TAG</span>
                <strong>{item.sap_tag}</strong>
              </div>
              <div>
                <span>Estação</span>
                <strong>{item.station_name}</strong>
              </div>
              <div>
                <span>Plataforma</span>
                <strong>{item.platform}</strong>
              </div>
              <div>
                <span>Diagnóstico</span>
                <strong>{item.will_fail ? 'Vai falhar' : 'Estável'}</strong>
              </div>
            </div>
          ))}
        </section>
      </div>
    </section>
  )
}

export default Dashboard
