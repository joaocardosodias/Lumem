import { useEffect, useMemo, useState } from 'react'
import Button from '../components/Button'
import { CountValue } from '../components/Card'
import Filtros, { FiltroCampo } from '../components/Filtros'
import Input from '../components/Input'
import Paginacao from '../components/Paginacao'
import { useToast } from '../context/Toast'
import { lineStations, mapStation, resolveStation, stationLabel } from '../data/stations'
import { api, errorMessage } from '../lib/api'
import { exportToCsv } from '../lib/exportCsv'
import { DEFAULT_PERIOD, PAGE_SIZE, inPeriod } from '../lib/filters'
import { getCachedInspections, loadInspections, patchCachedInspection } from '../lib/inspectionCache'
import './Inspecoes.css'

const emptyForm = {
  station_code: 'BUT',
  platform: 'P1',
  door_number: '',
  notes: '',
}

const STATUS_LABEL = {
  pending: 'Pendente',
  in_progress: 'Em andamento',
  completed: 'Concluída',
}

const emptyFilters = {
  station: '',
  platform: '',
  origin: '',
  period: DEFAULT_PERIOD,
}

function isAiInspection(item) {
  if (item.prediction_id) return true
  const notes = String(item.notes || '').toLowerCase()
  return notes.includes('automaticamente pela ia') || notes.includes('rodada #')
}

function inspectionStation(item) {
  return resolveStation(item) || String(item.station_code || '').toUpperCase()
}

