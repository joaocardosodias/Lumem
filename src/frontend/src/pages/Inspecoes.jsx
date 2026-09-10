import { useEffect, useMemo, useState } from 'react'
import Button from '../components/Button'
import CadastrarButton from '../components/CadastrarButton'
import Card from '../components/Card'
import Filtros, { FiltroCampo } from '../components/Filtros'
import Input from '../components/Input'
import Paginacao from '../components/Paginacao'
import { useToast } from '../context/Toast'
import { lineStations, mapStation, resolveStation, stationLabel } from '../data/stations'
import { api, errorMessage } from '../lib/api'
import { exportToCsv } from '../lib/exportCsv'
import { DEFAULT_PERIOD, PAGE_SIZE, inPeriod, periodHint } from '../lib/filters'
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

const emptyFilters = {
  station: '',
  platform: '',
  priority: '',
  origin: '',
  period: DEFAULT_PERIOD,
}

function toTimestamp(date, time) {
  if (!date) return ''
  const clock = time ? (time.length === 5 ? `${time}:00` : time) : '00:00:00'
  return `${date} ${clock}`
}

function isAiInspection(item) {
  const notes = String(item.notes || '').toLowerCase()
  return notes.includes('automaticamente pela ia') || notes.includes('rodada #')
}

function inspectionStation(item) {
  return resolveStation(item) || String(item.station_code || '').toUpperCase()
}

