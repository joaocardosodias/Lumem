import { useEffect, useMemo, useState } from 'react'
import Button from '../components/Button'
import { CountValue } from '../components/Card'
import Filtros, { FiltroCampo } from '../components/Filtros'
import Paginacao from '../components/Paginacao'
import PlanilhaImport from '../components/PlanilhaImport'
import UploadStatus from '../components/UploadStatus'
import { useToast } from '../context/Toast'
import { lineStations, mapStation, stationLabel } from '../data/stations'
import { api, asList, errorMessage } from '../lib/api'
import { exportToCsv } from '../lib/exportCsv'
import { DEFAULT_PERIOD, PAGE_SIZE, inPeriod, periodHint } from '../lib/filters'
import { pollUpload } from '../lib/runUpload'
import { getCachedAlarms, invalidateAlarms, setCachedAlarms } from '../lib/alarmCache'
import './Alarmes.css'

function mapAlarm(item) {
  return {
    id: item.id,
    tag: item.tag || `SP04-6-PSD-${item.station_code || 'BUT'}-${item.platform || 'P1'}-PP${String(item.door_number || '01').padStart(2, '0')}`,
    event_timestamp: item.event_timestamp || '—',
    station_code: item.station_code || '',
    station_name: stationLabel(item.station_code) || item.station_code || '—',
    platform: item.platform || 'P1',
    door_number: item.door_number,
    alarm_type: item.alarm_type || 'Alarme de Telemetria',
    duration_seconds: item.duration_seconds || 0,
    normalization_timestamp: item.normalization_timestamp || '',
    source: item.source || 'telemetria',
  }
}

function countBy(items, key) {
  return items.reduce((acc, item) => {
    const value = item[key]
    if (!value) return acc
    acc[value] = (acc[value] || 0) + 1
    return acc
  }, {})
}

const emptyFilters = {
  station: '',
  platform: '',
  alarmType: '',
  normalization: '',
  period: DEFAULT_PERIOD,
  search: '',
}

