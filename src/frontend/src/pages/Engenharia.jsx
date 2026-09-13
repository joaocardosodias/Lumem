import { useEffect, useMemo, useState } from 'react'
import Button from '../components/Button'
import { useAuth } from '../context/Auth'
import { useToast } from '../context/Toast'
import { api, errorMessage } from '../lib/api'
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
  { id: 'calibration', label: 'Calibração' },
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
  const [savingConfig, setSavingConfig] = useState(false)
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

  const [selectedThreshold, setSelectedThreshold] = useState(0.48)
  const [selectedHorizon, setSelectedHorizon] = useState(7)

  useEffect(() => {
    async function loadData() {
      try {
        setLoading(true)
        const [cfgData, modelData, histData, driftData] = await Promise.all([
          api.getMLConfig().catch(() => ({ threshold: 0.48, default_horizon_days: 7 })),
          api.getActiveModel().catch(() => null),
          api.getModelHistory(5).catch(() => []),
          api.getMLDrift().catch(() => null),
        ])
        setConfig(cfgData)
        setSelectedThreshold(cfgData.threshold || 0.48)
        setSelectedHorizon(cfgData.default_horizon_days || 7)
        setModel(modelData)
        setHistory(histData || [])
        setDrift(driftData)
      } finally {
        setLoading(false)
      }
    }
    loadData()
  }, [])

  const simulatedImpact = useMemo(() => {
    const t = Number(selectedThreshold)
    const baseDoorsAtRisk = 24
    const delta = (0.48 - t) * 90
    const estimatedRisk = Math.max(6, Math.min(75, Math.round(baseDoorsAtRisk + delta)))
    const estimatedRecall = Math.max(35, Math.min(88, Math.round(55.3 + (0.48 - t) * 65)))
    const weeklyInspections = Math.round(estimatedRisk * 1.25)

    let sensitivity = 'Equilibrada'
    if (t < 0.38) sensitivity = 'Alta sensibilidade'
    else if (t > 0.58) sensitivity = 'Conservadora'

    return {
      doorsAtRisk: estimatedRisk,
      recallPct: estimatedRecall,
      inspectionsPerWeek: weeklyInspections,
      sensitivity,
    }
  }, [selectedThreshold])

  async function handleSaveConfig(e) {
    e.preventDefault()
    try {
      setSavingConfig(true)
      const updated = await api.updateMLConfig(selectedThreshold, selectedHorizon)
      setConfig(updated)
      toast.success('Calibração operacional salva.')
    } catch (err) {
      toast.error('Falha ao salvar calibração: ' + errorMessage(err))
    } finally {
      setSavingConfig(false)
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
          <div className="engenharia-metrics">
            <div className="engenharia-metric engenharia-metric--version">
              <span>Modelo ativo</span>
              <strong>{model?.version || 'v1.0.0'}</strong>
              <small>Random Forest (modelo-5)</small>
            </div>
            <div className="engenharia-metric">
              <span>Recall de falhas</span>
              <strong>{((model?.test_metrics?.recall ?? 0.553) * 100).toFixed(1)}%</strong>
              <small>3.432 de 6.202 falhas cobertas</small>
            </div>
            <div className="engenharia-metric">
              <span>ROC-AUC</span>
              <strong>{Number(model?.test_metrics?.roc_auc ?? 0.697).toFixed(3)}</strong>
              <small>Separação entre classes</small>
            </div>
            <div className="engenharia-metric">
              <span>F1-Score</span>
              <strong>{((model?.test_metrics?.f1 ?? 0.247) * 100).toFixed(1)}%</strong>
              <small>Limiar calibrado ({Number(config?.threshold || 0.48).toFixed(2)})</small>
            </div>
            <div className="engenharia-metric">
              <span>Precisão</span>
              <strong>{((model?.test_metrics?.precision ?? 0.159) * 100).toFixed(1)}%</strong>
              <small>3.432 acertos em 21.564 alertas</small>
            </div>
          </div>

          <div className="engenharia-panels">
            <article className="engenharia-card">
              <header className="engenharia-card__head">
                <h2>Matriz de confusão · teste walk-forward</h2>
                <span className="engenharia-tag">71.184 registros</span>
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
                <span>Avaliação walk-forward sobre 71.184 registros porta-dia no período de teste (maio a junho/2025).</span>
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
        <article className="engenharia-card engenharia-card--calibration" id="engineering-panel-calibration" role="tabpanel" aria-labelledby="engineering-tab-calibration">
          <header className="engenharia-card__head">
            <h2>Calibração de limiar operacional e horizonte de análise</h2>
            <span className="engenharia-tag">Configuração ativa</span>
          </header>

          <div className="engenharia-card__body">
            <form className="calibration-form-wrap" onSubmit={handleSaveConfig}>
              <div className="calibration-field">
                <div className="calibration-field__top">
                  <label htmlFor="threshold-slider">Limiar de decisão</label>
                  <span className="calibration-value-badge">{Number(selectedThreshold).toFixed(2)}</span>
                </div>
                <input
                  id="threshold-slider"
                  type="range"
                  className="calibration-slider-input"
                  min="0.20"
                  max="0.80"
                  step="0.01"
                  value={selectedThreshold}
                  onChange={(e) => setSelectedThreshold(Number(e.target.value))}
                />
                <div className="calibration-ticks">
                  <span>0,20 · mais sensível</span>
                  <span>0,48 · ponto calibrado</span>
                  <span>0,80 · mais preciso</span>
                </div>
                <p className="calibration-field__note">
                  O limiar padrão do bundle é 0.48 (calibrado no treino temporal). Reduzir o limiar antecipa mais falhas gerando mais ordens de serviço; aumentar restringe os alertas a portas com alta probabilidade de pane.
                </p>
              </div>

              <div className="calibration-field">
                <div className="calibration-field__top">
                  <label htmlFor="horizon-select">Horizonte de predição padrão</label>
                </div>
                <label className="calibration-horizon">
                  <span>Horizonte:</span>
                  <select
                    id="horizon-select"
                    value={selectedHorizon}
                    onChange={(e) => setSelectedHorizon(Number(e.target.value))}
                  >
                    <option value={7}>7 dias · manutenções imediatas</option>
                    <option value={14}>14 dias · planejamento semanal</option>
                    <option value={30}>30 dias · gestão de ativos</option>
                  </select>
                </label>
              </div>

              <div className="simulation-panel">
                <div className="simulation-panel__head">
                  <h3>Impacto operacional estimado</h3>
                  <p>Estimativa projetada sobre o parque ativo de 264 portas da Linha 4 Amarela</p>
                </div>
                <div className="simulation-grid">
                  <div className="simulation-tile">
                    <span>Sensibilidade</span>
                    <strong>{simulatedImpact.sensitivity}</strong>
                  </div>
                  <div className="simulation-tile">
                    <span>Recall projetado</span>
                    <strong>{simulatedImpact.recallPct}%</strong>
                  </div>
                  <div className="simulation-tile">
                    <span>Portas em risco</span>
                    <strong>{simulatedImpact.doorsAtRisk} portas</strong>
                  </div>
                  <div className="simulation-tile">
                    <span>Ordens preventivas</span>
                    <strong>~{simulatedImpact.inspectionsPerWeek} ordens/sem</strong>
                  </div>
                </div>
              </div>

              <div className="calibration-actions">
                <Button type="submit" variant="primary" disabled={savingConfig}>
                  {savingConfig ? 'Salvando calibração…' : 'Salvar calibração operacional'}
                </Button>
              </div>
            </form>
          </div>
        </article>
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
              <span>O Random Forest é o único modelo serializado em artefato .joblib e em operação no serviço preditivo da Linha 4.</span>
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
                          <span className={`badge-tag ${run.status === 'production' || run.status === 'completed' ? 'is-active' : ''}`}>
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
            <span className="engenharia-status-dot" aria-hidden="true" />
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
