import { useEffect, useState } from 'react'
import TrainingPipeline from '../components/TrainingPipeline'
import Button from '../components/Button'
import { useAuth } from '../context/Auth'
import { useToast } from '../context/Toast'
import { api, errorMessage } from '../lib/api'
import { invalidateApplicationCache } from '../lib/appPrefetch'
import { ThresholdComparisonCharts } from '../components/ThresholdComparisonCharts'
import './Engenharia.css'


function PromoteIcon() {
  return (
    <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden="true">
      <path d="M8 11.5v-7m0 0L5.25 7.25M8 4.5l2.75 2.75" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M3.5 11.5v1h9v-1" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function ScheduleIcon() {
  return (
    <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true">
      <path d="M5.25 2.75v2.5M14.75 2.75v2.5M3.25 7.25h13.5" strokeLinecap="round" />
      <rect x="3.25" y="4.25" width="13.5" height="12.5" rx="2" />
      <path d="M10 9.25v3l2 1.25" strokeLinecap="round" strokeLinejoin="round" />
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

function formatModelDate(value, withTime = false) {
  if (!value) return 'Indisponível'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return 'Indisponível'
  return withTime ? date.toLocaleString('pt-BR') : date.toLocaleDateString('pt-BR')
}

function predictionScheduleStatusLabel(status) {
  const labels = {
    idle: 'Aguardando ativação',
    running: 'Executando agora',
    completed: 'Última execução concluída',
    failed: 'Última execução falhou',
  }
  return labels[status] || 'Sem execução registrada'
}

function formatTargetName(target) {
  if (!target) return 'Não informado'
  if (target === 'falha_nos_proximos_7_dias') return 'Falha nos próximos 7 dias'
  const normalized = String(target).replaceAll('_', ' ')
  return normalized.charAt(0).toUpperCase() + normalized.slice(1)
}

function formatParameterName(name) {
  const labels = {
    n_estimators: 'Número de árvores',
    max_depth: 'Profundidade máxima',
    min_samples_leaf: 'Amostras mínimas por folha',
    min_samples_split: 'Amostras mínimas para divisão',
    class_weight: 'Peso das classes',
    random_state: 'Semente aleatória',
    max_features: 'Features por divisão',
    criterion: 'Critério de divisão',
    bootstrap: 'Amostragem bootstrap',
    n_jobs: 'Processamento paralelo',
  }
  return labels[name] || name.replaceAll('_', ' ')
}

function formatParameterValue(value) {
  if (value === null || value === undefined) return 'Não definido'
  if (typeof value === 'boolean') return value ? 'Ativado' : 'Desativado'
  if (Array.isArray(value)) return value.join(', ')
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

const modelCardSections = [
  { id: 'purpose', index: '01', label: 'Finalidade' },
  { id: 'governance', index: '02', label: 'Governança' },
  { id: 'validation', index: '03', label: 'Validação' },
  { id: 'lineage', index: '04', label: 'Rastreabilidade' },
  { id: 'configuration', index: '05', label: 'Configuração' },
]

function ModelCard({ model, operationalThreshold, onClose }) {
  const [activeSection, setActiveSection] = useState('purpose')
  const manifest = model?.manifest || {}
  const features = Array.isArray(manifest.features) ? manifest.features : []
  const hyperparameters = Object.entries(manifest.hyperparameters || {})
  const modelThreshold = Number(operationalThreshold ?? model?.recommended_threshold ?? 0.5)
  const modelMetric = nearestThresholdMetric(model?.threshold_metrics || [], modelThreshold)
  const metrics = model?.test_metrics || {}
  const featureCount = features.length || manifest.features_count || model?.feature_importance?.length || 0

  function handleTabKeyDown(event) {
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return

    event.preventDefault()
    const currentIndex = modelCardSections.findIndex((section) => section.id === activeSection)
    let nextIndex = currentIndex

    if (event.key === 'Home') nextIndex = 0
    if (event.key === 'End') nextIndex = modelCardSections.length - 1
    if (event.key === 'ArrowLeft') nextIndex = (currentIndex - 1 + modelCardSections.length) % modelCardSections.length
    if (event.key === 'ArrowRight') nextIndex = (currentIndex + 1) % modelCardSections.length

    const nextSection = modelCardSections[nextIndex]
    setActiveSection(nextSection.id)
    requestAnimationFrame(() => document.getElementById(`model-card-tab-${nextSection.id}`)?.focus())
  }

  return (
    <div className="model-card-backdrop" onClick={onClose}>
      <article
        className="model-card-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="model-card-title"
        onClick={(event) => event.stopPropagation()}
      >
        <header className="model-card-head">
          <div>
            <span className="model-card-eyebrow">Model Card · Governança do artefato</span>
            <div className="model-card-title-row">
              <h2 id="model-card-title">{model?.version || 'Versão indisponível'}</h2>
              <span className={`badge-tag is-${model?.status || 'candidate'}`}>{trainingStatusLabel(model?.status)}</span>
            </div>
            <p>{model?.algorithm || 'Algoritmo não informado'} · {formatTargetName(model?.target || manifest.target)}</p>
          </div>
          <button type="button" className="model-card-close" onClick={onClose} aria-label="Fechar Model Card" autoFocus>✕</button>
        </header>

        <div className="model-card-workspace">
          <nav className="model-card-tabs" role="tablist" aria-label="Seções do Model Card" onKeyDown={handleTabKeyDown}>
            <p>Conteúdo</p>
            {modelCardSections.map((section) => (
              <button
                key={section.id}
                id={`model-card-tab-${section.id}`}
                type="button"
                role="tab"
                aria-selected={activeSection === section.id}
                aria-controls={`model-card-panel-${section.id}`}
                tabIndex={activeSection === section.id ? 0 : -1}
                className={activeSection === section.id ? 'is-active' : ''}
                onClick={() => setActiveSection(section.id)}
              >
                <span>{section.index}</span>
                <strong>{section.label}</strong>
              </button>
            ))}
          </nav>

          <div className="model-card-body">
          {activeSection === 'purpose' ? <section
            id="model-card-panel-purpose"
            className="model-card-purpose"
            role="tabpanel"
            aria-labelledby="model-card-tab-purpose"
          >
            <div>
              <div>
                <h3 id="model-card-purpose-title">Finalidade</h3>
                <p>Estimar o risco de falha de cada porta da Linha 4 dentro da janela preditiva definida pelo artefato, apoiando a priorização de inspeções preventivas.</p>
              </div>
            </div>
            <dl>
              <div><dt>Variável-alvo</dt><dd>{formatTargetName(model?.target || manifest.target)}</dd></div>
              <div><dt>Uso operacional</dt><dd>Priorização de portas para inspeção</dd></div>
              <div><dt>Saída</dt><dd>Probabilidade e classificação binária</dd></div>
            </dl>
          </section> : null}

          {activeSection === 'governance' ? <section
            id="model-card-panel-governance"
            className="model-card-section"
            role="tabpanel"
            aria-labelledby="model-card-tab-governance"
          >
            <header><div><h3 id="model-card-governance-title">Identidade e governança</h3><p>Versão, autoria e configuração usada para publicar decisões operacionais.</p></div></header>
            <dl className="model-card-definition-grid">
              <div><dt>Versão</dt><dd>{model?.version || 'Indisponível'}</dd></div>
              <div><dt>Status</dt><dd>{trainingStatusLabel(model?.status)}</dd></div>
              <div><dt>Threshold operacional</dt><dd>{modelThreshold.toFixed(2)}</dd></div>
              <div><dt>Threshold recomendado</dt><dd>{Number.isFinite(Number(model?.recommended_threshold)) ? Number(model.recommended_threshold).toFixed(2) : 'Indisponível'}</dd></div>
              <div><dt>Gerado em</dt><dd>{formatModelDate(manifest.generated_at || model?.finished_at, true)}</dd></div>
              <div><dt>Ativado em</dt><dd>{formatModelDate(model?.activated_at, true)}</dd></div>
              <div><dt>Responsável</dt><dd>{model?.created_by || 'Pipeline automatizado'}</dd></div>
              <div><dt>Total de features</dt><dd>{integer(featureCount)}</dd></div>
            </dl>
          </section> : null}

          {activeSection === 'validation' ? <section
            id="model-card-panel-validation"
            className="model-card-section"
            role="tabpanel"
            aria-labelledby="model-card-tab-validation"
          >
            <header><div><h3 id="model-card-validation-title">Validação temporal</h3><p>Métricas registradas no teste walk-forward do mesmo artefato.</p></div></header>
            <div className="model-card-metrics">
              <div><span>Recall</span><strong>{percent(modelMetric?.recall ?? metrics.recall)}</strong></div>
              <div><span>Precisão</span><strong>{percent(modelMetric?.precision ?? metrics.precision)}</strong></div>
              <div><span>F1-Score</span><strong>{percent(modelMetric?.f1_score ?? metrics.f1_score ?? metrics.f1)}</strong></div>
              <div><span>ROC-AUC</span><strong>{decimal(metrics.roc_auc)}</strong></div>
              <div><span>Registros de teste</span><strong>{integer(model?.test_rows)}</strong></div>
              <div><span>Falhas no teste</span><strong>{integer(model?.positive_rows)}</strong></div>
            </div>
            <dl className="model-card-periods">
              <div><dt>Treino até</dt><dd>{formatModelDate(model?.train_end || manifest.train_end)}</dd></div>
              <div><dt>Teste de</dt><dd>{formatModelDate(model?.test_start || manifest.test_start)}</dd></div>
              <div><dt>Teste até</dt><dd>{formatModelDate(model?.test_end || manifest.test_end)}</dd></div>
            </dl>
          </section> : null}

          {activeSection === 'lineage' ? <section
            id="model-card-panel-lineage"
            className="model-card-section"
            role="tabpanel"
            aria-labelledby="model-card-tab-lineage"
          >
            <header><div><h3 id="model-card-lineage-title">Rastreabilidade</h3><p>Referências necessárias para reproduzir e verificar esta versão.</p></div></header>
            <dl className="model-card-lineage">
              <div><dt>Artefato</dt><dd><code>{model?.artifact_uri || model?.artifact_path || 'Indisponível'}</code></dd></div>
              <div><dt>SHA-256 do artefato</dt><dd><code>{model?.artifact_sha256 || 'Indisponível'}</code></dd></div>
              <div><dt>SHA-256 do dataset</dt><dd><code>{model?.dataset_sha256 || 'Indisponível'}</code></dd></div>
            </dl>
          </section> : null}

          {activeSection === 'configuration' ? <div
            id="model-card-panel-configuration"
            className="model-card-configuration-panel"
            role="tabpanel"
            aria-labelledby="model-card-tab-configuration"
          >
            <section className="model-card-section" aria-labelledby="model-card-parameters-title">
              <header><div><h3 id="model-card-parameters-title">Configuração técnica</h3><p>Hiperparâmetros persistidos no manifest versionado.</p></div></header>
              {hyperparameters.length ? (
                <dl className="model-card-parameters">
                  {hyperparameters.map(([name, value]) => <div key={name}><dt>{formatParameterName(name)}</dt><dd>{formatParameterValue(value)}</dd></div>)}
                </dl>
              ) : <p className="model-card-empty">Hiperparâmetros indisponíveis para esta versão.</p>}
            </section>

            <details className="model-card-features">
              <summary><span>Features do modelo</span><strong>{integer(featureCount)} atributos</strong></summary>
              {features.length ? <div>{features.map((feature) => <code key={feature}>{feature}</code>)}</div> : <p>Lista de features indisponível no manifest.</p>}
            </details>
          </div> : null}
          </div>
        </div>
      </article>
    </div>
  )
}

function Engenharia() {
  const { user } = useAuth()
  const toast = useToast()

  const [activeTab, setActiveTab] = useState('model')
  const [loading, setLoading] = useState(true)
  const [runningExperiment, setRunningExperiment] = useState(false)
  const [promotingExperiment, setPromotingExperiment] = useState('')
  const [retraining, setRetraining] = useState(false)
  const [cancellingTraining, setCancellingTraining] = useState(false)
  const [trainingJob, setTrainingJob] = useState(null)
  const [trainingHistory, setTrainingHistory] = useState([])
  const [promotingModel, setPromotingModel] = useState('')
  const [selectedModelCard, setSelectedModelCard] = useState(null)
  const [savingSchedule, setSavingSchedule] = useState(false)
  const [predictionSchedule, setPredictionSchedule] = useState({
    enabled: false,
    interval_minutes: 1440,
    last_status: 'idle',
  })

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

  const [selectedThreshold, setSelectedThreshold] = useState(0.5)
  const selectedProfile = thresholdProfile(selectedThreshold)
  const activeTrainingJobID = ['queued', 'running'].includes(trainingJob?.status) ? trainingJob.id : ''

  useEffect(() => {
    async function loadData() {
      try {
        setLoading(true)
        const [cfgData, modelData, histData, driftData, experimentData, officialData, trainingData, scheduleData] = await Promise.all([
          api.getMLConfig().catch(() => ({ threshold: 0.5, default_horizon_days: 7 })),
          api.getActiveModel().catch(() => null),
          api.getModelHistory(5).catch(() => []),
          api.getMLDrift().catch(() => null),
          api.predictionExperiments(20).catch(() => []),
          api.predictionsLatest().catch(() => null),
          api.getTrainingJobs(20).catch(() => []),
          api.getPredictionSchedule().catch(() => ({ enabled: false, interval_minutes: 1440, last_status: 'idle' })),
        ])
        setConfig(cfgData)
        setSelectedThreshold(cfgData.threshold ?? 0.5)
        setModel(modelData)
        setHistory(histData || [])
        setDrift(driftData)
        setExperiments(experimentData || [])
        setOfficialRun(officialData)
        setTrainingHistory(trainingData || [])
        setPredictionSchedule(scheduleData)
        const latestTrainingJob = (trainingData || [])[0] || null
        setTrainingJob(latestTrainingJob)
        setRetraining(['queued', 'running'].includes(latestTrainingJob?.status))
      } finally {
        setLoading(false)
      }
    }
    loadData()
  }, [])

  useEffect(() => {
    if (!selectedModelCard) return undefined
    const previousOverflow = document.body.style.overflow
    const closeOnEscape = (event) => {
      if (event.key === 'Escape') setSelectedModelCard(null)
    }
    document.body.style.overflow = 'hidden'
    document.addEventListener('keydown', closeOnEscape)
    return () => {
      document.body.style.overflow = previousOverflow
      document.removeEventListener('keydown', closeOnEscape)
    }
  }, [selectedModelCard])

  useEffect(() => {
    const jobID = activeTrainingJobID
    if (!jobID) return undefined

    let cancelled = false
    let timer

    async function pollTrainingJob() {
      try {
        const current = await api.getTrainingJob(jobID)
        if (cancelled) return

        if (current.status === 'completed') {
          const [models, active, jobs] = await Promise.all([
            api.getModelHistory(20),
            api.getActiveModel(),
            api.getTrainingJobs(20),
          ])
          if (cancelled) return
          setTrainingJob(current)
          setHistory(models || [])
          setModel(active)
          setTrainingHistory(jobs || [])
          setRetraining(false)
          toast.success('Retreinamento concluído. A nova versão está disponível como candidata.')
          return
        }

        if (current.status === 'failed') {
          setTrainingJob(current)
          setTrainingHistory((items) => [current, ...items.filter((item) => item.id !== current.id)])
          setRetraining(false)
          toast.error(`Retreinamento falhou: ${current.error || 'consulte o checkpoint interrompido.'}`)
          return
        }

        if (current.status === 'cancelled') {
          setTrainingJob(current)
          setTrainingHistory((items) => [current, ...items.filter((item) => item.id !== current.id)])
          setRetraining(false)
          return
        }

        setTrainingJob(current)
        setTrainingHistory((items) => [current, ...items.filter((item) => item.id !== current.id)])
        timer = window.setTimeout(pollTrainingJob, 2500)
      } catch {
        if (cancelled) return
        timer = window.setTimeout(pollTrainingJob, 5000)
      }
    }

    timer = window.setTimeout(pollTrainingJob, 800)
    return () => {
      cancelled = true
      window.clearTimeout(timer)
    }
  }, [activeTrainingJobID, toast])

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
    } catch (err) {
      toast.error('Erro no retreinamento: ' + errorMessage(err))
      setRetraining(false)
    }
  }

  async function handleResumeTraining() {
    if (!trainingJob?.id) return
    try {
      setRetraining(true)
      const resumed = await api.resumeTraining(trainingJob.id)
      setTrainingJob(resumed)
      setTrainingHistory((current) => [resumed, ...current.filter((item) => item.id !== resumed.id)])
      if (resumed.status === 'completed') {
        const models = await api.getModelHistory(20)
        setHistory(models || [])
        setRetraining(false)
        toast.success('Registro retomado e concluído a partir da etapa interrompida.')
        return
      }
      toast.success('Treinamento retomado a partir do último checkpoint válido.')
    } catch (err) {
      toast.error('Erro ao retomar treinamento: ' + errorMessage(err))
      setRetraining(false)
    }
  }

  async function handleCancelTraining() {
    if (!trainingJob?.id || !['queued', 'running'].includes(trainingJob.status)) return
    try {
      setCancellingTraining(true)
      const cancelled = await api.cancelTraining(trainingJob.id)
      setTrainingJob(cancelled)
      setTrainingHistory((current) => [cancelled, ...current.filter((item) => item.id !== cancelled.id)])
      setRetraining(false)
      toast.success('Treinamento cancelado. Os checkpoints concluídos foram preservados.')
    } catch (err) {
      toast.error('Erro ao cancelar treinamento: ' + errorMessage(err))
    } finally {
      setCancellingTraining(false)
    }
  }

  async function handlePromoteModel(version) {
    try {
      setPromotingModel(version)
      const result = await api.promoteModel(version)
      const models = await api.getModelHistory(20)
      setModel(result.model)
      setOfficialRun(result.prediction_run)
      setHistory(models || [])
      invalidateApplicationCache()
      toast.success(`Modelo ${version} promovido e nova rodada oficial publicada.`)
    } catch (err) {
      toast.error('Falha ao promover modelo: ' + errorMessage(err))
    } finally {
      setPromotingModel('')
    }
  }

  async function handleSavePredictionSchedule(event) {
    event.preventDefault()
    try {
      setSavingSchedule(true)
      const updated = await api.updatePredictionSchedule(
        predictionSchedule.enabled,
        predictionSchedule.interval_minutes,
      )
      setPredictionSchedule(updated)
      toast.success(updated.enabled ? 'Execução automática configurada.' : 'Execução automática desativada.')
    } catch (err) {
      toast.error('Falha ao salvar agendamento: ' + errorMessage(err))
    } finally {
      setSavingSchedule(false)
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
            <span><small>Modelo em produção</small><strong title={model?.version}>{model?.version || 'Indisponível'}</strong></span>
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
            <button type="button" className="engenharia-model-card-trigger" onClick={() => setSelectedModelCard(model)} disabled={!model}>
              Ver Model Card
              <span aria-hidden="true">↗</span>
            </button>
          </div>

          <form
            className={`prediction-schedule-card${predictionSchedule.enabled ? ' is-enabled' : ''}${predictionSchedule.last_status === 'failed' ? ' is-failed' : ''}`}
            onSubmit={handleSavePredictionSchedule}
          >
            <header className="prediction-schedule-card__header">
              <div className="prediction-schedule-card__identity">
                <span className="prediction-schedule-card__icon"><ScheduleIcon /></span>
                <div className="prediction-schedule-card__copy">
                  <span className="lab-eyebrow">Automação operacional</span>
                  <h2>Agendamento de predições</h2>
                  <p>Gera rodadas oficiais com o modelo e o threshold ativos no momento de cada execução.</p>
                </div>
              </div>

              <div className="prediction-schedule-card__activation">
                <div className="prediction-schedule-card__state" aria-live="polite">
                  <i aria-hidden="true" />
                  <span>{predictionScheduleStatusLabel(predictionSchedule.last_status)}</span>
                </div>
                <label className="prediction-schedule-toggle">
                  <strong>{predictionSchedule.enabled ? 'Automação ativa' : 'Automação pausada'}</strong>
                  <input
                    type="checkbox"
                    checked={Boolean(predictionSchedule.enabled)}
                    onChange={(event) => setPredictionSchedule((current) => ({ ...current, enabled: event.target.checked }))}
                  />
                  <span aria-hidden="true" />
                </label>
              </div>
            </header>

            <div className="prediction-schedule-card__controls">
              <label className="prediction-schedule-field">
                <span>Frequência</span>
                <select
                  value={predictionSchedule.interval_minutes}
                  disabled={!predictionSchedule.enabled}
                  onChange={(event) => setPredictionSchedule((current) => ({
                    ...current,
                    interval_minutes: Number(event.target.value),
                  }))}
                >
                  <option value="15">A cada 15 minutos</option>
                  <option value="30">A cada 30 minutos</option>
                  <option value="60">A cada 1 hora</option>
                  <option value="360">A cada 6 horas</option>
                  <option value="720">A cada 12 horas</option>
                  <option value="1440">A cada 24 horas</option>
                  <option value="10080">A cada 7 dias</option>
                </select>
              </label>

              <div className="prediction-schedule-card__detail">
                <span>Próxima execução</span>
                <strong>{predictionSchedule.enabled && predictionSchedule.next_run_at
                  ? formatModelDate(predictionSchedule.next_run_at, true)
                  : 'Não agendada'}</strong>
                <small>{predictionSchedule.enabled ? 'Horário calculado pelo worker' : 'Ative a automação para programar'}</small>
              </div>

              <div className="prediction-schedule-card__detail">
                <span>Última execução</span>
                <strong>{predictionSchedule.last_completed_at
                  ? formatModelDate(predictionSchedule.last_completed_at, true)
                  : 'Ainda não executada'}</strong>
                <small>{predictionScheduleStatusLabel(predictionSchedule.last_status)}</small>
              </div>

              <button type="submit" className="prediction-schedule-save" disabled={savingSchedule}>
                {savingSchedule ? 'Salvando…' : 'Salvar agenda'}
              </button>
            </div>

            {predictionSchedule.last_error ? (
              <p className="prediction-schedule-card__error" role="alert">{predictionSchedule.last_error}</p>
            ) : null}
          </form>

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
              <div className="lab-official__model-metric">
                <dt>Modelo</dt>
                <dd
                  className="lab-official__model-version"
                  title={officialRun?.model_version || model?.version || 'Indisponível'}
                >
                  {officialRun?.model_version || model?.version || 'Indisponível'}
                </dd>
              </div>
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
                    disabled={runningExperiment}
                  >
                    {runningExperiment ? 'Executando cenário…' : 'Executar cenário'}
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
                            <td className="lab-scenarios-table__identity" data-label="Cenário">
                              <strong>Cenário {run.id.slice(0, 4).toUpperCase()}</strong>
                              <time dateTime={run.created_at}>{new Date(String(run.created_at).replace(' ', 'T')).toLocaleString('pt-BR')}</time>
                              <small>ID {run.id.slice(0, 8)}</small>
                            </td>
                            <td className="lab-scenarios-table__threshold" data-label="Threshold">{Number(run.threshold_used).toFixed(2)}</td>
                            <td data-label="Perfil"><span className="lab-scenario__kind">{thresholdProfile(run.threshold_used).label}</span></td>
                            <td data-label="Portas em risco"><strong>{risk}</strong><small>{delta > 0 ? '+' : ''}{delta} versus o oficial</small></td>
                            <td data-label="Taxa de alerta"><strong>{total ? ((risk / total) * 100).toFixed(1) : '0,0'}%</strong><small>{risk} de {total} portas</small></td>
                            <td data-label="Precisão" title="Precisão do teste histórico da versão informada">{percent(run.estimated_precision)}</td>
                            <td data-label="Recall" title="Recall do teste histórico da versão informada">{percent(run.estimated_recall)}</td>
                            <td data-label="F1-Score" title="F1 do teste histórico da versão informada">{percent(run.estimated_f1)}</td>
                            <td data-label="Maior risco">{percent(run.maximum_probability)}</td>
                            <td className="lab-scenarios-table__action" data-label="Ação">
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

          </section>
        </div>
      ) : null}

      {!loading && activeTab === 'retrain' ? (
        <div className="engenharia-panel-full" id="engineering-panel-retrain" role="tabpanel" aria-labelledby="engineering-tab-retrain">
          <TrainingPipeline
            job={trainingJob}
            retraining={retraining}
            cancelling={cancellingTraining}
            onTriggerRetrain={handleTriggerRetrain}
            onResumeTraining={handleResumeTraining}
            onCancelTraining={handleCancelTraining}
          />

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
                    <th className="table-action-column">Ação</th>
                  </tr>
                </thead>
                <tbody>
                  {history.length ? history.map((run) => (
                    <tr key={run.id}>
                      <td>
                        <button type="button" className="model-card-version-link" onClick={() => setSelectedModelCard(run)}>
                          {run.version}
                        </button>
                      </td>
                      <td>{run.algorithm || 'Indisponível'}</td>
                      <td><strong>{percent(run.test_metrics?.recall)}</strong></td>
                      <td>{percent(run.test_metrics?.precision)}</td>
                      <td><strong>{percent(run.test_metrics?.f1_score ?? run.test_metrics?.f1)}</strong></td>
                      <td>{decimal(run.test_metrics?.roc_auc)}</td>
                      <td><code>{run.artifact_uri || run.artifact_path || 'Indisponível'}</code></td>
                      <td><span className={`badge-tag ${run.status === 'production' ? 'is-active' : ''}`}>{trainingStatusLabel(run.status)}</span></td>
                      <td className="table-action-column">
                        {run.status !== 'production' ? (
                          <button
                            type="button"
                            className={`engenharia-promote-btn${run.status === 'retired' ? ' is-restore' : ''}`}
                            onClick={() => handlePromoteModel(run.version)}
                            disabled={Boolean(promotingModel)}
                            aria-label={`${run.status === 'retired' ? 'Restaurar' : 'Promover'} modelo ${run.version} para produção`}
                          >
                            <PromoteIcon />
                            <span>{promotingModel === run.version ? 'Publicando…' : run.status === 'retired' ? 'Restaurar' : 'Promover'}</span>
                          </button>
                        ) : <span className="engenharia-action-empty" aria-label="Modelo já está em produção">—</span>}
                      </td>
                    </tr>
                  )) : <tr><td className="table-empty" colSpan="9">Nenhuma versão registrada.</td></tr>}
                </tbody>
              </table>
            </div>

            <footer className="engenharia-card__footer">
              <span>Ao promover uma versão, o backend ativa o artefato e publica automaticamente uma nova rodada oficial com o threshold operacional vigente.</span>
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
                        <td>
                          <button type="button" className="model-card-version-link" onClick={() => setTrainingJob(job)}>
                            {job.id?.slice(0, 14)}
                          </button>
                        </td>
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

      {selectedModelCard ? (
        <ModelCard
          model={selectedModelCard}
          operationalThreshold={selectedModelCard.status === 'production' ? activeThreshold : selectedModelCard.recommended_threshold}
          onClose={() => setSelectedModelCard(null)}
        />
      ) : null}
    </section>
  )
}

export default Engenharia
