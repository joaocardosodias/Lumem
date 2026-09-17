import { useEffect, useMemo, useState } from 'react'
import './TrainingPipeline.css'

const STAGE_DEFINITIONS = [
  ['snapshot_acquisition', 'Snapshots de dados', 'INGESTÃO', 'Aquisição de snapshots diários de telemetria e ordens ZM01/ZM02/ZM03.', 'Dataset consolidado'],
  ['data_loading', 'Carga & telemetria', 'INGESTÃO', 'Carregamento do painel histórico e unificação dos registros porta-dia.', '71.184 registros porta-dia'],
  ['feature_engineering', 'Engenharia de features', 'FEATURES', 'Cálculo de 104 atributos temporais, recência de corretivas e degradação.', '104 atributos temporais'],
  ['temporal_split', 'Split temporal', 'FEATURES', 'Separação cronológica estrita walk-forward entre janela de treino e teste.', 'Treino (80%) · Teste (20%)'],
  ['initial_training', 'Treinamento inicial', 'MODELAGEM', 'Treinamento preliminar do Random Forest (400 árvores) para extração SHAP.', 'Modelo inicial convergido'],
  ['feature_selection', 'Seleção SHAP', 'MODELAGEM', 'Filtragem por TreeExplainer descartando atributos com impacto < 1% do pico.', '72 features selecionadas'],
  ['final_training', 'Treinamento final', 'MODELAGEM', 'Ajuste do modelo definitivo otimizado com a matriz de 72 features.', 'RandomForestClassifier final'],
  ['evaluation', 'Avaliação & limiar', 'VALIDAÇÃO', 'Avaliação walk-forward de Recall, Precisão, F1 e calibração de threshold.', 'Recall 55.3% · F1 24.7%'],
  ['artifact_publish', 'Serialização', 'PUBLICAÇÃO', 'Exportação do bundle serializado (.joblib) e manifesto de parâmetros.', 'modelo_5.joblib (14.2 MB)'],
  ['catalog_registration', 'Registro catálogo', 'GOVERNANÇA', 'Disponibilização da nova versão candidata para auditoria e promoção.', 'Registrado em ml_model_runs'],
]

const STAGE_METADATA = Object.fromEntries(
  STAGE_DEFINITIONS.map(([key, title, category, description, output]) => [
    key,
    { title, category, description, output },
  ])
)

const STATUS_LABELS = {
  queued: 'Na fila',
  running: 'Em execução',
  completed: 'Concluído',
  failed: 'Falhou',
  pending: 'Pendente',
  skipped: 'Ignorado',
}

const CARD_BORDER_PATHS = {
  left: [
    'M1 50 V14 Q1 1 6 1 H94 Q99 1 99 14 V50',
    'M1 50 V86 Q1 99 6 99 H94 Q99 99 99 86 V50',
  ],
  right: [
    'M99 50 V14 Q99 1 94 1 H6 Q1 1 1 14 V50',
    'M99 50 V86 Q99 99 94 99 H6 Q1 99 1 86 V50',
  ],
  top: [
    'M50 1 H6 Q1 1 1 14 V86 Q1 99 6 99 H50',
    'M50 1 H94 Q99 1 99 14 V86 Q99 99 94 99 H50',
  ],
}

function AlertIcon() {
  return (
    <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="8" cy="8" r="6" />
      <line x1="8" y1="5" x2="8" y2="8.5" />
      <circle cx="8" cy="11.5" r="0.5" fill="currentColor" />
    </svg>
  )
}

function TriggerIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" width="15" height="15" aria-hidden="true">
      <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" />
    </svg>
  )
}

