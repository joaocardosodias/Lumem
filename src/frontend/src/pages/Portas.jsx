import { useMemo, useState } from 'react'
import Card from '../components/Card'
import CadastrarButton from '../components/CadastrarButton'
import {
  dashboardSummary,
  getStationDoors,
  lineStations,
} from '../data/mock'
import './Portas.css'

function padDoor(value) {
  return String(value).padStart(2, '0')
}

function Portas() {
  const [selectedCode, setSelectedCode] = useState(null)
  const [openedCode, setOpenedCode] = useState(null)
  const [doors, setDoors] = useState([])
  const [lastTraining, setLastTraining] = useState('08/09/2026')
  const [lastPrediction, setLastPrediction] = useState(null)

  const openedStation = lineStations.find((item) => item.code === openedCode)
  const selectedStation = lineStations.find((item) => item.code === selectedCode)
  const topStation = dashboardSummary.critical_stations[0]
  const doorsAtRisk = doors.filter((item) => item.will_fail).length
  const selectedIndex = lineStations.findIndex((item) => item.code === selectedCode)

  const maxRisk = useMemo(
    () => Math.max(...doors.map((item) => item.risk), 0.01),
    [doors],
  )

  function handleSelectStation(code) {
    setSelectedCode((current) => (current === code ? null : code))
  }

  function handleOpenStation(code) {
    const next = getStationDoors(code)
    setDoors(next)
    setOpenedCode(code)
    setSelectedCode(null)
    setLastPrediction(next[0]?.last_evaluated_at ?? '2026-09-08 18:40:00')
  }

  function handlePredict() {
    const stamp = new Date().toISOString().slice(0, 19).replace('T', ' ')
    setDoors((current) =>
      current.map((item) => ({
        ...item,
        last_evaluated_at: stamp,
      })),
    )
    setLastPrediction(stamp)
  }

  function handleRetrain() {
    const now = new Date()
    const stamp = `${String(now.getDate()).padStart(2, '0')}/${String(now.getMonth() + 1).padStart(2, '0')}/${now.getFullYear()}`
    setLastTraining(stamp)
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

        <div className="portas__kpis">
          <Card layout="banner" label="Total de portas" value="48" hint="24 por plataforma" />
          <Card layout="banner" label="Portas em risco" value={doorsAtRisk} hint="próximos 7 dias" />
        </div>

        <article className="portas__panel">
          <h2>Riscos por porta</h2>
          <div className="portas-risk-chart">
            {doors.map((item) => (
              <div key={item.door_id} className="portas-risk-chart__col">
                <span>
                  <i style={{ height: `${(item.risk / maxRisk) * 100}%` }} />
                </span>
                <strong>{padDoor(item.door_number)}</strong>
              </div>
            ))}
          </div>
        </article>

        <article className="portas__panel">
          <h2>Portas de Estação</h2>
          <div className="portas__table">
            <div className="portas__row portas__row--head">
              <span>Porta</span>
              <span>Risco</span>
              <span>Alarmes</span>
              <span>Última Manutenção</span>
              <span>Status</span>
            </div>
            {doors.map((item) => (
              <div key={item.door_id} className="portas__row">
                <strong>{padDoor(item.door_number)}</strong>
                <div className="portas-risk">
                  <b>
                    <i style={{ width: `${Math.round(item.risk * 100)}%` }} />
                  </b>
                  <em>{Math.round(item.risk * 100)}%</em>
                </div>
                <strong>{item.alarms}</strong>
                <strong>{item.last_maintenance_days} dias</strong>
                <span className={`portas-status${item.will_fail ? '' : ' is-ok'}`}>
                  {item.will_fail ? 'Em risco' : 'Estável'}
                </span>
              </div>
            ))}
          </div>
        </article>

        <div className="portas__actions">
          <CadastrarButton onClick={handlePredict}>Realizar Previsão</CadastrarButton>
          <div>
            <CadastrarButton onClick={handleRetrain}>Re-treinar modelo</CadastrarButton>
            <small>Último treinamento {lastTraining}</small>
          </div>
        </div>
        {lastPrediction ? <p className="portas__eval">Última predição {lastPrediction}</p> : null}
      </section>
    )
  }

  return (
    <section className="portas" onClick={() => setSelectedCode(null)}>
      <div className="page-head">
        <div>
          <h1>Portas</h1>
          <p>Linha 4 - Amarela</p>
        </div>
      </div>

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
            className={`portas-tooltip${selectedIndex % 2 ? ' is-up' : ' is-down'}${selectedIndex === 0 ? ' is-start' : ''}${selectedIndex === lineStations.length - 1 ? ' is-end' : ''}`}
            style={{ left: `${(selectedIndex / (lineStations.length - 1)) * 100}%` }}
          >
            <div>
              <div>
                <span>Estação</span>
                <strong>{selectedStation.name}</strong>
              </div>
              <i aria-hidden="true">{selectedStation.code}</i>
            </div>
            <button type="button" onClick={() => handleOpenStation(selectedStation.code)}>
              Acessar
            </button>
          </aside>
        ) : null}
      </div>

      <div className="portas__cards">
        <article>
          <span>Estação com maior probabilidade de defeito</span>
          <strong>{topStation.station_name}</strong>
        </article>
        <article>
          <span>Número de alarmes total da Linha</span>
          <strong>{dashboardSummary.alerts_last_7d}</strong>
        </article>
        <article>
          <span>Portas em risco nos próximos 7 dias</span>
          <strong>{dashboardSummary.doors_at_risk}</strong>
        </article>
      </div>
    </section>
  )
}

export default Portas
