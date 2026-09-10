import { useEffect, useMemo, useState } from 'react'
import CadastrarButton from '../components/CadastrarButton'
import Filtros, { FiltroCampo } from '../components/Filtros'
import Paginacao from '../components/Paginacao'
import { usePredictionLoad } from '../components/PredictionLoad'
import { useToast } from '../context/Toast'
import { lineStations, mapStation, stationLabel } from '../data/stations'
import { api, asList, errorMessage } from '../lib/api'
import { exportToCsv } from '../lib/exportCsv'
import { PAGE_SIZE } from '../lib/filters'
import { invalidateInspections } from '../lib/inspectionCache'
import { getCachedPrediction, setCachedPrediction } from '../lib/predictionCache'
import './Predicoes.css'

function formatTag(stationCode, platform, doorNumber) {
  const pad = String(doorNumber || '').padStart(2, '0')
  return `SP04-6-PSD-${stationCode}-${platform || 'P1'}-PP${pad}`
}

function normalizeItem(item) {
  return {
    door_id: item.door_id ?? item.id,
    run_id: item.run_id || '',
    station_code: item.station_code || '',
    station_name: stationLabel(item.station_code) || item.station_name || item.station_code || '',
    platform: item.platform || 'P1',
    door_number: item.door_number,
    sap_tag: item.sap_tag || formatTag(item.station_code, item.platform, item.door_number),
    will_fail: Boolean(item.will_fail),
  }
}

const emptyFilters = {
  station: '',
  platform: '',
  search: '',
}

