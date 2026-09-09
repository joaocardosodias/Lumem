import { useEffect, useMemo, useState } from 'react'
import Button from '../components/Button'
import CadastrarButton from '../components/CadastrarButton'
import Card from '../components/Card'
import Input from '../components/Input'
import Paginacao from '../components/Paginacao'
import { lineStations, stationLabel } from '../data/stations'
import { api, errorMessage } from '../lib/api'
import { PAGE_SIZE } from '../lib/filters'
import { getCachedInspections, loadInspections, patchCachedInspection } from '../lib/inspectionCache'
import './Inspecoes.css'

const emptyForm = {
  station_code: 'BUT',
  platform: 'P1',
  door_number: '',
  priority: 'normal',
  date: '',
  time: '',
  notes: '',
}

const STATUS_LABEL = {
  pending: 'Pendente',
  in_progress: 'Em andamento',
  completed: 'Concluída',
}

const PRIORITY_LABEL = {
  low: 'Baixa',
  normal: 'Normal',
  high: 'Alta',
  critical: 'Crítica',
}

function toTimestamp(date, time) {
  if (!date) return ''
  const clock = time ? (time.length === 5 ? `${time}:00` : time) : '00:00:00'
  return `${date} ${clock}`
}

function Inspecoes() {
  const [items, setItems] = useState(() => getCachedInspections() ?? [])
  const [loading, setLoading] = useState(() => !getCachedInspections())
  const [error, setError] = useState('')
  const [filter, setFilter] = useState('')
  const [page, setPage] = useState(1)
  const [open, setOpen] = useState(false)
  const [form, setForm] = useState(emptyForm)
  const [formError, setFormError] = useState('')
  const [reloadAt, setReloadAt] = useState(0)

  useEffect(() => {
    let cancelled = false

    async function load() {
      setLoading(true)
      setError('')
      try {
        const batch = await loadInspections({ force: Boolean(reloadAt) })
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

  useEffect(() => {
    setPage(1)
  }, [filter])

  const visible = useMemo(
    () => (filter ? items.filter((item) => item.status === filter) : items),
    [items, filter],
  )
  const pageCount = Math.max(1, Math.ceil(visible.length / PAGE_SIZE))
  const currentPage = Math.min(page, pageCount)
  const paged = visible.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE)
  const pending = items.filter((item) => item.status === 'pending').length
  const inProgress = items.filter((item) => item.status === 'in_progress').length
  const completed = items.filter((item) => item.status === 'completed').length
  const selectedStation = lineStations.find((item) => item.code === form.station_code)
  const platforms = selectedStation?.platforms ?? ['P1', 'P2']

  function updateField(field) {
    return (event) => {
      const value = event.target.value
      setForm((current) => {
        const next = { ...current, [field]: value }
        if (field === 'station_code') {
          const station = lineStations.find((item) => item.code === value)
          if (station && !station.platforms.includes(current.platform)) {
            next.platform = station.platforms[0]
          }
        }
        return next
      })
    }
  }

  async function handleSubmit(event) {
    event.preventDefault()
    setFormError('')
    try {
      await api.createInspection({
        station_code: form.station_code,
        platform: form.platform,
        door_number: String(form.door_number),
        priority: form.priority,
        scheduled_for: toTimestamp(form.date, form.time) || undefined,
        notes: form.notes.trim() || undefined,
      })
      setForm(emptyForm)
      setOpen(false)
      setReloadAt(Date.now())
    } catch (err) {
      setFormError(errorMessage(err))
    }
  }

  async function handleStatus(id, status) {
    try {
      await api.updateInspectionStatus(id, { status })
      patchCachedInspection(id, { status })
      setItems((current) => current.map((item) => (item.id === id ? { ...item, status } : item)))
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  return (
    <section className="inspecoes">
      <div className="page-head">
        <h1>Inspeções</h1>
        <CadastrarButton onClick={() => setOpen(true)}>Abrir inspeção +</CadastrarButton>
      </div>

      {loading ? <p className="page-status">Carregando inspeções…</p> : null}
      {error ? <p className="page-status is-error">{error}</p> : null}

      <div className="inspecoes__kpis">
        <Card layout="banner" label="Pendentes" value={pending} hint="aguardando campo" />
        <Card layout="banner" label="Em andamento" value={inProgress} hint="em execução" />
        <Card layout="banner" label="Concluídas" value={completed} hint="finalizadas" />
      </div>

      <div className="inspecoes-tabs" role="tablist" aria-label="Filtro de status">
        {[
          ['', 'Todas'],
          ['pending', 'Pendentes'],
          ['in_progress', 'Em andamento'],
          ['completed', 'Concluídas'],
        ].map(([value, label]) => (
          <button
            key={value || 'all'}
            type="button"
            role="tab"
            aria-selected={filter === value}
            className={filter === value ? 'is-active' : ''}
            onClick={() => setFilter(value)}
          >
            {label}
          </button>
        ))}
      </div>

      <div className="inspecoes__list">
        {visible.length ? (
          paged.map((item) => (
            <article key={item.id} className="card-inspecao">
              <div>
                <span>Estação</span>
                <strong>{item.station_code}</strong>
                <em>{stationLabel(item.station_code)}</em>
              </div>
              <div>
                <span>Plataforma</span>
                <strong>{item.platform}</strong>
              </div>
              <div>
                <span>Porta</span>
                <strong>{String(item.door_number).padStart(2, '0')}</strong>
              </div>
              <div>
                <span>Prioridade</span>
                <strong>{PRIORITY_LABEL[item.priority] || item.priority}</strong>
              </div>
              <div>
                <span>Agendada para</span>
                <strong>{item.scheduled_for || '—'}</strong>
              </div>
              <div>
                <span>Notas</span>
                <strong>{item.notes || '—'}</strong>
              </div>
              <label className="card-inspecao__status">
                <span>Status</span>
                <select
                  value={item.status}
                  onChange={(event) => handleStatus(item.id, event.target.value)}
                >
                  <option value="pending">{STATUS_LABEL.pending}</option>
                  <option value="in_progress">{STATUS_LABEL.in_progress}</option>
                  <option value="completed">{STATUS_LABEL.completed}</option>
                </select>
              </label>
            </article>
          ))
        ) : (
          <p className="page-empty">Nenhuma inspeção retornada pela API.</p>
        )}
        <Paginacao page={currentPage} pageSize={PAGE_SIZE} total={visible.length} onPage={setPage} />
      </div>

      {open ? (
        <div className="inspecoes-modal" onClick={() => setOpen(false)}>
          <form onClick={(event) => event.stopPropagation()} onSubmit={handleSubmit}>
            <h2>Abrir inspeção</h2>
            {formError ? <p className="page-status is-error">{formError}</p> : null}
            <label className="cadastro-select">
              <span>Estação</span>
              <select value={form.station_code} required onChange={updateField('station_code')}>
                {lineStations.map((station) => (
                  <option key={station.code} value={station.code}>
                    {station.code} — {station.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="cadastro-select">
              <span>Plataforma</span>
              <select value={form.platform} required onChange={updateField('platform')}>
                {platforms.map((platform) => (
                  <option key={platform} value={platform}>
                    {platform}
                  </option>
                ))}
              </select>
            </label>
            <Input
              id="insp-door"
              label="Número da porta"
              type="number"
              min="1"
              max="24"
              value={form.door_number}
              required
              onChange={updateField('door_number')}
            />
            <label className="cadastro-select">
              <span>Prioridade</span>
              <select value={form.priority} onChange={updateField('priority')}>
                <option value="low">low — Baixa</option>
                <option value="normal">normal — Normal</option>
                <option value="high">high — Alta</option>
                <option value="critical">critical — Crítica</option>
              </select>
            </label>
            <div className="ui-input-row">
              <Input id="insp-date" label="Data" type="date" value={form.date} onChange={updateField('date')} />
              <Input id="insp-time" label="Hora" type="time" step="60" value={form.time} onChange={updateField('time')} />
            </div>
            <Input id="insp-notes" label="Notas" value={form.notes} onChange={updateField('notes')} />
            <Button type="submit" variant="light">
              Abrir ordem
            </Button>
          </form>
        </div>
      ) : null}
    </section>
  )
}

export default Inspecoes
