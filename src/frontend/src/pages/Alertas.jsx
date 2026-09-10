import { useEffect, useMemo, useState } from 'react'
import Button from '../components/Button'
import CadastrarButton from '../components/CadastrarButton'
import Card from '../components/Card'
import CardAlerta from '../components/CardAlerta'
import Filtros, { FiltroCampo } from '../components/Filtros'
import Paginacao from '../components/Paginacao'
import PlanilhaImport from '../components/PlanilhaImport'
import UploadStatus from '../components/UploadStatus'
import { useToast } from '../context/Toast'
import { lineStations, stationLabel } from '../data/stations'
import { DEFAULT_PERIOD, PAGE_SIZE, inPeriod, periodHint } from '../lib/filters'
import { api, asList, errorMessage } from '../lib/api'
import { exportToCsv } from '../lib/exportCsv'
import { pollUpload } from '../lib/runUpload'
import './Alertas.css'

function countBy(items, key) {
  return items.reduce((acc, item) => {
    const value = item[key] || '—'
    acc[value] = (acc[value] || 0) + 1
    return acc
  }, {})
}

function mapPrediction(item) {
  return {
    tag: item.sap_tag || '—',
    started_at: item.last_evaluated_at || '—',
    station: item.station_name || stationLabel(item.station_code) || item.station_code || '—',
    station_code: item.station_code || '',
    door_number: item.door_number,
    platform: item.platform || '',
    alarm: item.will_fail ? 'Vai falhar' : 'Estável',
    will_fail: Boolean(item.will_fail),
    normalized_at: [item.platform, item.door_number != null ? String(item.door_number).padStart(2, '0') : '']
      .filter(Boolean)
      .join(' · ') || '—',
    door_id: item.door_id,
  }
}

const emptyFilters = {
  station: '',
  diagnosis: '',
  platform: '',
  period: DEFAULT_PERIOD,
}

