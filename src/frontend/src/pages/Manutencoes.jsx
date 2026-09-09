import { useEffect, useMemo, useState } from 'react'
import Button from '../components/Button'
import CadastrarButton from '../components/CadastrarButton'
import Card from '../components/Card'
import CardManutencao from '../components/CardManutencao'
import Filtros, { FiltroCampo } from '../components/Filtros'
import Input from '../components/Input'
import Paginacao from '../components/Paginacao'
import PlanilhaImport from '../components/PlanilhaImport'
import UploadStatus from '../components/UploadStatus'
import { lineStations } from '../data/stations'
import { DEFAULT_PERIOD, PAGE_SIZE, inDurationRange, inPeriod, periodHint } from '../lib/filters'
import { api, errorMessage } from '../lib/api'
import { loadMaintenance, getCachedMaintenance } from '../lib/maintenanceCache'
import { pollUpload } from '../lib/runUpload'
import './Manutencoes.css'

const emptyForm = {
  order_type: '',
  sap_order_number: '',
  installation_location: '',
  station_code: '',
  start_date: '',
  start_time: '',
  end_date: '',
  end_time: '',
  duration_hours: '',
  symptom: '',
  description: '',
}

const emptyFilters = {
  station: '',
  orderType: '',
  source: '',
  period: DEFAULT_PERIOD,
  durationMin: '',
  durationMax: '',
}

function countBy(items, key) {
  return items.reduce((acc, item) => {
    const value = item[key]
    acc[value] = (acc[value] || 0) + 1
    return acc
  }, {})
}

function toTimestamp(date, time) {
  if (!date) return ''
  const clock = time ? (time.length === 5 ? `${time}:00` : time) : '00:00:00'
  return `${date} ${clock}`
}

function durationFrom(start, end) {
  if (!start || !end) return null
  const started = Date.parse(start.replace(' ', 'T'))
  const ended = Date.parse(end.replace(' ', 'T'))
  if (!Number.isFinite(started) || !Number.isFinite(ended) || ended <= started) return null
  return Math.round(((ended - started) / 36e5) * 100) / 100
}