function Alarmes({ onNavigate, params }) {
  const toast = useToast()
  const cachedAlarms = getCachedAlarms()
  const [items, setItems] = useState(() => cachedAlarms || [])
  const [loading, setLoading] = useState(() => cachedAlarms === null)
  const [error, setError] = useState('')
  const [openModal, setOpenModal] = useState(false)
  const [file, setFile] = useState(null)
  const [importError, setImportError] = useState('')
  const [importing, setImporting] = useState(false)
  const [upload, setUpload] = useState(null)
  const [filters, setFilters] = useState(() => ({
    ...emptyFilters,
    station: params?.station || '',
  }))
  const [page, setPage] = useState(1)

  async function load(isBackground = false) {
    if (!isBackground && getCachedAlarms() === null) {
      setLoading(true)
    }
    setError('')
    try {
      const response = await api.alarms({ limit: 500 })
      const list = asList(response?.data || response?.alarms || response)
      const mapped = list.map(mapAlarm)
      setItems(mapped)
      setCachedAlarms(mapped)
    } catch (err) {
      if (getCachedAlarms() === null) {
        setError(errorMessage(err))
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load(cachedAlarms !== null)
  }, [])

  async function handleImportSpreadsheet(event) {
    event.preventDefault()
    if (!file) {
      setImportError('Selecione um arquivo CSV ou XLSX com os alarmes.')
      return
    }

    setImporting(true)
    setImportError('')

    try {
      const result = await api.uploadAlarms(file)
      const done = await pollUpload(result, setUpload)
      if (done.status === 2) {
        throw new Error('A API recusou o processamento da planilha de alarmes.')
      }
      setFile(null)
      setUpload(null)
      setOpenModal(false)
      invalidateAlarms()
      toast.success('Planilha de alarmes processada com sucesso!')
      await load()
    } catch (err) {
      const msg = errorMessage(err)
      setImportError(msg)
      toast.error(msg)
    } finally {
      setImporting(false)
    }
  }

  const platforms = useMemo(() => {
    if (filters.station) {
      return lineStations.find((s) => s.code === filters.station)?.platforms ?? []
    }
    return [...new Set(items.map((i) => i.platform).filter(Boolean))]
  }, [items, filters.station])

  const alarmTypes = useMemo(() => {
    return [...new Set(items.map((i) => i.alarm_type).filter(Boolean))].sort()
  }, [items])

  const visible = useMemo(() => {
    return items.filter((item) => {
      if (filters.station && mapStation(item.station_code) !== mapStation(filters.station)) {
        return false
      }
      if (filters.platform && item.platform !== filters.platform) return false
      if (filters.alarmType && item.alarm_type !== filters.alarmType) return false
      if (filters.normalization === 'open' && item.normalization_timestamp) return false
      if (filters.normalization === 'normalized' && !item.normalization_timestamp) return false
      if (!inPeriod(item.event_timestamp, filters.period)) return false
      if (filters.search.trim()) {
        const q = filters.search.trim().toLowerCase()
        const match =
          item.tag.toLowerCase().includes(q) ||
          item.alarm_type.toLowerCase().includes(q) ||
          item.station_name.toLowerCase().includes(q) ||
          item.station_code.toLowerCase().includes(q) ||
          String(item.door_number).includes(q)
        if (!match) return false
      }
      return true
    })
  }, [items, filters])

  const filtersDirty =
    Boolean(filters.station || filters.platform || filters.alarmType || filters.normalization || filters.search) ||
    filters.period !== DEFAULT_PERIOD

  const pageCount = Math.max(1, Math.ceil(visible.length / PAGE_SIZE))
  const currentPage = Math.min(page, pageCount)
  const paged = visible.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE)
  const rangeHint = periodHint(filters.period)

  const typeCounts = useMemo(() => countBy(visible, 'alarm_type'), [visible])
  const typeRows = useMemo(
    () => Object.entries(typeCounts).sort((a, b) => b[1] - a[1]),
    [typeCounts],
  )
  const topAlarmType = typeRows[0]
  const maxType = useMemo(
    () => Math.max(...typeRows.map(([, count]) => count), 1),
    [typeRows],
  )

  const stationCounts = useMemo(() => countBy(visible, 'station_name'), [visible])
  const stationRows = useMemo(
    () => Object.entries(stationCounts).sort((a, b) => b[1] - a[1]),
    [stationCounts],
  )
  const maxStation = useMemo(
    () => Math.max(...stationRows.map(([, count]) => count), 1),
    [stationRows],
  )

  const durations = useMemo(
    () =>
      visible
        .map((item) => Number(item.duration_seconds))
        .filter((val) => Number.isFinite(val) && val > 0),
    [visible],
  )
  const averageDuration = useMemo(() => {
    if (!durations.length) return null
    const sum = durations.reduce((acc, val) => acc + val, 0)
    return Math.round(sum / durations.length)
  }, [durations])

  const normalizedInVisible = useMemo(
    () => visible.filter((i) => Boolean(i.normalization_timestamp)).length,
    [visible],
  )
  const normalizationRate = useMemo(
    () => (visible.length ? Math.round((normalizedInVisible / visible.length) * 100) : 0),
    [visible.length, normalizedInVisible],
  )

  function handleExportCsv() {
    if (!visible.length) {
      toast.info('Não há registros para exportar com os filtros atuais.')
      return
    }
    const columns = [
      { label: 'Tag SAP', key: 'tag' },
      { label: 'Horário do Evento', key: 'event_timestamp' },
      { label: 'Estação', key: 'station_code' },
      { label: 'Nome da Estação', key: 'station_name' },
      { label: 'Plataforma', key: 'platform' },
      { label: 'Porta', key: 'door_number' },
      { label: 'Tipo de Alarme', key: 'alarm_type' },
      { label: 'Duração (s)', key: 'duration_seconds' },
      { label: 'Normalização', value: (item) => item.normalization_timestamp || 'Em aberto' },
      { label: 'Origem', key: 'source' },
    ]
    exportToCsv('alarmes_telemetria.csv', columns, visible)
    toast.success(`${visible.length} alarmes exportados com sucesso!`)
  }

  function updateFilter(field) {
    return (event) => {
      const val = event.target.value
      setFilters((curr) => {
        const next = { ...curr, [field]: val }
        if (field === 'station') next.platform = ''
        return next
      })
      setPage(1)
    }
  }

  return (
    <section className="alarmes">
      <header className="page-head alarmes__head">
        <div>
          <p className="alarmes__eyebrow"><span aria-hidden="true" />Telemetria das portas</p>
          <h1>Alarmes</h1>
          <p className="alarmes__lead">
            Localize eventos abertos e consulte o histórico recebido das portas da Linha 4.
          </p>
        </div>
        <div className="page-head__actions">
          <Button variant="secondary" onClick={handleExportCsv}>
            Exportar CSV
          </Button>
          <Button
            onClick={() => {
              setImportError('')
              setFile(null)
              setUpload(null)
              setOpenModal(true)
            }}
          >
            Importar alarmes
          </Button>
        </div>
      </header>

      {loading ? <p className="page-status">Carregando alarmes do banco…</p> : null}
      {error ? <p className="page-status is-error">{error}</p> : null}

      {loading ? null : (
        <>
          <div className="alarmes__hero">
            <section className="alarmes__metrics" aria-label="Resumo dos alarmes">
              <article className="alarmes__metric alarmes__metric--primary">
                <span className="alarmes__metric-top">
                  <span>Em aberto</span>
                </span>
                <strong className="alarmes__metric-value">
                  <CountValue value={visible.length - normalizedInVisible} />
                </strong>
                <span className="alarmes__metric-description">
                  {visible.length - normalizedInVisible ? 'Eventos sem normalização registrada' : 'Nenhum evento aberto no recorte'}
                </span>
              </article>

              <article className="alarmes__metric">
                <span className="alarmes__metric-top">
                  <span>Eventos no recorte</span>
                </span>
                <strong className="alarmes__metric-value"><CountValue value={visible.length} /></strong>
                <span className="alarmes__metric-description">{rangeHint || 'Todo o período disponível'}</span>
              </article>

              <article className="alarmes__metric">
                <span className="alarmes__metric-top">
                  <span>Tipo mais recorrente</span>
                </span>
                <strong
                  className={`alarmes__metric-value alarmes__metric-value--code${String(topAlarmType?.[0] || '').length <= 5 ? ' is-short' : ''}`}
                  title={topAlarmType?.[0]}
                >
                  {topAlarmType?.[0] || '—'}
                </strong>
                <span className="alarmes__metric-description">
                  {topAlarmType ? `${topAlarmType[1].toLocaleString('pt-BR')} eventos` : 'Sem eventos no período'}
                </span>
              </article>

              <article className="alarmes__metric">
                <span className="alarmes__metric-top">
                  <span>Normalizados</span>
                </span>
                <strong className="alarmes__metric-value">
                  {normalizationRate}<small>%</small>
                </strong>
                <span className="alarmes__metric-description">
                  {averageDuration ? `Duração média de ${averageDuration}s` : `${normalizedInVisible} eventos normalizados`}
                </span>
              </article>
            </section>

            <div className="alarmes__charts">
              <article className="alarmes__chart">
                <header className="alarmes__panel-head">
                  <h2>Alarmes por tipo</h2>
                </header>
                <div className="alarmes__chart-body">
                  {typeRows.length ? (
                    typeRows.map(([label, value]) => (
                      <div key={label} className="alarmes__bar alarmes__bar--type">
                        <strong title={label}>{label}</strong>
                        <span className="alarmes__bar-track">
                          <i style={{ width: `${(value / maxType) * 100}%` }} />
                        </span>
                        <em>{value}</em>
                      </div>
                    ))
                  ) : (
                    <p className="alarmes__empty">Nenhum alarme neste recorte.</p>
                  )}
                </div>
              </article>

              <article className="alarmes__chart">
                <header className="alarmes__panel-head">
                  <h2>Alarmes por estação</h2>
                </header>
                <div className="alarmes__chart-body">
                  {stationRows.length ? (
                    <div className="alarmes__bars alarmes__bars--full">
                      {stationRows.map(([label, value]) => (
                        <div key={label} className="alarmes__bar">
                          <strong title={label}>{label}</strong>
                          <span className="alarmes__bar-track">
                            <i style={{ width: `${(value / maxStation) * 100}%` }} />
                          </span>
                          <em>{value}</em>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="alarmes__empty">Nenhuma estação com alarmes neste recorte.</p>
                  )}
                </div>
              </article>
            </div>
          </div>

          <Filtros
            period={filters.period}
            onPeriod={(period) => {
              setPage(1)
              setFilters((curr) => ({ ...curr, period }))
            }}
            dirty={filtersDirty}
            onClear={() => {
              setPage(1)
              setFilters(emptyFilters)
            }}
          >
            <FiltroCampo label="Busca">
              <input
                type="search"
                value={filters.search}
                onChange={updateFilter('search')}
                placeholder="Buscar por TAG, tipo de alarme, estação..."
              />
            </FiltroCampo>
            <FiltroCampo label="Estação">
              <select value={filters.station} onChange={updateFilter('station')}>
                <option value="">Todas as estações</option>
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
                {platforms.map((p) => (
                  <option key={p} value={p}>
                    {p}
                  </option>
                ))}
              </select>
            </FiltroCampo>
            <FiltroCampo label="Situação">
              <select value={filters.normalization} onChange={updateFilter('normalization')}>
                <option value="">Todas</option>
                <option value="open">Em aberto</option>
                <option value="normalized">Normalizados</option>
              </select>
            </FiltroCampo>
            <FiltroCampo label="Tipo de alarme">
              <select value={filters.alarmType} onChange={updateFilter('alarmType')}>
                <option value="">Todos</option>
                {alarmTypes.map((type) => (
                  <option key={type} value={type}>
                    {type}
                  </option>
                ))}
              </select>
            </FiltroCampo>
          </Filtros>

          <section className="alarmes__list" aria-labelledby="alarm-records-title">
            <header className="alarmes__panel-head alarmes__list-head">
              <h2 id="alarm-records-title">Eventos de telemetria</h2>
              <span className="alarmes__record-count">
                {visible.length.toLocaleString('pt-BR')} {visible.length === 1 ? 'registro' : 'registros'}
              </span>
            </header>

            <div className="alarmes__table-head" aria-hidden="true">
              <span>Identificação</span>
              <span>Horário</span>
              <span>Local</span>
              <span>Alarme</span>
              <span>Situação</span>
              <span>Ação</span>
            </div>
            <div className="alarmes__records">
              {paged.length ? (
                paged.map((item) => (
                  <article
                    key={item.id || `${item.station_code}-${item.event_timestamp}-${item.door_number}`}
                    className="card-alarme"
                  >
                    <div>
                      <span>TAG SAP</span>
                      <strong title={item.tag}>{item.tag}</strong>
                      <em>{item.source}</em>
                    </div>
                    <div>
                      <span>Horário</span>
                      <strong>{item.event_timestamp}</strong>
                    </div>
                    <div>
                      <span>Local</span>
                      <strong>{item.station_code}</strong>
                      <em>{item.station_name} · {item.platform} · P{String(item.door_number || '01').padStart(2, '0')}</em>
                    </div>
                    <div>
                      <span>Tipo de Alarme</span>
                      <strong title={item.alarm_type}>{item.alarm_type}</strong>
                    </div>
                    <div>
                      <span>Situação</span>
                      <strong className={item.normalization_timestamp ? undefined : 'card-alarme__open'}>
                        {item.normalization_timestamp || 'Em aberto'}
                      </strong>
                      <em>{item.duration_seconds}s de duração</em>
                    </div>
                    <div className="card-alarme-actions">
                      <Button
                        variant="secondary"
                        className="btn-alarme-action"
                        onClick={() =>
                          onNavigate?.('portas', {
                            station: item.station_code,
                            platform: item.platform,
                            door: item.door_number,
                          })
                        }
                      >
                        Ver porta
                      </Button>
                    </div>
                  </article>
                ))
              ) : (
                <p className="page-empty">
                  {items.length
                    ? 'Nenhum alarme encontrado com os filtros selecionados.'
                    : 'Nenhum alarme registrado.'}
                </p>
              )}
            </div>

            <Paginacao page={currentPage} pageSize={PAGE_SIZE} total={visible.length} onPage={setPage} />
          </section>
        </>
      )}

      {openModal ? (
        <div className="alarmes-modal" onClick={() => setOpenModal(false)}>
          <form onClick={(e) => e.stopPropagation()} onSubmit={handleImportSpreadsheet}>
            <div className="alarmes-modal-head">
              <h2>Importar alarmes</h2>
              <p className="alarmes-modal-desc">
                Envie um arquivo de planilha no formato CSV ou XLSX contendo os eventos de telemetria da Linha 4.
              </p>
            </div>
            {importError ? <p className="page-status is-error">{importError}</p> : null}
            <PlanilhaImport file={file} onFile={setFile} />
            <UploadStatus upload={upload} />
            <div className="alarmes-modal-actions">
              <Button type="button" variant="secondary" onClick={() => setOpenModal(false)}>
                Cancelar
              </Button>
              <Button type="submit" disabled={!file || importing}>
                {importing ? 'Processando planilha...' : 'Importar Planilha'}
              </Button>
            </div>
          </form>
        </div>
      ) : null}
    </section>
  )
}

export default Alarmes
