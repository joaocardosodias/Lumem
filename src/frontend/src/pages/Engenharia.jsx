import { useEffect, useState } from 'react'
import Button from '../components/Button'
import { useAuth } from '../context/Auth'
import { useToast } from '../context/Toast'
import { api, errorMessage } from '../lib/api'
import { invalidateApplicationCache } from '../lib/appPrefetch'
import { ThresholdComparisonCharts } from '../components/ThresholdComparisonCharts'
import './Engenharia.css'

function TriggerIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" width="16" height="16" aria-hidden="true">
      <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" />
    </svg>
  )
}

const engineeringTabs = [
  { id: 'model', label: 'Modelo e métricas' },
  { id: 'calibration', label: 'Laboratório' },
  { id: 'retrain', label: 'Ciclo de retreino' },
  { id: 'drift', label: 'Qualidade dos dados' },
]

function trainingStatusLabel(status) {
  const labels = {
    completed: 'Concluído',
    failed: 'Falhou',
    production: 'Produção',
    running: 'Em execução',
    queued: 'Na fila',
    candidate: 'Candidato',
    retired: 'Arquivado',
  }

  return labels[status] || status || 'Desconhecido'
}

function percent(value) {
  return Number.isFinite(Number(value)) ? `${(Number(value) * 100).toFixed(1)}%` : 'Indisponível'
}

function decimal(value, digits = 3) {
  return Number.isFinite(Number(value)) ? Number(value).toFixed(digits) : 'Indisponível'
}

function integer(value) {
  return Number.isFinite(Number(value)) ? Number(value).toLocaleString('pt-BR') : '—'
}

function nearestThresholdMetric(metrics, threshold) {
  if (!Array.isArray(metrics) || metrics.length === 0) return null
  return metrics.reduce((nearest, item) => (
    Math.abs(Number(item.threshold) - threshold) < Math.abs(Number(nearest.threshold) - threshold) ? item : nearest
  ))
}

function thresholdProfile(value) {
  const threshold = Number(value)
  if (threshold < 0.4) return { label: 'Mais sensível', detail: 'captura mais falhas e amplia o volume de inspeções' }
  if (threshold > 0.6) return { label: 'Mais seletivo', detail: 'reduz alertas e prioriza os casos de maior confiança' }
  return { label: 'Faixa equilibrada', detail: 'equilibra cobertura de falhas e esforço operacional' }
}

function EngineeringLoading() {
  return (
    <div className="engenharia-loading" role="status" aria-label="Carregando métricas e telemetria do modelo ativo">
      <span className="engenharia-visually-hidden">Carregando métricas e telemetria do modelo ativo…</span>
      <div className="engenharia-loading__metrics" aria-hidden="true">
        {Array.from({ length: 5 }, (_, index) => <span key={index} />)}
      </div>
      <div className="engenharia-loading__panels" aria-hidden="true"><span /><span /></div>
    </div>
  )
}