function StageGlyph({ stageKey }) {
  if (stageKey === 'snapshot_acquisition') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <ellipse cx="12" cy="5" rx="9" ry="3" />
        <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3" />
        <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5" />
      </svg>
    )
  }
  if (stageKey === 'data_loading') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <line x1="4" y1="6" x2="20" y2="6" />
        <line x1="4" y1="12" x2="14" y2="12" />
        <line x1="4" y1="18" x2="18" y2="18" />
        <circle cx="18" cy="12" r="2.5" />
      </svg>
    )
  }
  if (stageKey === 'feature_engineering') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <line x1="4" y1="21" x2="4" y2="14" />
        <line x1="4" y1="10" x2="4" y2="3" />
        <line x1="12" y1="21" x2="12" y2="12" />
        <line x1="12" y1="8" x2="12" y2="3" />
        <line x1="20" y1="21" x2="20" y2="16" />
        <line x1="20" y1="12" x2="20" y2="3" />
        <line x1="1" y1="14" x2="7" y2="14" />
        <line x1="9" y1="8" x2="15" y2="8" />
        <line x1="17" y1="16" x2="23" y2="16" />
      </svg>
    )
  }
  if (stageKey === 'temporal_split') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <rect x="3" y="4" width="18" height="18" rx="3" />
        <line x1="16" y1="2" x2="16" y2="6" />
        <line x1="8" y1="2" x2="8" y2="6" />
        <line x1="3" y1="10" x2="21" y2="10" />
        <line x1="10" y1="14" x2="10" y2="18" />
        <line x1="14" y1="14" x2="14" y2="18" />
      </svg>
    )
  }
  if (stageKey === 'initial_training') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <circle cx="12" cy="12" r="3" />
        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
      </svg>
    )
  }
  if (stageKey === 'feature_selection') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <circle cx="18" cy="5" r="3" />
        <circle cx="6" cy="12" r="3" />
        <circle cx="18" cy="19" r="3" />
        <line x1="8.59" y1="13.51" x2="15.42" y2="17.49" />
        <line x1="15.41" y1="6.51" x2="8.59" y2="10.49" />
      </svg>
    )
  }
  if (stageKey === 'final_training') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <circle cx="12" cy="12" r="10" />
        <circle cx="12" cy="12" r="6" />
        <circle cx="12" cy="12" r="2" fill="currentColor" />
      </svg>
    )
  }
  if (stageKey === 'evaluation') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <line x1="18" y1="20" x2="18" y2="10" />
        <line x1="12" y1="20" x2="12" y2="4" />
        <line x1="6" y1="20" x2="6" y2="14" />
      </svg>
    )
  }
  if (stageKey === 'artifact_publish') {
    return (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
        <polyline points="3.27 6.96 12 12.01 20.73 6.96" />
        <line x1="12" y1="22.08" x2="12" y2="12" />
      </svg>
    )
  }
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
    </svg>
  )
}

function durationLabel(milliseconds) {
  const seconds = Math.max(0, Math.round(Number(milliseconds) / 1000))
  if (!Number.isFinite(seconds)) return '0s'
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  const remainder = seconds % 60
  return `${minutes}m ${String(remainder).padStart(2, '0')}s`
}

function elapsedLabel(startedAt, finishedAt, now) {
  if (!startedAt) return '0s'
  const start = new Date(startedAt).getTime()
  const end = finishedAt ? new Date(finishedAt).getTime() : now
  if (!Number.isFinite(start) || !Number.isFinite(end)) return '0s'
  return durationLabel(end - start)
}

function normalizedStages(job) {
  const received = job?.progress?.stages
  if (Array.isArray(received) && received.length) {
    return received.map((stage) => {
      const meta = STAGE_METADATA[stage.key] || {}
      return {
        ...stage,
        title: meta.title || stage.label,
        category: meta.category || 'PIPELINE',
        description: meta.description || stage.label,
        output: meta.output || '',
      }
    })
  }

  if (job?.status === 'completed') {
    return STAGE_DEFINITIONS.map(([key, title, category, description, output]) => ({
      key,
      label: title,
      title,
      category,
      description,
      output,
      status: 'completed',
    }))
  }

  return STAGE_DEFINITIONS.map(([key, title, category, description, output], index) => ({
    key,
    label: title,
    title,
    category,
    description,
    output,
    status: job?.status === 'failed' && index === 0 ? 'failed' : 'pending',
  }))
}

