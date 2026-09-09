import { useMemo, useState } from 'react'
import Button from '../components/Button'
import CadastrarButton from '../components/CadastrarButton'
import Card from '../components/Card'
import CardAlerta from '../components/CardAlerta'
import Input from '../components/Input'
import PlanilhaImport from '../components/PlanilhaImport'
import UploadStatus from '../components/UploadStatus'
import { mapAlarmRows } from '../lib/importRows'
import { runUploadProgress } from '../lib/runUpload'
import { parseSpreadsheet } from '../lib/spreadsheet'
import { alarms as initialAlarms, createUpload } from '../data/mock'
import './Alertas.css'

const emptyForm = {
  tag: '',
  start_date: '',
  start_time: '',
  end_date: '',
  end_time: '',
  station: '',
  alarm: '',
}

function formatDateTime(date, time) {
  if (!date) return ''
  const [year, month, day] = date.split('-')
  const clock = time ? (time.length === 5 ? `${time}:00` : time) : '00:00:00'
  return `${day}/${month}/${year} ${clock}`
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
  const [mode, setMode] = useState('manual')
  const [form, setForm] = useState(emptyForm)
  const [file, setFile] = useState(null)
  const [importError, setImportError] = useState('')
  const [importing, setImporting] = useState(false)
  const [upload, setUpload] = useState(null)

  const byType = useMemo(() => countBy(items, 'alarm'), [items])
  const byStation = useMemo(() => countBy(items, 'station'), [items])
  const maxType = Math.max(...Object.values(byType), 1)
  const maxStation = Math.max(...Object.values(byStation), 1)
  const topType = Object.entries(byType).sort((a, b) => b[1] - a[1])[0]

  function updateField(field) {
    return (event) => setForm((current) => ({ ...current, [field]: event.target.value }))
  }

  function handleSubmit(event) {
    event.preventDefault()
    setItems((current) => [
      {
        tag: form.tag,
        started_at: formatDateTime(form.start_date, form.start_time),
        station: form.station,
        alarm: form.alarm,
        normalized_at: formatDateTime(form.end_date, form.end_time),
      },
      ...current,
    ])
    setForm(emptyForm)
    setOpen(false)
  }

  async function handleImport(event) {
    event.preventDefault()
    if (!file) {
      setImportError('Selecione um arquivo CSV ou XLSX.')
      return
    }

    setImporting(true)
    setImportError('')

    try {
      const rows = mapAlarmRows(await parseSpreadsheet(file))
      if (!rows.length) {
        setImportError('Nenhum alarme válido encontrado na planilha.')
        return
      }

      const job = createUpload({ fileName: file.name, fileType: 'alarms', rowCount: rows.length })
      await runUploadProgress(job, rows.length, setUpload)
      setItems((current) => [...rows, ...current])
      setFile(null)
      setUpload(null)
      setOpen(false)
    } catch (error) {
      setImportError(error.message || 'Não foi possível ler a planilha.')
    } finally {
      setImporting(false)
    }
  }

  function openCadastro() {
    setMode('manual')
    setImportError('')
    setFile(null)
    setUpload(null)
    setOpen(true)
  }

  return (
    <section className="alertas">
      <div className="page-head">
        <h1>Alertas</h1>
        <CadastrarButton onClick={openCadastro}>Cadastrar Alarmes +</CadastrarButton>
      </div>

      <div className="alertas__hero">
        <Card
          layout="banner"
          label="Alarmes registrados"
          value={items.length}
          hint={topType ? `${topType[1]} do tipo ${topType[0]}` : 'Nenhum alarme na linha'}
        />
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
        <div className="alertas-modal" onClick={() => !importing && setOpen(false)}>
          <form
            onClick={(event) => event.stopPropagation()}
            onSubmit={mode === 'import' ? handleImport : handleSubmit}
          >
            <h2>Cadastrar alarme</h2>
            <div className="cadastro-tabs">
              <button type="button" className={mode === 'manual' ? 'is-active' : ''} onClick={() => setMode('manual')}>
                Manual
              </button>
              <button type="button" className={mode === 'import' ? 'is-active' : ''} onClick={() => setMode('import')}>
                CSV / XLSX
              </button>
            </div>

            {mode === 'import' ? (
              <>
                <PlanilhaImport
                  file={file}
                  onFile={(next) => {
                    setFile(next)
                    setImportError('')
                    setUpload(null)
                  }}
                  hint="Planilha de alarmes de telemetria em CSV ou XLSX."
                  error={importError}
                />
                {upload ? <UploadStatus upload={upload} /> : null}
                <Button type="submit" variant="light">
                  {importing ? 'Processando upload…' : 'Enviar planilha'}
                </Button>
              </>
            ) : (
              <>
                <Input id="alarm-tag" label="TAG" value={form.tag} onChange={updateField('tag')} />
                <div className="ui-input-row">
                  <Input
                    id="alarm-start-date"
                    label="Data de início"
                    type="date"
                    value={form.start_date}
                    required
                    onChange={updateField('start_date')}
                  />
                  <Input
                    id="alarm-start-time"
                    label="Hora de início"
                    type="time"
                    step="60"
                    value={form.start_time}
                    required
                    onChange={updateField('start_time')}
                  />
                </div>
                <div className="ui-input-row">
                  <Input
                    id="alarm-end-date"
                    label="Data de normalização"
                    type="date"
                    value={form.end_date}
                    onChange={updateField('end_date')}
                  />
                  <Input
                    id="alarm-end-time"
                    label="Hora de normalização"
                    type="time"
                    step="60"
                    value={form.end_time}
                    onChange={updateField('end_time')}
                  />
                </div>
                <Input id="alarm-station" label="Estação" value={form.station} onChange={updateField('station')} />
                <Input id="alarm-type" label="Tipo de alarme" value={form.alarm} onChange={updateField('alarm')} />
                <Button type="submit" variant="light">
                  Cadastrar
                </Button>
              </>
            )}
          </form>
        </div>
      ) : null}
    </section>
  )
}

export default Alertas