function Inspecoes({ onNavigate, params }) {
  const toast = useToast()
  const [items, setItems] = useState(() => getCachedInspections() ?? [])
  const [loading, setLoading] = useState(() => !getCachedInspections())
  const [error, setError] = useState('')
  const [filter, setFilter] = useState('')
  const [search, setSearch] = useState('')
  const [filters, setFilters] = useState(emptyFilters)
  const [page, setPage] = useState(1)
  const [open, setOpen] = useState(false)
  const [form, setForm] = useState(emptyForm)
  const [formError, setFormError] = useState('')
  const [reloadAt, setReloadAt] = useState(0)

  useEffect(() => {
    if (params?.status) {
      setFilter(params.status)
    }
    if (params?.origin === 'ai') {
      setFilters((prev) => ({ ...prev, origin: 'ai' }))
    }
    if (params?.station) {
      setFilters((prev) => ({ ...prev, station: params.station }))
    }
    if (params?.door) {
      setSearch(String(params.door))
    }
    if (params?.openCreate) {
      setForm((prev) => ({
        ...prev,
        station_code: params.station || prev.station_code,
        door_number: params.door ? String(params.door) : prev.door_number,
      }))
      setOpen(true)
    }
  }, [params])

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
  }, [filter, filters, search])

  const scoped = useMemo(
    () =>
      items.filter((item) => {
        if (filters.station && mapStation(inspectionStation(item)) !== mapStation(filters.station)) return false
        if (filters.platform && item.platform !== filters.platform) return false
        if (filters.priority && item.priority !== filters.priority) return false
        if (filters.origin === 'ai' && !isAiInspection(item)) return false
        if (filters.origin === 'manual' && isAiInspection(item)) return false
        if (!inPeriod(item.scheduled_for || item.created_at, filters.period)) return false
        if (search.trim()) {
          const q = search.trim().toLowerCase()
          const match =
            String(item.door_number).includes(q) ||
            (item.notes && item.notes.toLowerCase().includes(q)) ||
            (item.station_code && item.station_code.toLowerCase().includes(q)) ||
            (item.platform && item.platform.toLowerCase().includes(q))
          if (!match) return false
        }
        return true
      }),
    [items, filters, search],
  )

  const visible = useMemo(
    () => (filter ? scoped.filter((item) => item.status === filter) : scoped),
    [scoped, filter],
  )
  const filtersDirty =
    Boolean(filters.station || filters.platform || filters.priority || filters.origin || search) ||
    filters.period !== DEFAULT_PERIOD
  const pageCount = Math.max(1, Math.ceil(visible.length / PAGE_SIZE))
  const currentPage = Math.min(page, pageCount)
  const paged = visible.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE)
  const rangeHint = periodHint(filters.period)
  const pending = scoped.filter((item) => item.status === 'pending').length
  const inProgress = scoped.filter((item) => item.status === 'in_progress').length
  const completed = scoped.filter((item) => item.status === 'completed').length
  const selectedStation = lineStations.find((item) => item.code === form.station_code)
  const platforms = selectedStation?.platforms ?? ['P1', 'P2']
  const filterPlatforms = filters.station
    ? lineStations.find((station) => station.code === filters.station)?.platforms ?? []
    : [...new Set(items.map((item) => item.platform).filter(Boolean))]

  function updateFilter(field) {
    return (event) =>
      setFilters((current) => {
        const next = { ...current, [field]: event.target.value }
        if (field === 'station') next.platform = ''
        return next
      })
  }

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
      toast.success('Ordem de inspeção criada com sucesso!')
      setReloadAt(Date.now())
    } catch (err) {
      const msg = errorMessage(err)
      setFormError(msg)
      toast.error(msg)
    }
  }

  async function handleStatus(id, status) {
    try {
      await api.updateInspectionStatus(id, { status })
      patchCachedInspection(id, { status })
      setItems((current) => current.map((item) => (item.id === id ? { ...item, status } : item)))
      toast.success(`Status atualizado para ${STATUS_LABEL[status] || status}`)
    } catch (err) {
      const msg = errorMessage(err)
      setError(msg)
      toast.error(msg)
    }
  }

  function handleExportCsv() {
    if (!visible.length) {
      toast.info('Não há registros para exportar com os filtros atuais.')
      return
    }
    const columns = [
      { label: 'Estação', key: 'station_code' },
      { label: 'Nome da Estação', value: (item) => stationLabel(item.station_code) },
      { label: 'Plataforma', key: 'platform' },
      { label: 'Porta', value: (item) => String(item.door_number).padStart(2, '0') },
      { label: 'Prioridade', value: (item) => PRIORITY_LABEL[item.priority] || item.priority },
      { label: 'Origem', value: (item) => (isAiInspection(item) ? 'IA Preditiva' : 'Manual') },
      { label: 'Status', value: (item) => STATUS_LABEL[item.status] || item.status },
      { label: 'Agendada para', key: 'scheduled_for' },
      { label: 'Notas', key: 'notes' },
    ]
    exportToCsv('inspecoes.csv', columns, visible)
    toast.success(`${visible.length} inspeções exportadas com sucesso!`)
  }

  return (
    <section className="inspecoes">
      <div className="page-head">
        <div>
          <p className="dashboard__kicker">Ordens e Manutenção de Campo</p>
          <h1>Inspeções</h1>
          <p className="dashboard__lead">
            Acompanhamento de ordens de serviço e inspeções geradas pela IA e equipe operacional.
          </p>
        </div>
        <div className="page-head__actions">
          <CadastrarButton variant="secondary" onClick={handleExportCsv}>
            Exportar CSV
          </CadastrarButton>
          <CadastrarButton onClick={() => setOpen(true)}>Abrir inspeção +</CadastrarButton>
        </div>
      </div>

      {loading ? <p className="page-status">Carregando inspeções…</p> : null}
      {error ? <p className="page-status is-error">{error}</p> : null}

      <div className="inspecoes__kpis">
        <Card
          layout="banner"
          label="Pendentes"
          value={pending}
          hint={rangeHint ? `aguardando campo · ${rangeHint}` : 'aguardando campo'}
        />
        <Card
          layout="banner"
          label="Em andamento"
          value={inProgress}
          hint={rangeHint ? `em execução · ${rangeHint}` : 'em execução'}
        />
        <Card
          layout="banner"
          label="Concluídas"
          value={completed}
          hint={rangeHint ? `finalizadas · ${rangeHint}` : 'finalizadas'}
        />
      </div>

      <Filtros
        period={filters.period}
        onPeriod={(period) => setFilters((current) => ({ ...current, period }))}
        dirty={filtersDirty}
        onClear={() => {
          setSearch('')
          setFilters(emptyFilters)
        }}
      >
        <FiltroCampo label="Busca">
          <input
            type="search"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Buscar por porta, notas..."
          />
        </FiltroCampo>
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
        <FiltroCampo label="Plataforma">
          <select value={filters.platform} onChange={updateFilter('platform')}>
            <option value="">Todas</option>
            {filterPlatforms.map((platform) => (
              <option key={platform} value={platform}>
                {platform}
              </option>
            ))}
          </select>
        </FiltroCampo>
        <FiltroCampo label="Prioridade">
          <select value={filters.priority} onChange={updateFilter('priority')}>
            <option value="">Todas</option>
            {Object.entries(PRIORITY_LABEL).map(([value, label]) => (
              <option key={value} value={value}>
                {label}
              </option>
            ))}
          </select>
        </FiltroCampo>
        <FiltroCampo label="Origem">
          <select value={filters.origin} onChange={updateFilter('origin')}>
            <option value="">Todas</option>
            <option value="ai">Geradas por IA</option>
            <option value="manual">Manuais</option>
          </select>
        </FiltroCampo>
      </Filtros>

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
                <span>TAG SAP</span>
                <strong>{`SP04-6-PSD-${item.station_code}-${item.platform}-PP${String(item.door_number).padStart(2, '0')}`}</strong>
              </div>
              <div>
                <span>Estação</span>
                <strong>{item.station_code}</strong>
                <em>{stationLabel(item.station_code)}</em>
              </div>
              <div>
                <span>Plat / Porta</span>
                <strong>{item.platform} · P{String(item.door_number).padStart(2, '0')}</strong>
              </div>
              <div>
                <span>Prioridade</span>
                <strong className={item.priority === 'critical' ? 'priority--critical' : undefined}>
                  {PRIORITY_LABEL[item.priority] || item.priority}
                </strong>
                {isAiInspection(item) ? (
                  <span className="badge-ia-inspecao">Gerada por IA</span>
                ) : null}
              </div>
              <div>
                <span>Agendada para</span>
                <strong>{item.scheduled_for || 'Imediata'}</strong>
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
              <div className="card-inspecao-actions">
                <button
                  type="button"
                  className="btn-inspecao-action"
                  onClick={() =>
                    onNavigate?.('portas', {
                      station: item.station_code,
                      platform: item.platform,
                      door: item.door_number,
                    })
                  }
                >
                  Ver Porta
                </button>
              </div>
            </article>
          ))
        ) : (
          <p className="page-empty">
            {items.length ? 'Nenhuma inspeção com esses filtros.' : 'Nenhuma inspeção retornada pela API.'}
          </p>
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
              <Input
                id="insp-date"
                label="Data"
                type="date"
                value={form.date}
                onChange={updateField('date')}
              />
              <Input
                id="insp-time"
                label="Hora"
                type="time"
                step="60"
                value={form.time}
                onChange={updateField('time')}
              />
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