function Manutencoes() {
  const [items, setItems] = useState(() => getCachedMaintenance() ?? [])
  const [loading, setLoading] = useState(() => !getCachedMaintenance())
  const [error, setError] = useState('')
  const [open, setOpen] = useState(false)
  const [mode, setMode] = useState('manual')
  const [form, setForm] = useState(emptyForm)
  const [file, setFile] = useState(null)
  const [orderType, setOrderType] = useState('ZM01')
  const [importError, setImportError] = useState('')
  const [importing, setImporting] = useState(false)
  const [upload, setUpload] = useState(null)
  const [filters, setFilters] = useState(emptyFilters)
  const [page, setPage] = useState(1)
  const [reloadAt, setReloadAt] = useState(0)

  useEffect(() => {
    let cancelled = false

    async function load() {
      setLoading(true)
      setError('')
      try {
        const batch = await loadMaintenance({ force: Boolean(reloadAt) })
        if (!cancelled) setItems(batch)
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
  }, [reloadAt])

  const visible = useMemo(
    () =>
      items.filter((item) => {
        if (
          filters.station &&
          String(item.station_code || '').toUpperCase() !== filters.station.toUpperCase()
        ) {
          return false
        }
        if (filters.orderType && item.order_type !== filters.orderType) return false
        if (filters.source && item.source !== filters.source) return false
        if (!inPeriod(item.start_timestamp, filters.period)) return false
        if (!inDurationRange(item.duration_hours, filters.durationMin, filters.durationMax)) return false
        return true
      }),
    [items, filters],
  )
  const filtersDirty =
    Boolean(filters.station || filters.orderType || filters.source || filters.durationMin || filters.durationMax) ||
    filters.period !== DEFAULT_PERIOD
  const pageCount = Math.max(1, Math.ceil(visible.length / PAGE_SIZE))
  const currentPage = Math.min(page, pageCount)
  const paged = visible.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE)
  const rangeHint = periodHint(filters.period)
  const orderTypes = useMemo(() => {
    const values = new Set(['ZM01', 'ZM02', 'ZM03'])
    items.forEach((item) => {
      if (item.order_type) values.add(item.order_type)
    })
    return [...values]
  }, [items])
  const sources = useMemo(() => {
    const values = [...new Set(items.map((item) => item.source).filter(Boolean))]
    return values.length ? values : ['manual', 'upload']
  }, [items])
  const byType = useMemo(() => countBy(visible, 'order_type'), [visible])
  const byStation = useMemo(() => countBy(visible, 'station_code'), [visible])
  const maxType = Math.max(...Object.values(byType), 1)
  const maxStation = Math.max(...Object.values(byStation), 1)
  const topType = Object.entries(byType).sort((a, b) => b[1] - a[1])[0]

  function updateField(field) {
    return (event) => {
      const value = event.target.value
      setForm((current) => {
        const next = { ...current, [field]: value }
        if (['start_date', 'start_time', 'end_date', 'end_time'].includes(field)) {
          const duration = durationFrom(
            toTimestamp(next.start_date, next.start_time),
            toTimestamp(next.end_date, next.end_time),
          )
          if (duration != null) next.duration_hours = String(duration)
        }
        return next
      })
    }
  }

  async function handleSubmit(event) {
    event.preventDefault()

    const start_timestamp = toTimestamp(form.start_date, form.start_time)
    const end_timestamp = toTimestamp(form.end_date, form.end_time)
    const computed = durationFrom(start_timestamp, end_timestamp)
    const duration = form.duration_hours === '' ? computed : Number(form.duration_hours)

    setImportError('')
    try {
      await api.createMaintenance({
        order_type: form.order_type.trim().toUpperCase(),
        sap_order_number: form.sap_order_number.trim() || undefined,
        installation_location: form.installation_location.trim() || undefined,
        station_code: form.station_code.trim().toUpperCase(),
        start_timestamp,
        end_timestamp: end_timestamp || undefined,
        duration_hours: Number.isFinite(duration) ? duration : undefined,
        description: form.description.trim() || undefined,
        symptom: form.symptom.trim() || undefined,
      })
      setForm(emptyForm)
      setOpen(false)
      setPage(1)
      setReloadAt(Date.now())
    } catch (err) {
      setImportError(errorMessage(err))
    }
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
      const result = await api.uploadMaintenance(file, orderType)
      const done = await pollUpload(result, setUpload)
      if (done.status === 2) {
        throw new Error('A API recusou o processamento deste arquivo.')
      }
      setFile(null)
      setUpload(null)
      setOpen(false)
      setPage(1)
      setReloadAt(Date.now())
    } catch (err) {
      setImportError(errorMessage(err))
    } finally {
      setImporting(false)
    }
  }

  function updateFilter(field) {
    return (event) => {
      setPage(1)
      setFilters((current) => ({ ...current, [field]: event.target.value }))
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
    <section className="manutencoes">
      <div className="page-head">
        <h1>Manutenções</h1>
        <CadastrarButton onClick={openCadastro}>Cadastrar Manutenções +</CadastrarButton>
      </div>

      {loading ? <p className="page-status">Carregando ordens de manutenção…</p> : null}
      {error ? <p className="page-status is-error">{error}</p> : null}

      {loading ? null : (
      <>
      <div className="manutencoes__hero">
        <Card
          layout="banner"
          label="Ordens de manutenção"
          value={visible.length}
          hint={
            topType
              ? `${topType[1]} do tipo ${topType[0]}${rangeHint ? ` · ${rangeHint}` : ''}`
              : 'Nenhuma ordem na linha'
          }
        />
        <div className="manutencoes__charts">
          <article className="manutencoes__chart">
            <h2>Ordens por tipo</h2>
            {Object.entries(byType).map(([label, value]) => (
              <div key={label} className="manutencoes__bar">
                <strong>{label}</strong>
                <i style={{ width: `${(value / maxType) * 100}%` }} />
                <em>{value}</em>
              </div>
            ))}
          </article>
          <article className="manutencoes__chart">
            <h2>Ordens por estação</h2>
            {Object.entries(byStation).map(([label, value]) => (
              <div key={label} className="manutencoes__bar">
                <strong>{label}</strong>
                <i style={{ width: `${(value / maxStation) * 100}%` }} />
                <em>{value}</em>
              </div>
            ))}
          </article>
        </div>
      </div>

      <Filtros
        period={filters.period}
        onPeriod={(period) => {
          setPage(1)
          setFilters((current) => ({ ...current, period }))
        }}
        dirty={filtersDirty}
        onClear={() => {
          setPage(1)
          setFilters(emptyFilters)
        }}
      >
        <FiltroCampo label="Estação">
          <select value={filters.station} onChange={updateFilter('station')}>
            <option value="">Todas</option>
            {lineStations.map((station) => (
              <option key={station.code} value={station.code}>
                {station.code} — {station.name}
              </option>
            ))}
          </select>
        </FiltroCampo>
        <FiltroCampo label="Tipo">
          <select value={filters.orderType} onChange={updateFilter('orderType')}>
            <option value="">Todos</option>
            {orderTypes.map((type) => (
              <option key={type} value={type}>
                {type}
              </option>
            ))}
          </select>
        </FiltroCampo>
        <FiltroCampo label="Origem">
          <select value={filters.source} onChange={updateFilter('source')}>
            <option value="">Todas</option>
            {sources.map((source) => (
              <option key={source} value={source}>
                {source}
              </option>
            ))}
          </select>
        </FiltroCampo>
        <FiltroCampo label="Mín. (h)">
          <input
            type="number"
            min="0"
            step="0.25"
            value={filters.durationMin}
            onChange={updateFilter('durationMin')}
          />
        </FiltroCampo>
        <FiltroCampo label="Máx. (h)">
          <input
            type="number"
            min="0"
            step="0.25"
            value={filters.durationMax}
            onChange={updateFilter('durationMax')}
          />
        </FiltroCampo>
      </Filtros>

      <div className="manutencoes__list">
        {paged.length ? (
          paged.map((item) => (
            <CardManutencao
              key={item.id ?? `${item.sap_order_number}-${item.start_timestamp}-${item.station_code}`}
              orderType={item.order_type}
              sapOrder={item.sap_order_number}
              station={item.station_code}
              location={item.installation_location}
              startedAt={item.start_timestamp}
              endedAt={item.end_timestamp}
              durationHours={item.duration_hours}
              symptom={item.symptom}
              description={item.description}
              source={item.source}
            />
          ))
        ) : (
          <p className="page-empty">
            {items.length ? 'Nenhuma ordem com esses filtros.' : 'Nenhuma ordem de manutenção retornada pela API.'}
          </p>
        )}
        <Paginacao page={currentPage} pageSize={PAGE_SIZE} total={visible.length} onPage={setPage} />
      </div>
      </>
      )}

      {open ? (
        <div className="manutencoes-modal" onClick={() => !importing && setOpen(false)}>
          <form
            onClick={(event) => event.stopPropagation()}
            onSubmit={mode === 'import' ? handleImport : handleSubmit}
          >
            <h2>Cadastrar manutenção</h2>
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
                <label className="cadastro-select">
                  <span>Tipo da ordem</span>
                  <select value={orderType} onChange={(event) => setOrderType(event.target.value)}>
                    <option value="ZM01">ZM01 — Corretiva emergencial</option>
                    <option value="ZM02">ZM02 — Corretiva programada</option>
                    <option value="ZM03">ZM03 — Preventiva</option>
                  </select>
                </label>
                <PlanilhaImport
                  file={file}
                  onFile={(next) => {
                    setFile(next)
                    setImportError('')
                    setUpload(null)
                  }}
                  hint="Planilha ZM01, ZM02 ou ZM03 em CSV ou XLSX."
                  error={importError}
                />
                {upload ? <UploadStatus upload={upload} /> : null}
                <Button type="submit" variant="light">
                  {importing ? 'Processando upload…' : 'Enviar planilha'}
                </Button>
              </>
            ) : (
              <>
                {importError ? <p className="page-status is-error">{importError}</p> : null}
                <Input
                  id="mnt-type"
                  label="Tipo da ordem"
                  placeholder="ZM01"
                  value={form.order_type}
                  required
                  onChange={updateField('order_type')}
                />
                <Input
                  id="mnt-order"
                  label="Número da ordem SAP"
                  value={form.sap_order_number}
                  onChange={updateField('sap_order_number')}
                />
                <Input
                  id="mnt-station"
                  label="Estação"
                  placeholder="FAL"
                  value={form.station_code}
                  required
                  onChange={updateField('station_code')}
                />
                <Input
                  id="mnt-location"
                  label="Local de instalação"
                  placeholder="SP04-6-PSD-FAL-P1-PP04"
                  value={form.installation_location}
                  onChange={updateField('installation_location')}
                />
                <div className="ui-input-row">
                  <Input
                    id="mnt-start-date"
                    label="Data de início"
                    type="date"
                    value={form.start_date}
                    required
                    onChange={updateField('start_date')}
                  />
                  <Input
                    id="mnt-start-time"
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
                    id="mnt-end-date"
                    label="Data de fim"
                    type="date"
                    value={form.end_date}
                    onChange={updateField('end_date')}
                  />
                  <Input
                    id="mnt-end-time"
                    label="Hora de fim"
                    type="time"
                    step="60"
                    value={form.end_time}
                    onChange={updateField('end_time')}
                  />
                </div>
                <Input
                  id="mnt-duration"
                  label="Duração (horas)"
                  type="number"
                  step="0.25"
                  min="0"
                  value={form.duration_hours}
                  onChange={updateField('duration_hours')}
                />
                <Input
                  id="mnt-symptom"
                  label="Sintoma"
                  value={form.symptom}
                  onChange={updateField('symptom')}
                />
                <Input
                  id="mnt-desc"
                  label="Descrição"
                  value={form.description}
                  onChange={updateField('description')}
                />
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

export default Manutencoes
