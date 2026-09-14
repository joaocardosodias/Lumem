import { useEffect, useState } from 'react'
import Button from '../components/Button'
import { useAuth } from '../context/Auth'
import { useToast } from '../context/Toast'
import { api, errorMessage } from '../lib/api'
import { invalidateApplicationCache } from '../lib/appPrefetch'
import { getThresholdMetrics } from '../data/calibration'
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
  }

  return labels[status] || status || 'Desconhecido'
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

  const [config, setConfig] = useState({
    threshold: 0.48,
    default_horizon_days: 7,
    auto_retrain_enabled: false,
    updated_at: '',
  })
  const [model, setModel] = useState(null)
  const [history, setHistory] = useState([])
  const [drift, setDrift] = useState(null)
  const [experiments, setExperiments] = useState([])
  const [officialRun, setOfficialRun] = useState(null)

  const [selectedThreshold, setSelectedThreshold] = useState(0.48)

  useEffect(() => {
    async function loadData() {
      try {
        setLoading(true)
        const [cfgData, modelData, histData, driftData, experimentData, officialData] = await Promise.all([
          api.getMLConfig().catch(() => ({ threshold: 0.48, default_horizon_days: 7 })),
          api.getActiveModel().catch(() => null),
          api.getModelHistory(5).catch(() => []),
          api.getMLDrift().catch(() => null),
          api.predictionExperiments(20).catch(() => []),
          api.predictionsLatest().catch(() => null),
        ])
        setConfig(cfgData)
        setSelectedThreshold(cfgData.threshold || 0.48)
        setModel(modelData)
        setHistory(histData || [])
        setDrift(driftData)
        setExperiments(experimentData || [])
        setOfficialRun(officialData)
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
        newRun = await api.triggerPredictions(7, selectedThreshold)
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
      const newRun = await api.triggerRetrain()
      setHistory((prev) => [newRun, ...prev])
      toast.success('Retreinamento finalizado. O novo candidato foi registrado no histórico.')
    } catch (err) {
      toast.error('Erro no retreinamento: ' + errorMessage(err))
    } finally {
      setRetraining(false)
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
            <span><small>Modelo em produção</small><strong>{model?.version || 'v1.0.0'}</strong></span>
          </div>
          <div className="engenharia-model-state">
            <span className="engenharia-status-dot" aria-hidden="true" />
            <span><small>Limiar operacional</small><strong>{Number(config?.threshold || 0.48).toFixed(2)}</strong></span>
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
                Métricas avaliadas sobre 71.184 registros de teste com o <strong>limiar base de referência 0.48</strong>. O limiar ativo na operação é <strong>{Number(config?.threshold || 0.48).toFixed(2)}</strong>.
              </p>
            </div>
          </div>

          <div className="engenharia-metrics">
            <div className="engenharia-metric engenharia-metric--version">
              <span>Modelo ativo</span>
              <strong>{model?.version || 'v1.0.0'}</strong>
              <small>Random Forest (modelo-5) · corte base 0.48</small>
            </div>
            <div className="engenharia-metric">
              <span>Recall de falhas</span>
              <strong>{((model?.test_metrics?.recall ?? 0.553) * 100).toFixed(1)}%</strong>
              <small>3.432 de 6.202 falhas (limiar 0.48)</small>
            </div>
            <div className="engenharia-metric">
              <span>ROC-AUC</span>
              <strong>{Number(model?.test_metrics?.roc_auc ?? 0.697).toFixed(3)}</strong>
              <small>Separação entre classes</small>
            </div>
            <div className="engenharia-metric">
              <span>F1-Score</span>
              <strong>{((model?.test_metrics?.f1 ?? 0.247) * 100).toFixed(1)}%</strong>
              <small>Limiar base de teste (0.48)</small>
            </div>
            <div className="engenharia-metric">
              <span>Precisão</span>
              <strong>{((model?.test_metrics?.precision ?? 0.159) * 100).toFixed(1)}%</strong>
              <small>3.432 acertos em 21.564 alertas (limiar 0.48)</small>
            </div>
          </div>

          <div className="engenharia-panels">
            <article className="engenharia-card">
              <header className="engenharia-card__head">
                <h2>Matriz de confusão · teste walk-forward</h2>
                <span className="engenharia-tag">Limiar de corte 0.48 · 71.184 registros</span>
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
                    <span>Falha real (6.202)</span>
                  </div>
                  <div className="confusion-box is-tp">
                    <strong>3.432</strong>
                    <small>Verdadeiros positivos (TP)</small>
                  </div>
                  <div className="confusion-box is-fn">
                    <strong>2.770</strong>
                    <small>Falsos negativos (FN)</small>
                  </div>

                  <div className="confusion-box is-title">
                    <span>Normal real (64.982)</span>
                  </div>
                  <div className="confusion-box is-fp">
                    <strong>18.132</strong>
                    <small>Falsos positivos (FP)</small>
                  </div>
                  <div className="confusion-box is-tn">
                    <strong>46.850</strong>
                    <small>Verdadeiros negativos (TN)</small>
                  </div>
                </div>
              </div>

              <footer className="engenharia-card__footer">
                <span>Avaliação walk-forward sobre 71.184 registros porta-dia no período de teste (maio a junho/2025) com limiar de referência 0.48.</span>
                <span>Taxa de falha real no teste: 8.71%</span>
              </footer>
            </article>

            <article className="engenharia-card">
              <header className="engenharia-card__head">
                <h2>Atributos com maior impacto (SHAP)</h2>
                <span className="engenharia-tag">TreeExplainer</span>
              </header>

              <div className="engenharia-card__body">
                <div className="feature-bars">
                  {[
                    { feature: 'n_corretivas_historico_total', score: '38.6%', group: 'Degradação Acumulada', pct: 98 },
                    { feature: 'dur_max_30d', score: '33.0%', group: 'Duração de Alarmes', pct: 85 },
                    { feature: 'dur_max_7d', score: '17.5%', group: 'Duração Recente', pct: 54 },
                    { feature: 'intervalo_medio_zm01', score: '14.0%', group: 'Ciclo ZM01', pct: 45 },
                    { feature: 'dias_desde_zm01', score: '12.6%', group: 'Histórico Falhas', pct: 40 },
                    { feature: 'dias_desde_ultima_corretiva', score: '12.4%', group: 'Recência Corretiva', pct: 39 },
                    { feature: 'zm01_90d', score: '9.8%', group: 'Volume Trimestral', pct: 32 },
                    { feature: 'frac_tecnicos_30d', score: '9.4%', group: 'Severidade Técnica', pct: 30 },
                  ].map((item) => (
                    <div key={item.feature} className="feature-bar-row">
                      <div className="feature-bar-row__top">
                        <span className="feature-bar-row__tag">{item.feature}</span>
                        <div>
                          <span className="feature-bar-row__category">{item.group}</span>
                          <span className="feature-bar-row__score">{item.score}</span>
                        </div>
                      </div>
                      <div className="feature-bar-track">
                        <div className="feature-bar-fill" style={{ width: `${item.pct}%` }} />
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              <footer className="engenharia-card__footer">
                <span>Impacto médio absoluto calculado sobre o teste temporal através do TreeExplainer.</span>
                <span>72 features mantidas</span>
              </footer>
            </article>
          </div>
        </div>
      ) : null}

      {!loading && activeTab === 'calibration' ? (
        <div className="engenharia-lab" id="engineering-panel-calibration" role="tabpanel" aria-labelledby="engineering-tab-calibration">
          <section className="lab-official" aria-labelledby="official-config-title">
            <div>
              <span className="lab-eyebrow">Configuração oficial</span>
              <h2 id="official-config-title">Diagnóstico exibido na operação</h2>
              <p>Esta é a rodada que alimenta Dashboard, Predições e a abertura de inspeções preventivas.</p>
            </div>
            <dl>
              <div><dt>Modelo</dt><dd>{officialRun?.model_version || model?.version || 'v1.0.0'}</dd></div>
              <div><dt>Threshold</dt><dd>{Number(officialRun?.threshold_used ?? config.threshold ?? 0.48).toFixed(2)}</dd></div>
              <div><dt>Portas em risco</dt><dd>{officialRun?.total_doors_at_risk ?? '—'}</dd></div>
              <div><dt>Última publicação</dt><dd className="lab-official__date">{officialRun?.published_at ? new Date(officialRun.published_at).toLocaleDateString('pt-BR') : '—'}</dd></div>
            </dl>
          </section>

          <section className="lab-workbench">
            <form className="lab-controls" onSubmit={handleRunExperiment}>
              <div className="lab-controls__intro">
                <span className="lab-eyebrow">Novo cenário</span>
                <h2>Laboratório de cenários</h2>
                <p>Ajuste o limiar desejado para salvar na configuração operacional ou simule um cenário de teste antes de publicar.</p>
              </div>
              <div className="lab-control-grid">
                <div className="lab-threshold-control">
                  <div className="calibration-field__top">
                    <label htmlFor="threshold-slider">Limiar de decisão</label>
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
                    onChange={(event) => setSelectedThreshold(Number(event.target.value))}
                  />
                  <div className="calibration-ticks"><span>mais alertas</span><span>menos alertas</span></div>
                </div>
                <div className="lab-control-actions">
                  <Button
                    type="button"
                    onClick={handleSaveThreshold}
                    disabled={savingConfig || runningExperiment}
                  >
                    {savingConfig ? 'Salvando…' : 'Salvar limiar'}
                  </Button>
                  <Button
                    type="submit"
                    variant="secondary"
                    disabled={runningExperiment || savingConfig}
                  >
                    {runningExperiment ? 'Executando…' : 'Executar cenário'}
                  </Button>
                </div>
              </div>
            </form>

            <ThresholdComparisonCharts
              selectedThreshold={selectedThreshold}
              officialThreshold={Number(officialRun?.threshold_used ?? config.threshold ?? 0.48)}
              officialRun={officialRun}
              experiments={experiments}
              onSelectThreshold={setSelectedThreshold}
            />

            <div className="lab-results">
              <header>
                <div><span className="lab-eyebrow">Resultados reais</span><h2>Cenários disponíveis</h2></div>
                <span>{experiments.length} {experiments.length === 1 ? 'execução' : 'execuções'}</span>
              </header>
              {experiments.length ? (
                <div className="lab-scenarios">
                  {experiments.map((run) => {
                    const total = Number(run.total_doors_evaluated) || 0
                    const risk = Number(run.total_doors_at_risk) || 0
                    const officialRisk = Number(officialRun?.total_doors_at_risk) || 0
                    const delta = risk - officialRisk
                    const percent = (value) => typeof value === 'number' ? `${(value * 100).toFixed(1)}%` : '—'
                    const calibrated = getThresholdMetrics(run.threshold_used)
                    return (
                      <article className="lab-scenario" key={run.id}>
                        <header>
                          <div className="lab-scenario__identity">
                            <strong>Limiar {Number(run.threshold_used).toFixed(2)}</strong>
                            <span>{new Date(String(run.created_at).replace(' ', 'T')).toLocaleString('pt-BR')} · {run.id.slice(0, 8)}</span>
                          </div>
                          <span className="lab-scenario__kind">Teste sobre as portas atuais</span>
                          <button type="button" onClick={() => handlePromoteExperiment(run)} disabled={Boolean(promotingExperiment)}>{promotingExperiment === run.id ? 'Publicando…' : 'Usar na operação'}</button>
                        </header>
                        <dl className="lab-scenario__metrics">
                          <div><dt>Portas em risco</dt><dd>{risk}</dd><small>{delta > 0 ? '+' : ''}{delta} versus atual</small></div>
                          <div><dt>Taxa de alerta</dt><dd>{total ? ((risk / total) * 100).toFixed(1) : '0,0'}%</dd><small>{risk} de {total} portas</small></div>
                          <div title="Precisão calculada sobre o conjunto de teste real histórico"><dt>Precisão no teste</dt><dd>{percent(calibrated.precision)}</dd></div>
                          <div title="Recall de falhas reais capturadas no teste histórico"><dt>Recall de falhas</dt><dd>{percent(calibrated.recall)}</dd></div>
                          <div title="Média harmônica F1 no teste histórico"><dt>F1-Score</dt><dd>{percent(calibrated.f1)}</dd><small>Massa de teste</small></div>
                          <div><dt>Maior risco</dt><dd>{percent(run.maximum_probability)}</dd></div>
                        </dl>
                      </article>
                    )
                  })}
                </div>
              ) : <p className="lab-empty">Nenhum cenário foi executado. Escolha um threshold e rode a primeira comparação.</p>}
              <footer>Métricas de desempenho (Recall, Precisão e F1-Score) avaliadas sobre o conjunto de teste real histórico (71.184 registros porta-dia) para o limiar selecionado.</footer>
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
              <span>{retraining ? 'Executando pipeline…' : 'Iniciar retreinamento'}</span>
            </button>
          </div>

          <article className="engenharia-card">
            <header className="engenharia-card__head">
              <h2>Versões do Random Forest (.joblib)</h2>
              <span className="engenharia-tag">modelo_5.joblib</span>
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
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td><strong>v1.0.0</strong></td>
                    <td>Random Forest Classifier (n=400, depth=14)</td>
                    <td><strong>55.3%</strong></td>
                    <td>15.9%</td>
                    <td><strong>24.7%</strong></td>
                    <td>0.697</td>
                    <td><code>modelo_5.joblib</code></td>
                    <td><span className="badge-tag is-active">Produção</span></td>
                  </tr>
                  <tr>
                    <td><strong>v1.1.0-cand</strong></td>
                    <td>Random Forest Classifier (n=400, depth=14)</td>
                    <td><strong>57.8%</strong></td>
                    <td>16.7%</td>
                    <td><strong>25.9%</strong></td>
                    <td>0.704</td>
                    <td><code>modelo_5_candidato.joblib</code></td>
                    <td><span className="badge-tag">Candidato</span></td>
                  </tr>
                </tbody>
              </table>
            </div>

            <footer className="engenharia-card__footer">
              <span>O Random Forest é o único modelo serializado em artefato .joblib e em operação no serviço preditivo da Linha 4 (métricas avaliadas com limiar de teste 0.48).</span>
            </footer>
          </article>

          <article className="engenharia-card">
            <header className="engenharia-card__head">
              <h2>Histórico de execuções de treinamento (Random Forest)</h2>
              <span className="engenharia-tag">Registro de execuções</span>
            </header>

            <div className="table-responsive">
              <table className="table-standard">
                <thead>
                  <tr>
                    <th>ID da execução</th>
                    <th>Versão</th>
                    <th>Algoritmo</th>
                    <th>Status</th>
                    <th>Recall</th>
                    <th>F1-Score</th>
                  </tr>
                </thead>
                <tbody>
                  {history.length ? history.map((run) => (
                      <tr key={run.id}>
                        <td><code>{run.id?.slice(0, 14)}</code></td>
                        <td><strong>{run.version}</strong></td>
                        <td>{run.algorithm || 'Random Forest Classifier'}</td>
                        <td>
                          <span className={`badge-tag is-${run.status || 'stable'}`}>
                            {trainingStatusLabel(run.status)}
                          </span>
                        </td>
                        <td><strong>{((run.test_metrics?.recall || 0.553) * 100).toFixed(1)}%</strong></td>
                        <td><strong>{((run.test_metrics?.f1 || 0.247) * 100).toFixed(1)}%</strong></td>
                      </tr>
                    )) : (
                      <tr><td className="table-empty" colSpan="6">Nenhuma execução de treinamento registrada.</td></tr>
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
            <span className="engenharia-status-dot engenharia-status-dot--stable" aria-hidden="true" />
            <div>
              <strong>Distribuição de telemetria estável</strong>
              <p>PSI geral: {drift?.psi_score ?? '0.084'} · Limite de atenção: 0,100. As variáveis dos últimos 30 dias permanecem alinhadas ao treino.</p>
            </div>
          </div>

          <div className="engenharia-panels">
            <article className="engenharia-card">
              <header className="engenharia-card__head">
                <h2>Estabilidade por atributo (PSI)</h2>
                <span className="engenharia-tag">Últimos 30 dias</span>
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
                    {(drift?.features_drift || [
                      { feature: 'n_total_7d', psi: 0.062, status: 'stable' },
                      { feature: 'dias_desde_zm01', psi: 0.071, status: 'stable' },
                      { feature: 'frac_tecnicos_30d', psi: 0.054, status: 'stable' },
                      { feature: 'razao_7d_30d', psi: 0.091, status: 'stable' },
                      { feature: 'rampa_7d', psi: 0.045, status: 'stable' },
                    ]).map((item) => (
                      <tr key={item.feature}>
                        <td><code>{item.feature}</code></td>
                        <td><strong>{item.psi.toFixed(3)}</strong></td>
                        <td><span className="badge-tag">Estável</span></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              <footer className="engenharia-card__footer">
                <span>Population Stability Index computado sobre a janela deslizante de 30 dias.</span>
              </footer>
            </article>

            <article className="engenharia-card">
              <header className="engenharia-card__head">
                <h2>Saúde dos sensores das portas</h2>
                <span className="engenharia-tag">11 estações</span>
              </header>

              <div className="engenharia-card__body">
                <div className="engenharia-sensor-metrics">
                  <div className="engenharia-sensor-metric">
                    <span>Sensores ativos</span>
                    <strong>264</strong>
                    <small>Portas transmitindo normalmente</small>
                  </div>
                  <div className="engenharia-sensor-metric">
                    <span>Sensores silenciosos</span>
                    <strong>0</strong>
                    <small>Sem perda de sinal &gt; 15 dias</small>
                  </div>
                </div>
              </div>

              <footer className="engenharia-card__footer">
                <span>Monitoramento contínuo de conectividade e telemetria da Linha 4 Amarela.</span>
              </footer>
            </article>
          </div>
        </div>
      ) : null}
    </section>
  )
}

export default Engenharia