export default function TrainingPipeline({ job, retraining, onTriggerRetrain }) {
  const [now, setNow] = useState(0)
  const stages = useMemo(() => normalizedStages(job), [job])
  const actualCompletedPrefix = useMemo(() => {
    const firstOpenStage = stages.findIndex(
      (stage) => !['completed', 'skipped'].includes(stage.status)
    )
    return firstOpenStage === -1 ? stages.length : firstOpenStage
  }, [stages])
  const currentJobID = job?.id || null
  const [visualProgress, setVisualProgress] = useState(() => ({
    initialized: Boolean(currentJobID),
    jobID: currentJobID,
    completed: actualCompletedPrefix,
  }))

  const visualCompleted = !visualProgress.initialized
    ? actualCompletedPrefix
    : visualProgress.jobID !== currentJobID
    ? 0
    : Math.min(visualProgress.completed, actualCompletedPrefix)
  const isVisualPlayback = visualCompleted < actualCompletedPrefix

  useEffect(() => {
    if (!currentJobID) return undefined

    if (!visualProgress.initialized) {
      const timer = window.setTimeout(() => {
        setVisualProgress({ initialized: true, jobID: currentJobID, completed: actualCompletedPrefix })
      }, 0)
      return () => window.clearTimeout(timer)
    }

    if (visualProgress.jobID !== currentJobID) {
      const timer = window.setTimeout(() => {
        setVisualProgress({ initialized: true, jobID: currentJobID, completed: 0 })
      }, 0)
      return () => window.clearTimeout(timer)
    }

    if (visualProgress.completed < actualCompletedPrefix) {
      const stageDelay = visualProgress.completed === 0 ? 1000 : 1650
      const timer = window.setTimeout(() => {
        setVisualProgress((current) => ({
          ...current,
          completed: Math.min(current.completed + 1, actualCompletedPrefix),
        }))
      }, stageDelay)
      return () => window.clearTimeout(timer)
    }

    if (visualProgress.completed > actualCompletedPrefix) {
      const timer = window.setTimeout(() => {
        setVisualProgress((current) => ({ ...current, completed: actualCompletedPrefix }))
      }, 0)
      return () => window.clearTimeout(timer)
    }

    return undefined
  }, [actualCompletedPrefix, currentJobID, visualProgress])

  const visibleStages = useMemo(() => {
    if (!isVisualPlayback) return stages

    return stages.map((stage, index) => {
      if (index < visualCompleted) {
        return { ...stage, status: stage.status === 'skipped' ? 'skipped' : 'completed' }
      }
      if (index === visualCompleted) return { ...stage, status: 'running' }
      return { ...stage, status: 'pending' }
    })
  }, [isVisualPlayback, stages, visualCompleted])
  const effectiveStatus = isVisualPlayback ? 'running' : job?.status

  const activeStage = visibleStages.find((stage) => stage.status === 'running')
    || visibleStages.find((stage) => stage.status === 'failed')
    || (effectiveStatus === 'queued' ? visibleStages[0] : visibleStages.at(-1))

  const [selectedStageKey, setSelectedStageKey] = useState(null)
  const inspectedStage = visibleStages.find((s) => s.key === (selectedStageKey || activeStage?.key)) || visibleStages[0]
  const completed = visibleStages.filter((stage) => ['completed', 'skipped'].includes(stage.status)).length
  const percent = visibleStages.length ? Math.round((completed * 100) / visibleStages.length) : 0
  const isActive = ['queued', 'running'].includes(effectiveStatus) || Boolean(retraining)

  useEffect(() => {
    if (!isActive) return undefined
    const timer = window.setInterval(() => setNow(Date.now()), 1000)
    return () => window.clearInterval(timer)
  }, [isActive])

  const firstRow = visibleStages.slice(0, 5)
  const secondRow = visibleStages.slice(5, 10)

  function connectorState(currentStage, nextStage) {
    const currentCompleted = ['completed', 'skipped'].includes(currentStage?.status)
    const nextCompleted = ['completed', 'skipped'].includes(nextStage?.status)
    if (nextStage?.status === 'failed' && currentCompleted) return 'is-failed'
    if (currentCompleted && nextStage?.status === 'running') return 'is-flowing'
    if (currentCompleted && nextCompleted) return 'is-active'
    return ''
  }

  function renderCard(stage) {
    const isSelected = inspectedStage?.key === stage.key
    const stageIndex = visibleStages.findIndex((candidate) => candidate.key === stage.key)
    const previousStage = stageIndex > 0 ? visibleStages[stageIndex - 1] : null
    const hasIncomingFlow = stage.status === 'running'
      && ['completed', 'skipped'].includes(previousStage?.status)
    const incomingDirection = stageIndex === 5 ? 'top' : stageIndex > 5 ? 'right' : 'left'

    return (
      <button
        key={stage.key}
        type="button"
        className={`pipeline-flow-card is-${stage.status} flow-from-${incomingDirection} ${hasIncomingFlow ? 'has-incoming-flow' : ''} ${isSelected ? 'is-selected' : ''}`}
        onClick={() => setSelectedStageKey(stage.key)}
        aria-label={`${stage.title}: ${STATUS_LABELS[stage.status] || stage.status}`}
      >
        <svg
          className="pipeline-flow-card__trace"
          viewBox="0 0 100 100"
          preserveAspectRatio="none"
          aria-hidden="true"
        >
          {CARD_BORDER_PATHS[incomingDirection].map((path) => (
            <path
              key={path}
              d={path}
              ref={(node) => {
                if (node && !node.style.getPropertyValue('--path-len')) {
                  const svg = node.ownerSVGElement
                  if (svg) {
                    const rect = svg.getBoundingClientRect()
                    const screenLen = Math.ceil(rect.width + rect.height + 20)
                    if (screenLen > 20) {
                      node.style.setProperty('--path-len', `${screenLen}px`)
                    }
                  }
                }
              }}
            />
          ))}
        </svg>
        <span className="pipeline-flow-card__fill" aria-hidden="true" />
        <div className="pipeline-flow-card__center">
          <div className="pipeline-flow-card__icon-box">
            <StageGlyph stageKey={stage.key} />
          </div>
          <div className="pipeline-flow-card__titles">
            <strong className="pipeline-flow-card__title">{stage.title}</strong>
          </div>
        </div>
      </button>
    )
  }

  return (
    <article className="engenharia-card training-pipeline-card">
      <header className="engenharia-card__head">
        <div>
          <span className="overview-eyebrow">
            <span />
            MLOps · Retreinamento Contínuo
          </span>
          <h2>Ciclo de Retreinamento e Governança</h2>
        </div>

        <div className="pipeline-header-actions">
          <div className="pipeline-meta-pill-group">
            <span className="pipeline-meta-pill">
              Job <strong>{job?.id ? job.id.slice(0, 8) : 'em definição'}</strong>
            </span>
            <span className="pipeline-meta-pill">
              Versão <strong>{job?.version || 'v1.1.0-cand'}</strong>
            </span>
            <span className={`badge-tag is-${effectiveStatus || 'candidate'}`}>
              {STATUS_LABELS[effectiveStatus] || 'Pronto'}
            </span>
          </div>

          {onTriggerRetrain ? (
            <button
              type="button"
              className="engenharia-trigger-btn"
              onClick={onTriggerRetrain}
              disabled={isActive}
            >
              <TriggerIcon />
              <span>{isActive ? 'Executando…' : 'Iniciar retreinamento'}</span>
            </button>
          ) : null}
        </div>
      </header>

      <div className="pipeline-progress-bar-track" aria-hidden="true">
        <div className="pipeline-progress-bar-fill" style={{ width: `${percent}%` }} />
      </div>

      <div className="engenharia-card__body pipeline-body">
        <div className="pipeline-flow-row">
          {firstRow.map((stage, idx) => (
            <div key={stage.key} className="pipeline-flow-cell">
              {renderCard(stage)}
              {idx < 4 ? (
                <div className={`pipeline-line-connector ${connectorState(stage, firstRow[idx + 1])}`} />
              ) : (
                <div className={`pipeline-vertical-connector ${connectorState(stage, secondRow[0])}`} />
              )}
            </div>
          ))}
        </div>

        <div className="pipeline-flow-row is-reverse">
          {secondRow.map((stage, idx) => {
            return (
              <div key={stage.key} className="pipeline-flow-cell">
                {renderCard(stage)}
                {idx < 4 ? (
                  <div className={`pipeline-line-connector ${connectorState(stage, secondRow[idx + 1])}`} />
                ) : null}
              </div>
            )
          })}
        </div>

        {inspectedStage ? (
          <div className="pipeline-stage-detail-panel">
            <div className="pipeline-stage-detail-panel__main">
              <div className="pipeline-stage-detail-panel__head">
                <span className="pipeline-stage-category-pill">{inspectedStage.category}</span>
                <h3>{inspectedStage.title}</h3>
              </div>
              <p className="pipeline-stage-detail-panel__desc">{inspectedStage.description}</p>
              {inspectedStage.detail ? (
                <p className="pipeline-stage-detail-panel__log">{inspectedStage.detail}</p>
              ) : null}
            </div>

            <div className="pipeline-stage-detail-panel__metrics">
              <div className="pipeline-stage-metric">
                <small>Resultado esperado / saída</small>
                <strong>{inspectedStage.output || '—'}</strong>
              </div>
              <div className="pipeline-stage-metric">
                <small>Duração</small>
                <strong>
                  {inspectedStage.duration_ms != null
                    ? durationLabel(inspectedStage.duration_ms)
                    : inspectedStage.status === 'running'
                    ? elapsedLabel(job?.started_at, job?.finished_at, now)
                    : inspectedStage.status === 'completed'
                    ? '0s'
                    : 'Aguardando'}
                </strong>
              </div>
            </div>
          </div>
        ) : null}

        {job?.status === 'failed' && !isVisualPlayback ? (
          <div className="pipeline-error-alert" role="alert">
            <AlertIcon />
            <div>
              <strong>O pipeline foi interrompido na etapa “{activeStage?.title || activeStage?.label}”.</strong>
              <p>{activeStage?.detail || job?.error || 'O serviço não informou detalhes adicionais do erro.'}</p>
            </div>
          </div>
        ) : null}
      </div>

      <footer className="engenharia-card__footer">
        <span>Separação temporal walk-forward (80/20) · Random Forest (400 árvores) · Relevância SHAP</span>
        <span>{completed} de {visibleStages.length} etapas concluídas ({percent}%)</span>
      </footer>
    </article>
  )
}