function Predicoes({ onNavigate, params }) {
  const toast = useToast()
  const { run, busy } = usePredictionLoad()
  const initialCache = getCachedPrediction()
  const [runData, setRunData] = useState(() => initialCache?.runData || null)
  const [items, setItems] = useState(() => initialCache?.items || [])
  const [loading, setLoading] = useState(() => !initialCache?.items?.length)
  const [error, setError] = useState('')
  const [horizonDays, setHorizonDays] = useState(7)
  const [statusTab, setStatusTab] = useState(() => (params?.status || 'all'))
  const [filters, setFilters] = useState(() => ({
    ...emptyFilters,
    station: params?.station || '',
  }))
  const [page, setPage] = useState(1)

  useEffect(() => {
    if (params?.station) {
      setFilters((curr) => ({ ...curr, station: params.station }))
    }
    if (params?.status) {
      setStatusTab(params.status)
    }
  }, [params?.station, params?.status])

  async function load(isBackground = false) {
    if (!isBackground && !getCachedPrediction()?.items?.length) {
      setLoading(true)
    }
    setError('')
    try {
      const response = await api.predictionsLatest()
      if (response && typeof response === 'object') {
        const nextRunData = {
          id: response.id || '',
          horizon_days: response.horizon_days || 7,
          total_doors_evaluated: response.total_doors_evaluated || 0,
          total_doors_at_risk: response.total_doors_at_risk || 0,
          created_at: response.created_at || '',
        }
        const list = asList(response.predictions || response)
        const nextItems = list.map(normalizeItem)
        setRunData(nextRunData)
        setItems(nextItems)
        setCachedPrediction(nextRunData, nextItems)
      }
    } catch (err) {
      if (!getCachedPrediction()?.items?.length) {
        setError(errorMessage(err))
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load(Boolean(initialCache?.items?.length))
  }, [])

  async function handleTriggerPrediction() {
    if (busy) return
    setError('')
    try {
      let result = null
      await run({
        variant: 'overlay',
        mode: 'predict',
        task: async () => {
          result = await api.triggerPredictions(horizonDays)
        },
      })

      invalidateInspections()

      if (result) {
        const nextRunData = {
          id: result.id || '',
          horizon_days: result.horizon_days || horizonDays,
          total_doors_evaluated: result.total_doors_evaluated || 0,
          total_doors_at_risk: result.total_doors_at_risk || 0,
          created_at: result.created_at || '',
        }
        const list = asList(result.predictions || result)
        const nextItems = list.map(normalizeItem)
        setRunData(nextRunData)
        setItems(nextItems)
        setCachedPrediction(nextRunData, nextItems)
        if (nextRunData.total_doors_at_risk > 0) {
          setStatusTab('fail')
        }
      }

      setPage(1)
      toast.success(
        `Predição executada com sucesso! ${result?.total_doors_at_risk || 0} portas em risco de falha.`,
      )
    } catch (err) {
      const msg = errorMessage(err)
      setError(msg)
      toast.error(msg)
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
      if (statusTab === 'fail' && !item.will_fail) return false
      if (statusTab === 'ok' && item.will_fail) return false
      if (filters.station && mapStation(item.station_code) !== mapStation(filters.station)) {
        return false
      }
      if (filters.platform && item.platform !== filters.platform) return false
      if (filters.search.trim()) {
        const q = filters.search.trim().toLowerCase()
        const match =
          item.sap_tag.toLowerCase().includes(q) ||
          item.station_name.toLowerCase().includes(q) ||
          item.station_code.toLowerCase().includes(q) ||
          String(item.door_number).includes(q)
        if (!match) return false
      }
      return true
    })
  }, [items, statusTab, filters])

  const filtersDirty = Boolean(filters.station || filters.platform || filters.search)
  const pageCount = Math.max(1, Math.ceil(visible.length / PAGE_SIZE))
  const currentPage = Math.min(page, pageCount)
  const paged = visible.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE)

  const countRisk = items.filter((i) => i.will_fail).length
  const countOk = items.length - countRisk

  function handleExportCsv() {
    if (!visible.length) {
      toast.info('Não há registros para exportar com os filtros atuais.')
      return
    }
    const columns = [
      { label: 'Tag SAP', key: 'sap_tag' },
      { label: 'Estação', key: 'station_code' },
      { label: 'Nome da Estação', key: 'station_name' },
      { label: 'Plataforma', key: 'platform' },
      { label: 'Número da Porta', key: 'door_number' },
      { label: 'Diagnóstico IA', value: (item) => (item.will_fail ? 'Falha Prevista' : 'Estável') },
      { label: 'Em Risco', value: (item) => (item.will_fail ? 'Sim' : 'Não') },
      { label: 'Rodada ID', value: () => runData?.id || '—' },
      { label: 'Data da Execução', value: () => runData?.created_at || '—' },
      { label: 'Horizonte (dias)', value: () => String(runData?.horizon_days || 7) },
    ]
    exportToCsv('predicoes_linha4.csv', columns, visible)
    toast.success(`${visible.length} predições exportadas com sucesso!`)
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
    <section className="predicoes">
      <div className="page-head predicoes-head">
        <div>
          <p className="dashboard__kicker">Prevenção e Confiabilidade</p>
          <h1>Predições</h1>
          <p className="dashboard__lead">
            Motor de Inteligência Artificial para prevenção de falhas em portas de plataforma.
          </p>
        </div>
        <div className="predicoes-actions">
          <label className="predicoes-horizon">
            <span>Horizonte:</span>
            <select
              value={horizonDays}
              onChange={(e) => setHorizonDays(Number(e.target.value))}
              disabled={busy}
            >
              <option value={7}>7 dias</option>
              <option value={14}>14 dias</option>
              <option value={30}>30 dias</option>
            </select>
          </label>
          <CadastrarButton variant="secondary" onClick={handleExportCsv}>
            Exportar CSV
          </CadastrarButton>
          <button
            type="button"
            className="predicoes-trigger-btn"
            onClick={handleTriggerPrediction}
            disabled={busy}
          >
            <TriggerIcon />
            <span>{busy ? 'Executando...' : 'Executar Predição IA'}</span>
          </button>
        </div>
      </div>

      {loading ? <p className="page-status">Carregando dados da última rodada preditiva…</p> : null}
      {error ? <p className="page-status is-error">{error}</p> : null}

      {runData ? (
        <div className="predicoes-kpis">
          <div className="predicoes-kpi-card">
            <span>Rodada Atual</span>
            <strong title={runData.id}>{runData.id.slice(0, 8) || '—'}</strong>
            <small>{runData.created_at || 'Data não informada'}</small>
          </div>
          <div className="predicoes-kpi-card">
            <span>Portas Avaliadas</span>
            <strong>{runData.total_doors_evaluated || items.length}</strong>
            <small>portas ativas analisadas</small>
          </div>
          <div className="predicoes-kpi-card is-risk">
            <span>Em Risco de Falha</span>
            <strong>{runData.total_doors_at_risk ?? countRisk}</strong>
            <small>requerem manutenção preditiva</small>
          </div>
          <div className="predicoes-kpi-card is-ok">
            <span>Portas Estáveis</span>
            <strong>
              {Math.max(0, (runData.total_doors_evaluated || items.length) - (runData.total_doors_at_risk ?? countRisk))}
            </strong>
            <small>operação regular</small>
          </div>
          <div className="predicoes-kpi-card">
            <span>Horizonte Preditivo</span>
            <strong>{runData.horizon_days || 7} dias</strong>
            <small>janela de antecipação</small>
          </div>
        </div>
      ) : null}

      {runData && (runData.total_doors_at_risk > 0 || countRisk > 0) ? (
        <div className="predicoes-ai-banner">
          <div className="predicoes-ai-banner__text">
            <strong>Inspeções automáticas abertas pelo sistema</strong>
            <span>
              Foram abertas ordens de inspeção de prioridade crítica para todas as portas identificadas
              com risco de quebra na Linha 4 Amarela.
            </span>
          </div>
          <button
            type="button"
            className="predicoes-ai-banner__btn"
            onClick={() => onNavigate?.('inspecoes', { origin: 'ai' })}
          >
            <span>Ver Inspeções Abertas</span>
            <ArrowRightIcon />
          </button>
        </div>
      ) : null}

      <Filtros
        period=""
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
            placeholder="Buscar por TAG SAP, porta, estação..."
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

      <div className="predicoes-tabs" role="tablist">
        <button
          type="button"
          role="tab"
          aria-selected={statusTab === 'all'}
          className={statusTab === 'all' ? 'is-active' : ''}
          onClick={() => {
            setStatusTab('all')
            setPage(1)
          }}
        >
          Todas as Portas ({items.length})
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={statusTab === 'fail'}
          className={`${statusTab === 'fail' ? 'is-active' : ''} ${countRisk > 0 ? 'has-risk' : ''}`}
          onClick={() => {
            setStatusTab('fail')
            setPage(1)
          }}
        >
          Em Risco de Falha ({countRisk})
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={statusTab === 'ok'}
          className={statusTab === 'ok' ? 'is-active' : ''}
          onClick={() => {
            setStatusTab('ok')
            setPage(1)
          }}
        >
          Estáveis ({countOk})
        </button>
      </div>

      <div className="predicoes-list">
        {visible.length ? (
          paged.map((item) => (
            <article
              key={`${item.station_code}-${item.platform}-${item.door_number}`}
              className={`card-predicao${item.will_fail ? ' is-fail' : ''}`}
            >
              <div>
                <span>TAG SAP</span>
                <strong>{item.sap_tag}</strong>
              </div>
              <div>
                <span>Estação</span>
                <strong>{item.station_code}</strong>
                <em>{item.station_name}</em>
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
                <span>Diagnóstico</span>
                <span className={`badge-status ${item.will_fail ? 'is-fail' : 'is-ok'}`}>
                  {item.will_fail ? 'Falha Prevista' : 'Estável'}
                </span>
              </div>
              <div className="card-predicao-actions">
                {item.will_fail ? (
                  <button
                    type="button"
                    className="btn-predicao-action is-inspection"
                    onClick={() =>
                      onNavigate?.('inspecoes', {
                        station: item.station_code,
                        door: item.door_number,
                        origin: 'ai',
                      })
                    }
                  >
                    Ver Inspeção Aberta
                  </button>
                ) : null}
                <button
                  type="button"
                  className="btn-predicao-action"
                  onClick={() =>
                    onNavigate?.('portas', {
                      station: item.station_code,
                      platform: item.platform,
                      door: item.door_number,
                    })
                  }
                >
                  Dossiê da Porta
                </button>
              </div>
            </article>
          ))
        ) : (
          <p className="page-empty">
            {items.length
              ? 'Nenhuma porta encontrada com os filtros selecionados.'
              : 'Nenhuma predição executada até o momento. Clique em "Executar Predição IA" para iniciar a inferência.'}
          </p>
        )}
        <Paginacao page={currentPage} pageSize={PAGE_SIZE} total={visible.length} onPage={setPage} />
      </div>
    </section>
  )
}

function TriggerIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" width="18" height="18">
      <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" />
    </svg>
  )
}

function ArrowRightIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" width="16" height="16">
      <path d="M5 12h14M12 5l7 7-7 7" />
    </svg>
  )
}

export default Predicoes