function Alertas({ onNavigate, params }) {
  const toast = useToast()
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [open, setOpen] = useState(false)
  const [file, setFile] = useState(null)
  const [importError, setImportError] = useState('')
  const [importing, setImporting] = useState(false)
  const [upload, setUpload] = useState(null)
  const [search, setSearch] = useState('')
  const [filters, setFilters] = useState(() => ({
    ...emptyFilters,
    station: params?.station || '',
  }))
  const [page, setPage] = useState(1)

  async function load() {
    setLoading(true)
    setError('')
    try {
      setItems(asList(await api.predictionsLatest()).map(mapPrediction))
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  const visible = useMemo(
    () =>
      items.filter((item) => {
        if (
          filters.station &&
          String(item.station_code || '').toUpperCase() !== filters.station.toUpperCase()
        ) {
          return false
        }
        if (filters.diagnosis === 'fail' && !item.will_fail) return false
        if (filters.diagnosis === 'ok' && item.will_fail) return false
        if (filters.platform && item.platform !== filters.platform) return false
        if (!inPeriod(item.started_at, filters.period)) return false
        if (search.trim()) {
          const q = search.trim().toLowerCase()
          const match =
            item.tag.toLowerCase().includes(q) ||
            item.station.toLowerCase().includes(q) ||
            item.alarm.toLowerCase().includes(q) ||
            item.normalized_at.toLowerCase().includes(q)
          if (!match) return false
        }
        return true
      }),
    [items, filters, search],
  )
  const filtersDirty =
    Boolean(filters.station || filters.diagnosis || filters.platform || search) ||
    filters.period !== DEFAULT_PERIOD
  const pageCount = Math.max(1, Math.ceil(visible.length / PAGE_SIZE))
  const currentPage = Math.min(page, pageCount)
  const paged = visible.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE)
  const rangeHint = periodHint(filters.period)
  const platforms = useMemo(() => {
    if (filters.station) {
      return lineStations.find((station) => station.code === filters.station)?.platforms ?? []
    }
    return [...new Set(items.map((item) => item.platform).filter(Boolean))]
  }, [items, filters.station])
  const byType = useMemo(() => countBy(visible, 'alarm'), [visible])
  const byStation = useMemo(() => countBy(visible, 'station'), [visible])
  const maxType = Math.max(...Object.values(byType), 1)
  const maxStation = Math.max(...Object.values(byStation), 1)
  const atRisk = visible.filter((item) => item.will_fail).length

  async function handleImport(event) {
    event.preventDefault()
    if (!file) {
      setImportError('Selecione um arquivo CSV ou XLSX.')
      return
    }

    setImporting(true)
    setImportError('')

    try {
      const result = await api.uploadAlarms(file)
      const done = await pollUpload(result, setUpload)
      if (done.status === 2) {
        throw new Error('A API recusou o processamento deste arquivo.')
      }
      setFile(null)
      setUpload(null)
      setOpen(false)
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

  function handleExportCsv() {
    if (!visible.length) {
      toast.info('Não há registros para exportar com os filtros atuais.')
      return
    }
    const columns = [
      { label: 'Tag SAP', key: 'tag' },
      { label: 'Data e Hora', key: 'started_at' },
      { label: 'Estação', key: 'station' },
      { label: 'Código Estação', key: 'station_code' },
      { label: 'Plataforma / Porta', key: 'normalized_at' },
      { label: 'Diagnóstico', key: 'alarm' },
      { label: 'Risco de Falha', value: (item) => (item.will_fail ? 'Sim' : 'Não') },
    ]
    exportToCsv('alertas_preditivos.csv', columns, visible)
    toast.success(`${visible.length} registros exportados com sucesso!`)
  }

  function openCadastro() {
    setImportError('')
    setFile(null)
    setUpload(null)
    setOpen(true)
  }

  function updateFilter(field) {
    return (event) =>
      setFilters((current) => {
        const next = { ...current, [field]: event.target.value }
        if (field === 'station') next.platform = ''
        setPage(1)
        return next
      })
  }

  return (
    <section className="alertas">
      <div className="page-head">
        <h1>Alertas</h1>
        <div className="page-head__actions">
          <CadastrarButton variant="secondary" onClick={handleExportCsv}>
            Exportar CSV
          </CadastrarButton>
          <CadastrarButton onClick={openCadastro}>Cadastrar Alarmes +</CadastrarButton>
        </div>
      </div>

      {loading ? <p className="page-status">Carregando alertas da última inferência…</p> : null}
      {error ? <p className="page-status is-error">{error}</p> : null}

      <div className="alertas__hero">
        <Card
          layout="banner"
          label="Alertas preditivos"
          value={atRisk}
          hint={
            visible.length
              ? `${visible.length} portas na última rodada da IA${rangeHint ? ` · ${rangeHint}` : ''}`
              : 'Nenhum alerta na última rodada'
          }
        />
        <div className="alertas__charts">
          <article className="alertas__chart">
            <h2>Alertas por diagnóstico</h2>
            {Object.entries(byType).map(([label, value]) => (
              <div key={label} className="alertas__bar">
                <strong>{label}</strong>
                <i style={{ width: `${(value / maxType) * 100}%` }} />
                <em>{value}</em>
              </div>
            ))}
          </article>
          <article className="alertas__chart">
            <h2>Alertas por estação</h2>
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

      <Filtros
        period={filters.period}
        onPeriod={(period) => {
          setPage(1)
          setFilters((current) => ({ ...current, period }))
        }}
        dirty={filtersDirty}
        onClear={() => {
          setPage(1)
          setSearch('')
          setFilters(emptyFilters)
        }}
      >
        <FiltroCampo label="Busca">
          <input
            type="search"
            value={search}
            onChange={(event) => {
              setSearch(event.target.value)
              setPage(1)
            }}
            placeholder="Filtrar por TAG, porta..."
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
        <FiltroCampo label="Diagnóstico">
          <select value={filters.diagnosis} onChange={updateFilter('diagnosis')}>
            <option value="">Todos</option>
            <option value="fail">Vai falhar</option>
            <option value="ok">Estável</option>
          </select>
        </FiltroCampo>
        <FiltroCampo label="Plataforma">
          <select value={filters.platform} onChange={updateFilter('platform')}>
            <option value="">Todas</option>
            {platforms.map((platform) => (
              <option key={platform} value={platform}>
                {platform}
              </option>
            ))}
          </select>
        </FiltroCampo>
      </Filtros>

      <div className="alertas__list">
        {paged.length ? (
          paged.map((alarm) => (
            <CardAlerta
              key={`${alarm.tag}-${alarm.door_id}-${alarm.started_at}`}
              tag={alarm.tag}
              startedAt={alarm.started_at}
              station={alarm.station}
              alarm={alarm.alarm}
              normalizedAt={alarm.normalized_at}
              willFail={alarm.will_fail}
              onViewDoor={() =>
                onNavigate &&
                onNavigate('portas', {
                  station: alarm.station_code,
                  door: alarm.door_number,
                })
              }
              onOpenInspection={() =>
                onNavigate &&
                onNavigate('inspecoes', {
                  station: alarm.station_code,
                  door: alarm.door_number,
                })
              }
            />
          ))
        ) : (
          <p className="page-empty">
            {items.length ? 'Nenhum alerta com esses filtros.' : 'Nenhum alerta na última inferência.'}
          </p>
        )}
        <Paginacao page={currentPage} pageSize={PAGE_SIZE} total={visible.length} onPage={setPage} />
      </div>

      {open ? (
        <div className="alertas-modal" onClick={() => !importing && setOpen(false)}>
          <form onClick={(event) => event.stopPropagation()} onSubmit={handleImport}>
            <h2>Enviar planilha de alarmes</h2>
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
          </form>
        </div>
      ) : null}
    </section>
  )
}

export default Alertas
