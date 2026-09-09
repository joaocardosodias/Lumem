import Card from '../components/Card'
import { dashboardSummary } from '../data/mock'
import './Dashboard.css'

function Dashboard() {
  return (
    <section className="dashboard">
      <div className="page-head">
        <h1>Bem vindo ao Dashboard!</h1>
      </div>

      <div className="dashboard__cards">
        <Card
          label="Probabilidade de uma porta quebrar nos próximos 7 dias."
          value={`${Math.round(dashboardSummary.failure_probability_7d * 100)}%`}
        />
        <Card
          label="Número de alertas nos últimos 7 dias em todas as estações."
          value={dashboardSummary.alerts_last_7d}
        />
        <Card
          label="Estação com maior número de alarmes recentes."
          value={dashboardSummary.station_most_alarms}
        />
      </div>
    </section>
  )
}

export default Dashboard
