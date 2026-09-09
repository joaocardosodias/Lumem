import { useMemo, useState } from 'react'
import Button from '../components/Button'
import CadastrarButton from '../components/CadastrarButton'
import CardAlerta from '../components/CardAlerta'
import Input from '../components/Input'
import { alarms as initialAlarms } from '../data/mock'
import './Alertas.css'

const emptyForm = {
  tag: '',
  started_at: '',
  station: '',
  alarm: '',
  normalized_at: '',
}

function countBy(items, key) {
  return items.reduce((acc, item) => {
    const value = item[key]
    acc[value] = (acc[value] || 0) + 1
    return acc
  }, {})
}

function Alertas() {
  const [items, setItems] = useState(initialAlarms)
  const [open, setOpen] = useState(false)
  const [form, setForm] = useState(emptyForm)

  const byType = useMemo(() => countBy(items, 'alarm'), [items])
  const byStation = useMemo(() => countBy(items, 'station'), [items])
  const maxType = Math.max(...Object.values(byType), 1)
  const maxStation = Math.max(...Object.values(byStation), 1)

  function updateField(field) {
    return (event) => setForm((current) => ({ ...current, [field]: event.target.value }))
  }

  function handleSubmit(event) {
    event.preventDefault()
    setItems((current) => [form, ...current])
    setForm(emptyForm)
    setOpen(false)
  }

  return (
    <section className="alertas">
      <div className="page-head">
        <h1>Alertas</h1>
        <CadastrarButton onClick={() => setOpen(true)}>Cadastrar Alarmes +</CadastrarButton>
      </div>

      <div className="alertas__charts">
        <article className="alertas__chart">
          <h2>Alarmes por tipo</h2>
          {Object.entries(byType).map(([label, value]) => (
            <div key={label} className="alertas__bar">
              <strong>{label}</strong>
              <i style={{ width: `${(value / maxType) * 100}%` }} />
              <em>{value}</em>
            </div>
          ))}
        </article>
        <article className="alertas__chart">
          <h2>Alarmes por estação</h2>
          {Object.entries(byStation).map(([label, value]) => (
            <div key={label} className="alertas__bar">
              <strong>{label}</strong>
              <i style={{ width: `${(value / maxStation) * 100}%` }} />
              <em>{value}</em>
            </div>
          ))}
        </article>
      </div>

      <div className="alertas__list">
        {items.map((alarm) => (
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

      {open ? (
        <div className="alertas-modal" onClick={() => setOpen(false)}>
          <form
            onClick={(event) => event.stopPropagation()}
            onSubmit={handleSubmit}
          >
            <h2>Cadastrar alarme</h2>
            <Input id="alarm-tag" label="TAG" value={form.tag} onChange={updateField('tag')} />
            <Input id="alarm-start" label="Data de início" value={form.started_at} onChange={updateField('started_at')} />
            <Input id="alarm-end" label="Data de normalização" value={form.normalized_at} onChange={updateField('normalized_at')} />
            <Input id="alarm-station" label="Estação" value={form.station} onChange={updateField('station')} />
            <Input id="alarm-type" label="Tipo de alarme" value={form.alarm} onChange={updateField('alarm')} />
            <Button type="submit" variant="light">
              Cadastrar
            </Button>
          </form>
        </div>
      ) : null}
    </section>
  )
}

export default Alertas
