import Button from '../components/Button'
import CardAlerta from '../components/CardAlerta'
import { alarms } from '../data/mock'
import './Alertas.css'

function Alertas() {
  return (
    <section className="alertas">
      <div className="page-head">
        <h1>Alertas</h1>
        <Button>Cadastrar Alarmes +</Button>
      </div>

      <div className="alertas__list">
        {alarms.map((alarm) => (
          <CardAlerta
            key={`${alarm.tag}-${alarm.started_at}`}
            tag={alarm.tag}
            startedAt={alarm.started_at}
            station={alarm.station}
            alarm={alarm.alarm}
            normalizedAt={alarm.normalized_at}
          />
        ))}
      </div>
    </section>
  )
}

export default Alertas
