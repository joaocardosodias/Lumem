import { useEffect, useMemo, useState } from 'react'
import Button from '../components/Button'
import CadastrarButton from '../components/CadastrarButton'
import Filtros, { FiltroCampo } from '../components/Filtros'
import Paginacao from '../components/Paginacao'
import PlanilhaImport from '../components/PlanilhaImport'
import UploadStatus from '../components/UploadStatus'
import { useToast } from '../context/Toast'
import { lineStations, mapStation, stationLabel } from '../data/stations'
import { api, asList, errorMessage } from '../lib/api'
import { exportToCsv } from '../lib/exportCsv'
import { DEFAULT_PERIOD, PAGE_SIZE, inPeriod } from '../lib/filters'
import { pollUpload } from '../lib/runUpload'
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
    source: item.source || 'telemetry',
  }
}

const emptyFilters = {
  station: '',
  platform: '',
  period: DEFAULT_PERIOD,
  search: '',
}

function Alarmes({ onNavigate, params }) {
  const toast = useToast()
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(true)
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

  async function load() {
    setLoading(true)
    setError('')
    try {
      const response = await api.alarms({ limit: 500 })
      const list = asList(response?.data || response?.alarms || response)
      setItems(list.map(mapAlarm))
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
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

  const visible = useMemo(() => {
    return items.filter((item) => {
      if (filters.station && mapStation(item.station_code) !== mapStation(filters.station)) {
        return false
      }
      if (filters.platform && item.platform !== filters.platform) return false
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
    Boolean(filters.station || filters.platform || filters.search) ||
    filters.period !== DEFAULT_PERIOD

  const pageCount = Math.max(1, Math.ceil(visible.length / PAGE_SIZE))
  const currentPage = Math.min(page, pageCount)
  const paged = visible.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE)

  const activeAlarms = items.filter((i) => !i.normalization_timestamp).length
  const normalizedAlarms = items.length - activeAlarms

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
      <div className="page-head alarmes-head">
        <div>
          <p className="dashboard__kicker">Telemetria de Campo</p>
          <h1>Alarmes</h1>
          <p className="dashboard__lead">
            Monitoramento e telemetria de alarmes em tempo real das portas da Linha 4 Amarela.
          </p>
        </div>
        <div className="alarmes-actions">
          <CadastrarButton variant="secondary" onClick={handleExportCsv}>
            Exportar CSV
          </CadastrarButton>
          <CadastrarButton
            onClick={() => {
              setImportError('')
              setFile(null)
              setUpload(null)
              setOpenModal(true)
            }}
          >
            Cadastrar Alarmes +
          </CadastrarButton>
        </div>
      </div>

      {loading ? <p className="page-status">Carregando alarmes do banco…</p> : null}
      {error ? <p className="page-status is-error">{error}</p> : null}

      <div className="alarmes-kpis">
        <div className="alarmes-kpi-card">
          <span>Total de Alarmes</span>
          <strong>{items.length}</strong>
          <small>registrados no banco de dados</small>
        </div>
        <div className="alarmes-kpi-card">
          <span>Alarmes em Aberto</span>
          <strong>{activeAlarms}</strong>
          <small>sem normalização registrada</small>
        </div>
        <div className="alarmes-kpi-card">
          <span>Normalizados</span>
          <strong>{normalizedAlarms}</strong>
          <small>eventos restabelecidos</small>
        </div>
        <div className="alarmes-kpi-card">
          <span>Filtro Ativo</span>
          <strong>{visible.length}</strong>
          <small>alarmes visíveis na listagem</small>
        </div>
      </div>

      <Filtros
        period={filters.period}
        onPeriod={(period) => {
          setFilters((curr) => ({ ...curr, period }))
          setPage(1)
        }}
        dirty={filtersDirty}
        onClear={() => {
          setFilters(emptyFilters)
          setPage(1)
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
      </Filtros>

      <div className="alarmes-list">
        {visible.length ? (
          paged.map((item) => (
            <article key={item.id || `${item.station_code}-${item.event_timestamp}-${item.door_number}`} className="card-alarme">
              <div>
                <span>TAG SAP</span>
                <strong>{item.tag}</strong>
              </div>
              <div>
                <span>Horário</span>
                <strong>{item.event_timestamp}</strong>
              </div>
              <div>
                <span>Estação</span>
                <strong>{item.station_code}</strong>
                <em>{item.station_name}</em>
              </div>
              <div>
                <span>Plat / Porta</span>
                <strong>{item.platform} · P{String(item.door_number || '01').padStart(2, '0')}</strong>
              </div>
              <div>
                <span>Tipo de Alarme</span>
                <strong>{item.alarm_type}</strong>
              </div>
              <div>
                <span>Duração</span>
                <strong>{item.duration_seconds}s</strong>
              </div>
              <div>
                <span>Normalização</span>
                <strong>{item.normalization_timestamp || 'Em aberto'}</strong>
              </div>
              <div className="card-alarme-actions">
                <button
                  type="button"
                  className="btn-alarme-action"
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
            {items.length
              ? 'Nenhum alarme encontrado com os filtros selecionados.'
              : 'Nenhum alarme registrado no banco de dados.'}
          </p>
        )}
        <Paginacao page={currentPage} pageSize={PAGE_SIZE} total={visible.length} onPage={setPage} />
      </div>

      {openModal ? (
        <div className="alarmes-modal" onClick={() => setOpenModal(false)}>
          <form onClick={(e) => e.stopPropagation()} onSubmit={handleImportSpreadsheet}>
            <h2>Cadastrar Alarmes por Planilha</h2>
            <p className="alarmes-modal-desc">
              Envie um arquivo de planilha no formato CSV ou XLSX contendo os eventos de alarme de telemetria.
            </p>
            {importError ? <p className="page-status is-error">{importError}</p> : null}
            <PlanilhaImport file={file} onFile={setFile} />
            <UploadStatus upload={upload} />
            <div className="alarmes-modal-actions">
              <Button type="button" variant="ghost" onClick={() => setOpenModal(false)}>
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