function Engenharia() {
  const { user } = useAuth()
  const toast = useToast()

  const [activeTab, setActiveTab] = useState('model')
  const [loading, setLoading] = useState(true)
  const [runningExperiment, setRunningExperiment] = useState(false)
  const [savingConfig, setSavingConfig] = useState(false)
  const [promotingExperiment, setPromotingExperiment] = useState('')
  const [retraining, setRetraining] = useState(false)
  const [trainingJob, setTrainingJob] = useState(null)
  const [trainingHistory, setTrainingHistory] = useState([])
  const [promotingModel, setPromotingModel] = useState('')

  const [config, setConfig] = useState({
    threshold: 0.5,
    default_horizon_days: 7,
    auto_retrain_enabled: false,
    updated_at: '',
  })
  const [model, setModel] = useState(null)
  const [history, setHistory] = useState([])
  const [drift, setDrift] = useState(null)
  const [experiments, setExperiments] = useState([])
  const [officialRun, setOfficialRun] = useState(null)
  const [productionPerformance, setProductionPerformance] = useState(null)

  const [selectedThreshold, setSelectedThreshold] = useState(0.5)
  const selectedProfile = thresholdProfile(selectedThreshold)

  useEffect(() => {
    async function loadData() {
      try {
        setLoading(true)
        const [cfgData, modelData, histData, driftData, experimentData, officialData, performanceData, trainingData] = await Promise.all([
          api.getMLConfig().catch(() => ({ threshold: 0.5, default_horizon_days: 7 })),
          api.getActiveModel().catch(() => null),
          api.getModelHistory(5).catch(() => []),
          api.getMLDrift().catch(() => null),
          api.predictionExperiments(20).catch(() => []),
          api.predictionsLatest().catch(() => null),
          api.getMLProductionPerformance().catch(() => null),
          api.getTrainingJobs(20).catch(() => []),
        ])
        setConfig(cfgData)
        setSelectedThreshold(cfgData.threshold ?? 0.5)
        setModel(modelData)
        setHistory(histData || [])
        setDrift(driftData)
        setExperiments(experimentData || [])
        setOfficialRun(officialData)
        setProductionPerformance(performanceData)
        setTrainingHistory(trainingData || [])
      } finally {
        setLoading(false)
      }
    }
    loadData()
  }, [])

  async function handleSaveThreshold() {
    try {
      setSavingConfig(true)
      const updatedConfig = await api.updateMLConfig(selectedThreshold, 7)
      setConfig((current) => ({
        ...current,
        threshold: updatedConfig.threshold ?? selectedThreshold,
        default_horizon_days: updatedConfig.default_horizon_days ?? 7,
        updated_at: updatedConfig.updated_at ?? new Date().toISOString(),
      }))

      let newRun = null
      try {
        newRun = await api.triggerPredictions(7)
        if (newRun) {
          setOfficialRun(newRun)
        }
      } catch {
        newRun = null
      }

      invalidateApplicationCache()
      if (newRun) {
        toast.success(`Limiar ${Number(selectedThreshold).toFixed(2)} salvo e aplicado à operação com sucesso.`)
      } else {
        toast.success(`Limiar ${Number(selectedThreshold).toFixed(2)} salvo nas configurações.`)
      }
    } catch (err) {
      toast.error('Falha ao salvar limiar: ' + errorMessage(err))
    } finally {
      setSavingConfig(false)
    }
  }

  async function handleRunExperiment(e) {
    e.preventDefault()
    try {
      setRunningExperiment(true)
      const result = await api.runPredictionExperiment(selectedThreshold)
      setExperiments((current) => [result, ...current])
      toast.success('Cenário concluído e salvo no laboratório.')
    } catch (err) {
      toast.error('Falha ao executar cenário: ' + errorMessage(err))
    } finally {
      setRunningExperiment(false)
    }
  }

  async function handlePromoteExperiment(run) {
    try {
      setPromotingExperiment(run.id)
      const promoted = await api.promotePredictionExperiment(run.id)
      setOfficialRun(promoted)
      setConfig((current) => ({ ...current, threshold: promoted.threshold_used, default_horizon_days: 7 }))
      setSelectedThreshold(promoted.threshold_used)
      setExperiments((current) => current.filter((item) => item.id !== run.id))
      invalidateApplicationCache()
      toast.success('Cenário adotado como predição oficial.')
    } catch (err) {
      toast.error('Falha ao publicar cenário: ' + errorMessage(err))
    } finally {
      setPromotingExperiment('')
    }
  }

  async function handleTriggerRetrain() {
    try {
      setRetraining(true)
      let job = await api.triggerRetrain()
      setTrainingJob(job)
      setTrainingHistory((current) => [job, ...current.filter((item) => item.id !== job.id)])
      toast.success('Retreinamento enfileirado. O status será atualizado nesta tela.')
      for (let attempt = 0; attempt < 120 && !['completed', 'failed'].includes(job.status); attempt += 1) {
        await new Promise((resolve) => window.setTimeout(resolve, 3000))
        job = await api.getTrainingJob(job.id)
        setTrainingJob(job)
        setTrainingHistory((current) => current.map((item) => item.id === job.id ? job : item))
      }
      if (job.status === 'completed') {
        const [models, active, jobs] = await Promise.all([api.getModelHistory(20), api.getActiveModel(), api.getTrainingJobs(20)])
        setHistory(models || [])
        setModel(active)
        setTrainingHistory(jobs || [])
        toast.success('Retreinamento concluído. A nova versão está disponível como candidata.')
      } else if (job.status === 'failed') {
        toast.error(`Retreinamento falhou: ${job.error || 'consulte o log do job.'}`)
      }
    } catch (err) {
      toast.error('Erro no retreinamento: ' + errorMessage(err))
    } finally {
      setRetraining(false)
    }
  }

  async function handlePromoteModel(version) {
    try {
      setPromotingModel(version)
      const active = await api.promoteModel(version)
      const models = await api.getModelHistory(20)
      setModel(active)
      setHistory(models || [])
      toast.success(`Modelo ${version} promovido para produção.`)
    } catch (err) {
      toast.error('Falha ao promover modelo: ' + errorMessage(err))
    } finally {
      setPromotingModel('')
    }
  }

  function handleTabKeyDown(event) {
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
    event.preventDefault()
    const currentIndex = engineeringTabs.findIndex((tab) => tab.id === activeTab)
    let nextIndex = currentIndex
    if (event.key === 'ArrowLeft') nextIndex = (currentIndex - 1 + engineeringTabs.length) % engineeringTabs.length
    if (event.key === 'ArrowRight') nextIndex = (currentIndex + 1) % engineeringTabs.length
    if (event.key === 'Home') nextIndex = 0
    if (event.key === 'End') nextIndex = engineeringTabs.length - 1
    const nextTab = engineeringTabs[nextIndex].id
    setActiveTab(nextTab)
    requestAnimationFrame(() => document.getElementById(`engineering-tab-${nextTab}`)?.focus())
  }

  const activeThreshold = Number(config?.threshold ?? model?.recommended_threshold ?? 0.5)
  const thresholdMetrics = model?.threshold_metrics || []
  const activeMetric = nearestThresholdMetric(thresholdMetrics, activeThreshold)
  const confusion = activeMetric || {}
  const testMetrics = model?.test_metrics || {}
  const testRows = Number(model?.test_rows) || 0
  const positiveRows = Number(model?.positive_rows) || 0
  const negativeRows = Math.max(0, testRows - positiveRows)
  const featureImportance = Array.isArray(model?.feature_importance) ? model.feature_importance.slice(0, 8) : []
  const maxImportance = Math.max(0, ...featureImportance.map((item) => Number(item.importance) || 0))

  if (user?.role !== 'enginner') {
    return (
      <section className="engenharia">
        <div className="page-head engenharia-head">
          <div>
            <p className="overview-eyebrow"><span aria-hidden="true" />Linha 4 Amarela · MLOps e governança</p>
            <h1>Engenharia de dados</h1>
          </div>
        </div>
        <div className="engenharia-access-denied">
          <h2>Acesso restrito</h2>
          <p>O módulo de Engenharia de Dados e MLOps está disponível exclusivamente para a equipe de engenharia.</p>
        </div>
      </section>
    )
  }

  return (
    <section className="engenharia">
      <div className="page-head engenharia-head">
        <div>
          <p className="overview-eyebrow"><span aria-hidden="true" />Linha 4 Amarela · MLOps e governança</p>
          <h1>Engenharia de dados</h1>
          <p className="dashboard__lead">
            Supervisão do ciclo de vida dos modelos preditivos, calibração do limiar operacional e monitoramento contínuo de telemetria.
          </p>
        </div>
        <div className="page-head__actions">
          <div className="engenharia-model-state">
            <span className="engenharia-status-dot" aria-hidden="true" />
            <span><small>Modelo em produção</small><strong>{model?.version || 'Indisponível'}</strong></span>
          </div>
          <div className="engenharia-model-state">
            <span className="engenharia-status-dot" aria-hidden="true" />
            <span><small>Limiar operacional</small><strong>{Number(config?.threshold ?? 0.5).toFixed(2)}</strong></span>
          </div>
        </div>
      </div>

      <nav className="engenharia-tabs" role="tablist" aria-label="Áreas de engenharia" onKeyDown={handleTabKeyDown}>
        {engineeringTabs.map((tab) => (
          <button
            id={`engineering-tab-${tab.id}`}
            key={tab.id}
            type="button"
            role="tab"
            aria-controls={`engineering-panel-${tab.id}`}
            aria-selected={activeTab === tab.id}
            tabIndex={activeTab === tab.id ? 0 : -1}
            className={activeTab === tab.id ? 'is-active' : ''}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </nav>

      {loading ? <EngineeringLoading /> : null}

      {!loading && activeTab === 'model' ? (
        <div className="engenharia-tab-panel" id="engineering-panel-model" role="tabpanel" aria-labelledby="engineering-tab-model">
          <div className="engenharia-benchmark-banner">
            <div>
              <span className="lab-eyebrow">Benchmark de teste · Avaliação walk-forward</span>
              <p>
                Métricas históricas do artefato <strong>{model?.version || 'indisponível'}</strong>, calculadas em {integer(testRows)} registros do conjunto temporal de teste. O limiar ativo na operação é <strong>{activeThreshold.toFixed(2)}</strong>.
              </p>
            </div>
          </div>

          <div className="engenharia-metrics">
            <div className="engenharia-metric engenharia-metric--version">
              <span>Modelo ativo</span>
              <strong>{model?.version || 'Indisponível'}</strong>
              <small>{model?.algorithm || 'Manifest não carregado'}</small>
            </div>
            <div className="engenharia-metric">
              <span>Recall de falhas</span>
              <strong>{percent(activeMetric?.recall)}</strong>
              <small>{integer(confusion.true_positives)} de {integer(positiveRows)} falhas · threshold {activeMetric ? Number(activeMetric.threshold).toFixed(2) : '—'}</small>
            </div>
            <div className="engenharia-metric">
              <span>ROC-AUC</span>
              <strong>{decimal(testMetrics.roc_auc)}</strong>
              <small>Separação entre classes</small>
            </div>
            <div className="engenharia-metric">
              <span>F1-Score</span>
              <strong>{percent(activeMetric?.f1_score)}</strong>
              <small>Validação histórica versionada</small>
            </div>
            <div className="engenharia-metric">
              <span>Precisão</span>
              <strong>{percent(activeMetric?.precision)}</strong>
              <small>{integer(confusion.true_positives)} acertos em {integer(confusion.alert_count)} alertas</small>
            </div>
          </div>

          <div className="engenharia-panels">
            <article className="engenharia-card">
              <header className="engenharia-card__head">
                <h2>Matriz de confusão · teste walk-forward</h2>
                <span className="engenharia-tag">Threshold {activeMetric ? Number(activeMetric.threshold).toFixed(2) : 'indisponível'} · {integer(testRows)} registros</span>
              </header>

              <div className="engenharia-card__body">
                <div className="confusion-matrix-wrap">
                  <div className="confusion-box is-title">
                    <span>Real / previsto</span>
                  </div>
                  <div className="confusion-box is-title">
                    <span>Previsto: falha</span>
                  </div>
                  <div className="confusion-box is-title">
                    <span>Previsto: normal</span>
                  </div>

                  <div className="confusion-box is-title">
                    <span>Falha real ({integer(positiveRows)})</span>
                  </div>
                  <div className="confusion-box is-tp">
                    <strong>{integer(confusion.true_positives)}</strong>
                    <small>Verdadeiros positivos (TP)</small>
                  </div>
                  <div className="confusion-box is-fn">
                    <strong>{integer(confusion.false_negatives)}</strong>
                    <small>Falsos negativos (FN)</small>
                  </div>

                  <div className="confusion-box is-title">
                    <span>Normal real ({integer(negativeRows)})</span>
                  </div>
                  <div className="confusion-box is-fp">
                    <strong>{integer(confusion.false_positives)}</strong>
                    <small>Falsos positivos (FP)</small>
                  </div>
                  <div className="confusion-box is-tn">
                    <strong>{integer(confusion.true_negatives)}</strong>
                    <small>Verdadeiros negativos (TN)</small>
                  </div>
                </div>
              </div>

              <footer className="engenharia-card__footer">
                <span>Período de teste: {model?.test_start ? new Date(model.test_start).toLocaleDateString('pt-BR') : '—'} a {model?.test_end ? new Date(model.test_end).toLocaleDateString('pt-BR') : '—'}.</span>
                <span>Dataset {model?.dataset_sha256 ? `${model.dataset_sha256.slice(0, 12)}…` : 'sem hash legado'}</span>
              </footer>
            </article>

            <article className="engenharia-card">
              <header className="engenharia-card__head">
                <h2>Atributos com maior importância</h2>
                <span className="engenharia-tag">{model?.algorithm || 'Manifest'}</span>
              </header>

              <div className="engenharia-card__body">
                <div className="feature-bars">
                  {featureImportance.map((item) => (
                    <div key={item.feature} className="feature-bar-row">
                      <div className="feature-bar-row__top">
                        <span className="feature-bar-row__tag">{item.feature}</span>
                        <div>
                          <span className="feature-bar-row__category">{item.group || 'Feature do modelo'}</span>
                          <span className="feature-bar-row__score">{percent(item.importance)}</span>
                        </div>
                      </div>
                      <div className="feature-bar-track">
                        <div className="feature-bar-fill" style={{ width: `${maxImportance ? (Number(item.importance) / maxImportance) * 100 : 0}%` }} />
                      </div>
                    </div>
                  ))}
                  {!featureImportance.length ? <p className="lab-empty">Importância das features indisponível no manifest deste modelo.</p> : null}
                </div>
              </div>

              <footer className="engenharia-card__footer">
                <span>Importâncias registradas no manifest do mesmo artefato exibido acima.</span>
                <span>{model?.manifest?.features?.length ?? model?.manifest?.features_count ?? '—'} features mantidas</span>
              </footer>
            </article>
          </div>
        </div>
      ) : null}

      {!loading && activeTab === 'calibration' ? (
        <div className="engenharia-lab" id="engineering-panel-calibration" role="tabpanel" aria-labelledby="engineering-tab-calibration">
          <section className="lab-official" aria-labelledby="official-config-title">
            <div>
              <span className="lab-eyebrow"><i aria-hidden="true" />Configuração oficial</span>
              <h2 id="official-config-title">Diagnóstico exibido na operação</h2>
              <p>Esta é a rodada que alimenta Dashboard, Predições e a abertura de inspeções preventivas.</p>
            </div>
            <dl>
              <div><dt>Modelo</dt><dd>{officialRun?.model_version || model?.version || 'Indisponível'}</dd></div>
              <div><dt>Threshold</dt><dd>{Number(officialRun?.threshold_used ?? config.threshold ?? 0.5).toFixed(2)}</dd></div>
              <div><dt>Portas em risco</dt><dd>{officialRun?.total_doors_at_risk ?? '—'}</dd></div>
              <div><dt>Última publicação</dt><dd className="lab-official__date">{officialRun?.published_at ? new Date(officialRun.published_at).toLocaleDateString('pt-BR') : '—'}</dd></div>
            </dl>
          </section>

          <section className="lab-workbench">
            <form className="lab-controls" onSubmit={handleRunExperiment}>
              <div className="lab-controls__intro">
                <span className="lab-eyebrow">Novo cenário</span>
                <h2>Defina o nível de sensibilidade</h2>
                <p>Simule primeiro. O resultado fica isolado até que você decida adotá-lo na operação.</p>
              </div>
              <div className="lab-control-grid">
                <div className="lab-threshold-control">
                  <div className="calibration-field__top">
                    <div>
                      <label htmlFor="threshold-slider">Threshold</label>
                      <span className="lab-threshold-profile"><strong>{selectedProfile.label}</strong> · {selectedProfile.detail}</span>
                    </div>
                    <output className="calibration-value-badge" htmlFor="threshold-slider">{Number(selectedThreshold).toFixed(2)}</output>
                  </div>
                  <input
                    id="threshold-slider"
                    type="range"
                    className="calibration-slider-input"
                    min="0.05"
                    max="0.95"
                    step="0.01"
                    value={selectedThreshold}
                    style={{ '--threshold-progress': `${((selectedThreshold - 0.05) / 0.9) * 100}%` }}
                    onChange={(event) => setSelectedThreshold(Number(event.target.value))}
                  />
                  <div className="calibration-ticks"><span>0.05 · mais alertas</span><span>0.95 · menos alertas</span></div>
                </div>
                <div className="lab-control-actions">
                  <Button
                    type="submit"
                    disabled={savingConfig || runningExperiment}
                  >
                    {runningExperiment ? 'Executando cenário…' : 'Executar cenário'}
                  </Button>
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={handleSaveThreshold}
                    disabled={runningExperiment || savingConfig}
                  >
                    {savingConfig ? 'Salvando…' : 'Salvar como oficial'}
                  </Button>
                </div>
              </div>
            </form>

            <ThresholdComparisonCharts
              selectedThreshold={selectedThreshold}
              officialThreshold={Number(officialRun?.threshold_used ?? config.threshold ?? 0.5)}
              officialRun={officialRun}
              experiments={experiments}
              thresholdMetrics={thresholdMetrics}
              onSelectThreshold={setSelectedThreshold}
            />

            <div className="lab-results">
              <header>
                <div>
                  <span className="lab-eyebrow">Frota atual</span>
                  <h2>Cenários executados na frota atual</h2>
                  <p>As contagens vêm das probabilidades persistidas em cada execução. As métricas históricas pertencem ao artefato indicado.</p>
                </div>
                <span className="lab-results__count">{experiments.length} {experiments.length === 1 ? 'execução' : 'execuções'}</span>
              </header>
              {experiments.length ? (
                <div className="lab-scenarios-table-wrap">
                  <table className="lab-scenarios-table">
                    <thead>
                      <tr>
                        <th scope="col">Cenário</th>
                        <th scope="col">Threshold</th>
                        <th scope="col">Perfil</th>
                        <th scope="col">Portas em risco</th>
                        <th scope="col">Taxa de alerta</th>
                        <th scope="col">Precisão</th>
                        <th scope="col">Recall</th>
                        <th scope="col">F1-Score</th>
                        <th scope="col">Maior risco</th>
                        <th scope="col" className="lab-scenarios-table__action-title">Ação</th>
                      </tr>
                    </thead>
                    <tbody>
                      {experiments.map((run) => {
                        const total = Number(run.total_doors_evaluated) || 0
                        const risk = Number(run.total_doors_at_risk) || 0
                        const officialRisk = Number(officialRun?.total_doors_at_risk) || 0
                        const delta = risk - officialRisk
                        return (
                          <tr key={run.id}>
                            <td className="lab-scenarios-table__identity">
                              <strong>Cenário {run.id.slice(0, 4).toUpperCase()}</strong>
                              <time dateTime={run.created_at}>{new Date(String(run.created_at).replace(' ', 'T')).toLocaleString('pt-BR')}</time>
                              <small>ID {run.id.slice(0, 8)}</small>
                            </td>
                            <td className="lab-scenarios-table__threshold">{Number(run.threshold_used).toFixed(2)}</td>
                            <td><span className="lab-scenario__kind">{thresholdProfile(run.threshold_used).label}</span></td>
                            <td><strong>{risk}</strong><small>{delta > 0 ? '+' : ''}{delta} versus o oficial</small></td>
                            <td><strong>{total ? ((risk / total) * 100).toFixed(1) : '0,0'}%</strong><small>{risk} de {total} portas</small></td>
                            <td title="Precisão do teste histórico da versão informada">{percent(run.estimated_precision)}</td>
                            <td title="Recall do teste histórico da versão informada">{percent(run.estimated_recall)}</td>
                            <td title="F1 do teste histórico da versão informada">{percent(run.estimated_f1)}</td>
                            <td>{percent(run.maximum_probability)}</td>
                            <td className="lab-scenarios-table__action">
                              <button type="button" onClick={() => handlePromoteExperiment(run)} disabled={Boolean(promotingExperiment)}>{promotingExperiment === run.id ? 'Publicando…' : 'Usar na operação'}</button>
                            </td>
                          </tr>
                        )
                      })}
                    </tbody>
                  </table>
                </div>
              ) : <p className="lab-empty">Nenhum cenário foi executado. Escolha um threshold e rode a primeira comparação.</p>}
              <footer>Validação histórica do modelo {model?.version || 'indisponível'} · {integer(testRows)} registros · {model?.test_start ? new Date(model.test_start).toLocaleDateString('pt-BR') : 'período indisponível'} a {model?.test_end ? new Date(model.test_end).toLocaleDateString('pt-BR') : '—'}.</footer>
            </div>

            <div className="lab-results">
              <header>
                <div>
                  <span className="lab-eyebrow">Após o horizonte</span>
                  <h2>Desempenho observado em produção</h2>
                  <p>Somente previsões com a janela completa de sete dias entram nesta avaliação.</p>
                </div>
                <span className="lab-results__count">{productionPerformance?.mature_runs ?? 0} rodadas maduras</span>
              </header>
              <div className="engenharia-metrics">
                <div className="engenharia-metric"><span>Cobertura de rótulos</span><strong>{percent(productionPerformance?.label_coverage)}</strong><small>{integer(productionPerformance?.evaluated_predictions)} de {integer(productionPerformance?.total_predictions)} previsões</small></div>
                <div className="engenharia-metric"><span>Precisão observada</span><strong>{percent(productionPerformance?.precision)}</strong><small>TP {integer(productionPerformance?.true_positives)} · FP {integer(productionPerformance?.false_positives)}</small></div>
                <div className="engenharia-metric"><span>Recall observado</span><strong>{percent(productionPerformance?.recall)}</strong><small>TP {integer(productionPerformance?.true_positives)} · FN {integer(productionPerformance?.false_negatives)}</small></div>
                <div className="engenharia-metric"><span>F1 observado</span><strong>{percent(productionPerformance?.f1_score)}</strong><small>{productionPerformance?.failure_definition || 'Rótulo ainda não avaliado'}</small></div>
              </div>
            </div>
          </section>
        </div>
      ) : null}

      {!loading && activeTab === 'retrain' ? (
        <div className="engenharia-panel-full" id="engineering-panel-retrain" role="tabpanel" aria-labelledby="engineering-tab-retrain">
          <div className="engenharia-banner">
            <div className="engenharia-banner__content">
              <strong>Pipeline de retreinamento contínuo</strong>
              <span>Recomputa o pipeline temporal integrando novas telemetrias e ordens ZM01/ZM02/ZM03.</span>
            </div>
            <button
              type="button"
              className="engenharia-trigger-btn"
              onClick={handleTriggerRetrain}
              disabled={retraining}
            >
              <TriggerIcon />
              <span>{retraining ? 'Acompanhando pipeline…' : 'Iniciar retreinamento'}</span>
            </button>
          </div>

          {trainingJob ? (
            <div className="drift-status-banner">
              <span className="engenharia-status-dot" aria-hidden="true" />
              <div>
                <strong>Treinamento {trainingStatusLabel(trainingJob.status)}</strong>
                <p>Job {trainingJob.id} · versão solicitada {trainingJob.version || 'em definição'}{trainingJob.error ? ` · ${trainingJob.error}` : ''}</p>
              </div>
            </div>
          ) : null}

          <article className="engenharia-card">
            <header className="engenharia-card__head">
              <h2>Versões registradas</h2>
              <span className="engenharia-tag">Artefatos versionados</span>
            </header>

            <div className="table-responsive">
              <table className="table-standard">
                <thead>
                  <tr>
                    <th>Versão</th>
                    <th>Algoritmo</th>
                    <th>Recall de falhas</th>
                    <th>Precisão</th>
                    <th>F1-Score</th>
                    <th>ROC-AUC</th>
                    <th>Artefato serializado</th>
                    <th>Status</th>
                    <th>Ação</th>
                  </tr>
                </thead>
                <tbody>
                  {history.length ? history.map((run) => (
                    <tr key={run.id}>
                      <td><strong>{run.version}</strong></td>
                      <td>{run.algorithm || 'Indisponível'}</td>
                      <td><strong>{percent(run.test_metrics?.recall)}</strong></td>
                      <td>{percent(run.test_metrics?.precision)}</td>
                      <td><strong>{percent(run.test_metrics?.f1_score ?? run.test_metrics?.f1)}</strong></td>
                      <td>{decimal(run.test_metrics?.roc_auc)}</td>
                      <td><code>{run.artifact_uri || run.artifact_path || 'Indisponível'}</code></td>
                      <td><span className={`badge-tag ${run.status === 'production' ? 'is-active' : ''}`}>{trainingStatusLabel(run.status)}</span></td>
                      <td>{run.status !== 'production' ? <button type="button" onClick={() => handlePromoteModel(run.version)} disabled={Boolean(promotingModel)}>{promotingModel === run.version ? 'Promovendo…' : run.status === 'retired' ? 'Reverter' : 'Promover'}</button> : '—'}</td>
                    </tr>
                  )) : <tr><td className="table-empty" colSpan="9">Nenhuma versão registrada.</td></tr>}
                </tbody>
              </table>
            </div>

            <footer className="engenharia-card__footer">
              <span>Cada versão referencia um artefato imutável, hashes do modelo e do dataset e sua avaliação temporal.</span>
            </footer>
          </article>

          <article className="engenharia-card">
            <header className="engenharia-card__head">
              <h2>Histórico de execuções de treinamento</h2>
              <span className="engenharia-tag">Registro de execuções</span>
            </header>

            <div className="table-responsive">
              <table className="table-standard">
                <thead>
                  <tr>
                    <th>ID da execução</th>
                    <th>Versão solicitada</th>
                    <th>Status</th>
                    <th>Solicitado em</th>
                    <th>Iniciado em</th>
                    <th>Finalizado em</th>
                    <th>Resultado</th>
                  </tr>
                </thead>
                <tbody>
                  {trainingHistory.length ? trainingHistory.map((job) => (
                      <tr key={job.id}>
                        <td><code>{job.id?.slice(0, 14)}</code></td>
                        <td><strong>{job.version || 'Em definição'}</strong></td>
                        <td>
                          <span className={`badge-tag is-${job.status || 'stable'}`}>
                            {trainingStatusLabel(job.status)}
                          </span>
                        </td>
                        <td>{job.created_at ? new Date(job.created_at).toLocaleString('pt-BR') : '—'}</td>
                        <td>{job.started_at ? new Date(job.started_at).toLocaleString('pt-BR') : '—'}</td>
                        <td>{job.finished_at ? new Date(job.finished_at).toLocaleString('pt-BR') : '—'}</td>
                        <td>{job.error || (job.status === 'completed' ? 'Manifest registrado' : '—')}</td>
                      </tr>
                    )) : (
                      <tr><td className="table-empty" colSpan="7">Nenhuma execução de treinamento registrada.</td></tr>
                    )}
                </tbody>
              </table>
            </div>

            <footer className="engenharia-card__footer">
              <span>Trilhas de auditoria e governança MLOps no PostgreSQL.</span>
            </footer>
          </article>
        </div>
      ) : null}

      {!loading && activeTab === 'drift' ? (
        <div className="engenharia-panel-full" id="engineering-panel-drift" role="tabpanel" aria-labelledby="engineering-tab-drift">
          <div className="drift-status-banner">
            <span className={`engenharia-status-dot ${drift?.status === 'healthy' ? 'engenharia-status-dot--stable' : ''}`} aria-hidden="true" />
            <div>
              <strong>{drift?.status === 'unavailable' ? 'Drift indisponível para este artefato' : drift?.data_drift_detected ? 'Mudança de distribuição detectada' : 'Distribuição de telemetria estável'}</strong>
              <p>{drift?.message || `PSI máximo: ${decimal(drift?.psi_score)} · limite de atenção: 0,100 · ${drift?.features_monitored ?? 0} features avaliadas.`}</p>
            </div>
          </div>

          <div className="engenharia-panels">
            <article className="engenharia-card">
              <header className="engenharia-card__head">
                <h2>Estabilidade por atributo (PSI)</h2>
                <span className="engenharia-tag">Snapshot atual</span>
              </header>

              <div className="table-responsive">
                <table className="table-standard">
                  <thead>
                    <tr>
                      <th>Feature</th>
                      <th>Score PSI</th>
                      <th>Condição</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(drift?.features_drift || []).map((item) => (
                      <tr key={item.feature}>
                        <td><code>{item.feature}</code></td>
                        <td><strong>{decimal(item.psi)}</strong></td>
                        <td><span className="badge-tag">{item.status === 'stable' ? 'Estável' : item.status === 'warning' ? 'Atenção' : 'Crítico'}</span></td>
                      </tr>
                    ))}
                    {!drift?.features_drift?.length ? <tr><td className="table-empty" colSpan="3">Nenhuma distribuição de referência disponível.</td></tr> : null}
                  </tbody>
                </table>
              </div>

              <footer className="engenharia-card__footer">
                <span>Distribuição das features atuais comparada à referência registrada no treinamento.</span>
              </footer>
            </article>

            <article className="engenharia-card">
              <header className="engenharia-card__head">
                <h2>Procedência da comparação</h2>
                <span className="engenharia-tag">Modelo {drift?.model_version || model?.version || '—'}</span>
              </header>

              <div className="engenharia-card__body">
                <div className="engenharia-sensor-metrics">
                  <div className="engenharia-sensor-metric">
                    <span>Dataset de referência</span>
                    <strong>{drift?.reference_dataset_sha256 ? `${drift.reference_dataset_sha256.slice(0, 12)}…` : 'Indisponível'}</strong>
                    <small>Hash do snapshot usado no treinamento</small>
                  </div>
                  <div className="engenharia-sensor-metric">
                    <span>Última avaliação</span>
                    <strong>{drift?.evaluated_at ? new Date(drift.evaluated_at).toLocaleDateString('pt-BR') : 'Indisponível'}</strong>
                    <small>Distribuição atual contra o baseline versionado</small>
                  </div>
                </div>
              </div>

              <footer className="engenharia-card__footer">
                <span>O status é derivado dos PSI calculados; valores ausentes não recebem substituição fictícia.</span>
              </footer>
            </article>
          </div>
        </div>
      ) : null}
    </section>
  )
}

export default Engenharia