function formatDateTime(value) {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return String(value)
  return new Intl.DateTimeFormat('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

function countBy(items, getKey) {
  return items.reduce((acc, item) => {
    const key = typeof getKey === 'function' ? getKey(item) : item[getKey]
    if (!key) return acc
    acc[key] = (acc[key] || 0) + 1
    return acc
  }, {})
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
  const [selectedInspection, setSelectedInspection] = useState(null)
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
    if (params?.inspectionId && items.length) {
      const match = items.find((i) => i.id === params.inspectionId)
      if (match) setSelectedInspection(match)
    }
    if (params?.openCreate) {
      setForm((prev) => ({
        ...prev,
        station_code: params.station || prev.station_code,
        door_number: params.door ? String(params.door) : prev.door_number,
      }))
      setOpen(true)
    }
  }, [params, items])

  useEffect(() => {
    let cancelled = false

    async function load() {
      if (!getCachedInspections()?.length) {
        setLoading(true)
      }
      setError('')
      try {
        const batch = await loadInspections({ force: Boolean(reloadAt) })
        if (!cancelled) setItems(batch)
      } catch (err) {
        if (!cancelled && !getCachedInspections()?.length) setError(errorMessage(err))
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
        if (filters.origin === 'ai' && !isAiInspection(item)) return false
        if (filters.origin === 'manual' && isAiInspection(item)) return false
        if (!inPeriod(item.created_at, filters.period)) return false
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
    Boolean(filters.station || filters.platform || filters.origin || search) ||
    filters.period !== DEFAULT_PERIOD
  const pageCount = Math.max(1, Math.ceil(visible.length / PAGE_SIZE))
  const currentPage = Math.min(page, pageCount)
  const paged = visible.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE)
  const pending = scoped.filter((item) => item.status === 'pending').length
  const inProgress = scoped.filter((item) => item.status === 'in_progress').length
  const completed = scoped.filter((item) => item.status === 'completed').length
  const aiGenerated = scoped.filter(isAiInspection).length
  const completionRate = scoped.length ? Math.round((completed / scoped.length) * 100) : 0

  const byStation = useMemo(() => {
    return countBy(visible, (item) => inspectionStation(item))
  }, [visible])

  const stationRows = useMemo(() => {
    return Object.entries(byStation).sort((a, b) => b[1] - a[1])
  }, [byStation])

  const maxStation = useMemo(() => {
    return Math.max(...Object.values(byStation), 1)
  }, [byStation])

  const topStations = useMemo(() => stationRows.slice(0, 6), [stationRows])
  const otherStations = useMemo(() => stationRows.slice(6), [stationRows])
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
      setSelectedInspection((current) => (current?.id === id ? { ...current, status } : current))
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
      { label: 'Origem', value: (item) => (isAiInspection(item) ? 'IA Preditiva' : 'Manual') },
      { label: 'Status', value: (item) => STATUS_LABEL[item.status] || item.status },
      { label: 'Notas', key: 'notes' },
    ]
    exportToCsv('inspecoes.csv', columns, visible)
    toast.success(`${visible.length} inspeções exportadas com sucesso!`)
  }

  return (
    <section className="inspecoes">
      <header className="page-head inspecoes__head">
        <div>
          <p className="inspecoes__eyebrow"><span aria-hidden="true" />Operação de campo</p>
          <h1>Inspeções</h1>
          <p className="inspecoes__lead">
            Acompanhe as portas que aguardam inspeção e atualize o trabalho em campo.
          </p>
        </div>
        <div className="page-head__actions">
          <Button variant="secondary" onClick={handleExportCsv}>
            Exportar CSV
          </Button>
          <Button onClick={() => setOpen(true)}>Nova inspeção</Button>
        </div>
      </header>

      {loading ? <p className="page-status">Carregando inspeções…</p> : null}
      {error ? <p className="page-status is-error">{error}</p> : null}

      {loading ? null : (
        <>
          <div className="inspecoes__hero">
            <section className="inspecoes__metrics" aria-label="Resumo das inspeções">
              <article className="inspecoes__metric inspecoes__metric--primary">
                <span className="inspecoes__metric-top">
                  <span>Pendentes</span>
                </span>
                <strong className="inspecoes__metric-value"><CountValue value={pending} /></strong>
                <span className="inspecoes__metric-description">
                  {pending ? 'Aguardando atendimento em campo' : 'Nenhuma pendência neste recorte'}
                </span>
                <button
                  type="button"
                  className="inspecoes__pending-action"
                  onClick={() => {
                    setFilter('pending')
                    document.getElementById('inspection-records-title')?.scrollIntoView({ block: 'start' })
                  }}
                >
                  Ver pendentes
                  <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
                    <path d="M3 8h10m-4-4 4 4-4 4" />
                  </svg>
                </button>
              </article>

              <article className="inspecoes__metric">
                <span className="inspecoes__metric-top">
                  <span>Em andamento</span>
                </span>
                <strong className="inspecoes__metric-value"><CountValue value={inProgress} /></strong>
                <span className="inspecoes__metric-description">
                  {inProgress ? 'Com atendimento iniciado' : 'Nenhuma ordem iniciada'}
                </span>
              </article>

              <article className="inspecoes__metric">
                <span className="inspecoes__metric-top">
                  <span>Concluídas</span>
                </span>
                <strong className="inspecoes__metric-value"><CountValue value={completed} /></strong>
                <span className="inspecoes__metric-description">
                  {scoped.length ? `${completionRate}% das ordens do recorte` : 'Nenhuma ordem no recorte'}
                </span>
              </article>

              <article className="inspecoes__metric">
                <span className="inspecoes__metric-top">
                  <span>Geradas pelo modelo</span>
                </span>
                <strong className="inspecoes__metric-value"><CountValue value={aiGenerated} /></strong>
                <span className="inspecoes__metric-description">
                  {aiGenerated ? `${aiGenerated} ordens automáticas` : 'Nenhuma ordem automática'}
                </span>
              </article>
            </section>
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
            <FiltroCampo label="Origem">
              <select value={filters.origin} onChange={updateFilter('origin')}>
                <option value="">Todas</option>
                <option value="ai">Geradas por IA</option>
                <option value="manual">Manuais</option>
              </select>
            </FiltroCampo>
          </Filtros>

          <section className="inspecoes__list" aria-labelledby="inspection-records-title">
            <header className="inspecoes__panel-head">
              <h2 id="inspection-records-title">Fila de inspeções</h2>
              <span className="inspecoes__record-count">
                {visible.length.toLocaleString('pt-BR')} {visible.length === 1 ? 'registro' : 'registros'}
              </span>
            </header>

            <div className="inspecoes-tabs" role="tablist" aria-label="Filtro de status">
              {[
                ['', 'Todas', scoped.length],
                ['pending', 'Pendentes', pending],
                ['in_progress', 'Em andamento', inProgress],
                ['completed', 'Concluídas', completed],
              ].map(([value, label, count]) => (
                <button
                  key={value || 'all'}
                  type="button"
                  role="tab"
                  aria-selected={filter === value}
                  className={filter === value ? 'is-active' : ''}
                  onClick={() => setFilter(value)}
                >
                  <span>{label}</span>
                  <small>{count}</small>
                </button>
              ))}
            </div>

            <div className="inspecoes__table-head" aria-hidden="true">
              <span>Identificação</span>
              <span>Estação</span>
              <span>Plataforma e porta</span>
              <span>Status</span>
              <span>Ações</span>
            </div>
            <div className="inspecoes__records">
              {visible.length ? (
                paged.map((item) => (
                  <article
                    key={item.id}
                    className="card-inspecao card-inspecao--clickable"
                    onClick={() => setSelectedInspection(item)}
                    role="button"
                    tabIndex={0}
                    onKeyDown={(e) => {
                      if (e.target !== e.currentTarget) return
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault()
                        setSelectedInspection(item)
                      }
                    }}
                  >
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
                      {item.notes && !isAiInspection(item) ? <em>{item.notes}</em> : null}
                    </div>
                    <label className="card-inspecao__status" onClick={(event) => event.stopPropagation()}>
                      <span>Status</span>
                      <select
                        className={`status-select--${item.status}`}
                        value={item.status}
                        onChange={(event) => handleStatus(item.id, event.target.value)}
                      >
                        <option value="pending">{STATUS_LABEL.pending}</option>
                        <option value="in_progress">{STATUS_LABEL.in_progress}</option>
                        <option value="completed">{STATUS_LABEL.completed}</option>
                      </select>
                    </label>
                    <div className="card-inspecao-actions" onClick={(event) => event.stopPropagation()}>
                      <button
                        type="button"
                        className="btn-inspecao-action btn-inspecao-action--primary"
                        onClick={() => setSelectedInspection(item)}
                      >
                        Ver detalhes
                      </button>
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
                        Ver porta
                      </button>
                    </div>
                  </article>
                ))
              ) : (
                <p className="page-empty">
                  {items.length ? 'Nenhuma inspeção com esses filtros.' : 'Nenhuma inspeção cadastrada.'}
                </p>
              )}
            </div>
            <Paginacao page={currentPage} pageSize={PAGE_SIZE} total={visible.length} onPage={setPage} />
          </section>

          <div className="inspecoes__charts">
            <article className="inspecoes__chart">
              <header className="inspecoes__panel-head">
                <h2>Ordens por estação</h2>
              </header>
              <div className="inspecoes__chart-body">
                {stationRows.length ? (
                  <div className="inspecoes__bars">
                    <div>
                      {topStations.map(([label, value]) => (
                        <div key={label} className="inspecoes__bar">
                          <strong>{label}</strong>
                          <span className="inspecoes__bar-track">
                            <i style={{ width: `${(value / maxStation) * 100}%` }} />
                          </span>
                          <em>{value}</em>
                        </div>
                      ))}
                    </div>
                    {otherStations.length ? (
                      <div>
                        {otherStations.map(([label, value]) => (
                          <div key={label} className="inspecoes__bar">
                            <strong>{label}</strong>
                            <span className="inspecoes__bar-track">
                              <i style={{ width: `${(value / maxStation) * 100}%` }} />
                            </span>
                            <em>{value}</em>
                          </div>
                        ))}
                      </div>
                    ) : null}
                  </div>
                ) : (
                  <p className="inspecoes__empty">Nenhuma estação com ordens neste recorte.</p>
                )}
              </div>
            </article>
          </div>
        </>
      )}

      {selectedInspection ? (
        <div
          className="inspecoes-modal inspecoes-modal--detalhes"
          onClick={() => setSelectedInspection(null)}
        >
          <div
            className="inspecao-detalhes"
            onClick={(event) => event.stopPropagation()}
            role="dialog"
            aria-modal="true"
            aria-labelledby="detalhes-titulo"
          >
            <header className="inspecao-detalhes__head">
              <div>
                <div className="inspecao-detalhes__tags">
                  <span className="inspecao-tag-id">Ordem #{selectedInspection.id}</span>
                  <span className={`inspecao-badge inspecao-badge--${selectedInspection.status}`}>
                    {STATUS_LABEL[selectedInspection.status] || selectedInspection.status}
                  </span>
                  <span className={`inspecao-badge inspecao-badge--${isAiInspection(selectedInspection) ? 'ai' : 'manual'}`}>
                    {isAiInspection(selectedInspection) ? 'IA Preditiva' : 'Manual'}
                  </span>
                </div>
                <h2 id="detalhes-titulo">
                  {`SP04-6-PSD-${selectedInspection.station_code}-${selectedInspection.platform}-PP${String(selectedInspection.door_number).padStart(2, '0')}`}
                </h2>
                <p className="inspecao-detalhes__sub">
                  Estação {selectedInspection.station_code} ({stationLabel(selectedInspection.station_code)}) · Plataforma {selectedInspection.platform} · Porta {String(selectedInspection.door_number).padStart(2, '0')}
                </p>
              </div>
              <button
                type="button"
                className="inspecao-detalhes__close"
                onClick={() => setSelectedInspection(null)}
                aria-label="Fechar detalhes"
              >
                ✕
              </button>
            </header>

            <div className="inspecao-detalhes__body">
              <div className="inspecao-detalhes__grid">
                <div className="inspecao-detalhe-item">
                  <span>TAG SAP</span>
                  <strong>{`SP04-6-PSD-${selectedInspection.station_code}-${selectedInspection.platform}-PP${String(selectedInspection.door_number).padStart(2, '0')}`}</strong>
                </div>
                <div className="inspecao-detalhe-item">
                  <span>Estação</span>
                  <strong>{selectedInspection.station_code} — {stationLabel(selectedInspection.station_code)}</strong>
                </div>
                <div className="inspecao-detalhe-item">
                  <span>Localização</span>
                  <strong>Plataforma {selectedInspection.platform} · Porta {String(selectedInspection.door_number).padStart(2, '0')}</strong>
                </div>
                <div className="inspecao-detalhe-item">
                  <span>Origem da Ordem</span>
                  <strong>{isAiInspection(selectedInspection) ? 'Detecção Preditiva IA' : 'Abertura Manual de Campo'}</strong>
                </div>
                <div className="inspecao-detalhe-item">
                  <span>Data de Abertura</span>
                  <strong>{formatDateTime(selectedInspection.created_at)}</strong>
                </div>
                <div className="inspecao-detalhe-item">
                  <span>Última Atualização</span>
                  <strong>{formatDateTime(selectedInspection.updated_at)}</strong>
                </div>
                {selectedInspection.completed_at ? (
                  <div className="inspecao-detalhe-item">
                    <span>Data de Conclusão</span>
                    <strong>{formatDateTime(selectedInspection.completed_at)}</strong>
                  </div>
                ) : null}
              </div>

              <div className="inspecao-detalhes__notes">
                <h4>Observações / Diagnóstico</h4>
                <div className="inspecao-detalhes__notes-content">
                  {selectedInspection.notes ? (
                    <p>{selectedInspection.notes}</p>
                  ) : (
                    <p className="inspecao-detalhes__no-notes">Nenhuma anotação técnica registrada para esta ordem de inspeção.</p>
                  )}
                </div>
              </div>

              <div className="inspecao-detalhes__status-flow">
                <h4>Alterar Status da Ordem</h4>
                <div className="inspecao-detalhes__status-actions">
                  <button
                    type="button"
                    disabled={selectedInspection.status === 'pending'}
                    className={`btn-flow-status btn-flow-status--pending ${selectedInspection.status === 'pending' ? 'is-current' : ''}`}
                    onClick={() => handleStatus(selectedInspection.id, 'pending')}
                  >
                    Marcar como Pendente
                  </button>
                  <button
                    type="button"
                    disabled={selectedInspection.status === 'in_progress'}
                    className={`btn-flow-status btn-flow-status--in_progress ${selectedInspection.status === 'in_progress' ? 'is-current' : ''}`}
                    onClick={() => handleStatus(selectedInspection.id, 'in_progress')}
                  >
                    Iniciar Atendimento
                  </button>
                  <button
                    type="button"
                    disabled={selectedInspection.status === 'completed'}
                    className={`btn-flow-status btn-flow-status--completed ${selectedInspection.status === 'completed' ? 'is-current' : ''}`}
                    onClick={() => handleStatus(selectedInspection.id, 'completed')}
                  >
                    Concluir Inspeção
                  </button>
                </div>
              </div>
            </div>

            <footer className="inspecao-detalhes__foot">
              <Button
                type="button"
                variant="secondary"
                onClick={() => {
                  const item = selectedInspection
                  setSelectedInspection(null)
                  onNavigate?.('portas', {
                    station: item.station_code,
                    platform: item.platform,
                    door: item.door_number,
                  })
                }}
              >
                Ver porta no mapa
              </Button>
              <Button type="button" onClick={() => setSelectedInspection(null)}>
                Fechar
              </Button>
            </footer>
          </div>
        </div>
      ) : null}

      {open ? (
        <div className="inspecoes-modal" onClick={() => setOpen(false)}>
          <form onClick={(event) => event.stopPropagation()} onSubmit={handleSubmit}>
            <h2>Nova ordem de inspeção</h2>
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
            <Input id="insp-notes" label="Notas" value={form.notes} onChange={updateField('notes')} />
            <Button type="submit">
              Abrir ordem
            </Button>
          </form>
        </div>
      ) : null}
    </section>
  )
}

export default Inspecoes
