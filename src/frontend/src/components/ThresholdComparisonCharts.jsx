import { useMemo, useState } from 'react'
import './ThresholdComparisonCharts.css'

function normalizeThresholdMetrics(metrics) {
  if (!Array.isArray(metrics)) return []
  return metrics
    .map((item) => ({
      ...item,
      threshold: Number(item.threshold),
      precision: Number(item.precision),
      recall: Number(item.recall),
      f1: Number(item.f1_score),
    }))
    .filter((item) => [item.threshold, item.precision, item.recall, item.f1].every(Number.isFinite))
    .sort((a, b) => a.threshold - b.threshold)
}

function nearestThresholdMetric(metrics, threshold) {
  if (!metrics.length) return null
  return metrics.reduce((nearest, item) => (
    Math.abs(item.threshold - threshold) < Math.abs(nearest.threshold - threshold) ? item : nearest
  ))
}

function formatPercent(value) {
  if (value == null || !Number.isFinite(Number(value))) return '—'
  return `${(Number(value) * 100).toFixed(1)}%`
}

function formatDelta(value, isPercent = true) {
  if (value == null) return '—'
  const num = Number(value)
  if (!Number.isFinite(num)) return '0'
  const sign = num > 0 ? '+' : ''
  if (isPercent) {
    return `${sign}${(num * 100).toFixed(1)}%`
  }
  return `${sign}${Math.round(num)}`
}

function TradeoffCurveChart({ selectedThreshold, officialThreshold, thresholdMetrics, onSelectThreshold }) {
  const [hoverTh, setHoverTh] = useState(null)
  const [mobileChartExpanded, setMobileChartExpanded] = useState(false)
  const activeTh = hoverTh ?? selectedThreshold
  const calibrationPoints = useMemo(() => normalizeThresholdMetrics(thresholdMetrics), [thresholdMetrics])

  const width = 760
  const height = 300
  const inset = { top: 34, right: 32, bottom: 46, left: 54 }
  const chartWidth = width - inset.left - inset.right
  const chartHeight = height - inset.top - inset.bottom

  const x = (th) => inset.left + ((Math.max(0.05, Math.min(0.95, th)) - 0.05) / 0.90) * chartWidth
  const y = (val) => inset.top + (1 - Math.max(0, Math.min(1, val))) * chartHeight

  const recallPolyline = calibrationPoints.map((p) => `${x(p.threshold).toFixed(1)},${y(p.recall).toFixed(1)}`).join(' ')
  const precisionPolyline = calibrationPoints.map((p) => `${x(p.threshold).toFixed(1)},${y(p.precision).toFixed(1)}`).join(' ')
  const f1Polyline = calibrationPoints.map((p) => `${x(p.threshold).toFixed(1)},${y(p.f1).toFixed(1)}`).join(' ')

  const activeMetrics = nearestThresholdMetric(calibrationPoints, activeTh)
  const bestMetric = calibrationPoints.reduce((best, item) => (!best || item.f1 > best.f1 ? item : best), null)

  const activeX = x(activeTh)
  const officialX = x(officialThreshold)

  function handlePointer(event) {
    const rect = event.currentTarget.getBoundingClientRect()
    const clickX = event.clientX - rect.left
    const scale = width / rect.width
    const scaledX = clickX * scale
    const normalized = (scaledX - inset.left) / chartWidth
    if (normalized >= 0 && normalized <= 1) {
      const th = Math.round((0.05 + normalized * 0.90) * 100) / 100
      return Math.max(0.05, Math.min(0.95, th))
    }
    return null
  }

  function handlePointerMove(event) {
    const th = handlePointer(event)
    if (th !== null) {
      setHoverTh(th)
    }
  }

  function handleClick(event) {
    const th = handlePointer(event)
    if (th !== null) {
      onSelectThreshold(th)
    }
  }

  const yTicks = [0, 0.25, 0.50, 0.75, 1.0]
  const xTicks = [...new Set([0.10, 0.30, Number(officialThreshold.toFixed(2)), 0.70, 0.90])].sort((a, b) => a - b)

  return (
    <article className="threshold-chart-card">
      <header className="threshold-chart-card__head">
        <div>
          <span className="lab-eyebrow">Validação temporal</span>
          <h3>Curvas de calibração walk-forward</h3>
        </div>
        <div className="threshold-chart-legend">
          <span className="legend-chip is-recall">
            <span className="legend-chip__dot" />
            Recall: <strong>{formatPercent(activeMetrics?.recall)}</strong>
          </span>
          <span className="legend-chip is-precision">
            <span className="legend-chip__dot" />
            Precisão: <strong>{formatPercent(activeMetrics?.precision)}</strong>
          </span>
          <span className="legend-chip is-f1">
            <span className="legend-chip__dot" />
            F1: <strong>{formatPercent(activeMetrics?.f1)}</strong>
          </span>
        </div>
      </header>

      <button
        type="button"
        className="threshold-chart-mobile-toggle"
        aria-expanded={mobileChartExpanded}
        aria-controls="threshold-calibration-plot"
        onClick={() => setMobileChartExpanded((expanded) => !expanded)}
      >
        {mobileChartExpanded ? 'Ocultar gráfico' : 'Ver gráfico de calibração'}
        <span aria-hidden="true">{mobileChartExpanded ? '−' : '+'}</span>
      </button>
      <div id="threshold-calibration-plot" className={`threshold-chart-svg-wrap${mobileChartExpanded ? ' is-expanded' : ''}`}>
        <svg
          viewBox={`0 0 ${width} ${height}`}
          className="threshold-chart-svg"
          onMouseMove={handlePointerMove}
          onMouseLeave={() => setHoverTh(null)}
          onClick={handleClick}
          role="img"
          aria-label="Curvas de sensibilidade de Recall, Precisão e F1-Score ao longo dos limiares de decisão"
        >
          {yTicks.map((tick) => (
            <g key={tick}>
              <line x1={inset.left} x2={width - inset.right} y1={y(tick)} y2={y(tick)} className="threshold-grid-line" />
              <text x={inset.left - 8} y={y(tick) + 4} textAnchor="end" className="threshold-axis-label">
                {Math.round(tick * 100)}%
              </text>
            </g>
          ))}

          {xTicks.map((tick) => (
            <g key={tick}>
              <line x1={x(tick)} x2={x(tick)} y1={inset.top} y2={height - inset.bottom} className="threshold-grid-line is-vertical" />
              <text x={x(tick)} y={height - inset.bottom + 16} textAnchor="middle" className="threshold-axis-label">
                {tick.toFixed(2)}
              </text>
            </g>
          ))}

          <line
            x1={officialX}
            x2={officialX}
            y1={inset.top}
            y2={height - inset.bottom}
            className="threshold-official-line"
          />
          <text
            x={officialX}
            y={inset.top + 12}
            textAnchor={officialX > width - 120 ? 'end' : 'start'}
            dx={officialX > width - 120 ? -6 : 6}
            className="threshold-official-label"
          >
            Operação ({officialThreshold.toFixed(2)})
          </text>

          {calibrationPoints.length ? <polyline points={recallPolyline} className="threshold-line is-recall" /> : null}
          {calibrationPoints.length ? <polyline points={precisionPolyline} className="threshold-line is-precision" /> : null}
          {calibrationPoints.length ? <polyline points={f1Polyline} className="threshold-line is-f1" /> : null}

          <line
            x1={activeX}
            x2={activeX}
            y1={inset.top}
            y2={height - inset.bottom}
            className="threshold-cursor-line"
          />

          {activeMetrics ? <circle cx={activeX} cy={y(activeMetrics.recall)} r="6" className="threshold-point is-recall" /> : null}
          {activeMetrics ? <circle cx={activeX} cy={y(activeMetrics.precision)} r="6" className="threshold-point is-precision" /> : null}
          {activeMetrics ? <circle cx={activeX} cy={y(activeMetrics.f1)} r="6" className="threshold-point is-f1" /> : null}

          <g transform={`translate(${activeX}, ${height - inset.bottom})`}>
            <rect x="-23" y="6" width="46" height="22" rx="4" className="threshold-badge-bg" />
            <text x="0" y="21" textAnchor="middle" className="threshold-badge-text">
              {Number(activeTh).toFixed(2)}
            </text>
          </g>
        </svg>
      </div>

      <footer className="threshold-chart-card__footer">
        <span>{bestMetric ? <>Maior F1 no teste: <strong>{bestMetric.threshold.toFixed(2)}</strong> (F1: {formatPercent(bestMetric.f1)}, Recall: {formatPercent(bestMetric.recall)}, Precisão: {formatPercent(bestMetric.precision)}).</> : 'Curva indisponível para este artefato legado.'}</span>
        <span>Mova o cursor para explorar. Clique para selecionar.</span>
      </footer>
    </article>
  )
}

function OperationalImpactChart({
  experiments,
  officialRun,
  officialThreshold,
  selectedThreshold,
  onSelectThreshold,
}) {
  const comparisonItems = useMemo(() => {
    const map = new Map()

    const offRisk = Number(officialRun?.total_doors_at_risk) || 0
    const offTotal = Number(officialRun?.total_doors_evaluated) || 0
    const offTh = Number(officialRun?.threshold_used ?? officialThreshold)
    map.set(Number(offTh.toFixed(2)), {
      threshold: offTh,
      doorsAtRisk: offRisk,
      totalDoors: offTotal,
      isOfficial: true,
      label: 'Operação ativa',
      runId: officialRun?.id || 'oficial',
    })

    if (Array.isArray(experiments)) {
      experiments.forEach((run) => {
        const th = Number(Number(run.threshold_used).toFixed(2))
        if (!Number.isFinite(th)) return
        const risk = Number(run.total_doors_at_risk) || 0
        const total = Number(run.total_doors_evaluated) || 0
        const existing = map.get(th)
        if (!existing || (!existing.isOfficial && !existing.isCurrentSelection)) {
          map.set(th, {
            threshold: th,
            doorsAtRisk: risk,
            totalDoors: total,
            isOfficial: false,
            label: `Teste ${run.id?.slice(0, 8) || ''}`,
            runId: run.id,
          })
        }
      })
    }

    const items = Array.from(map.values())
    if (items.length > 7) {
      const relevant = items
        .sort((a, b) => {
          if (a.isOfficial !== b.isOfficial) return a.isOfficial ? -1 : 1
          if (a.isSimulated !== b.isSimulated) return a.isSimulated ? -1 : 1
          return Math.abs(a.threshold - selectedThreshold) - Math.abs(b.threshold - selectedThreshold)
        })
        .slice(0, 7)
      return relevant.sort((a, b) => a.threshold - b.threshold)
    }
    return items.sort((a, b) => a.threshold - b.threshold)
  }, [experiments, officialRun, officialThreshold, selectedThreshold])

  const maxDoors = Math.max(
    5,
    ...comparisonItems.map((item) => item.doorsAtRisk),
    Number(officialRun?.total_doors_at_risk) || 0
  )
  const officialDoors = Number(officialRun?.total_doors_at_risk) || 0
  const officialTotal = Number(officialRun?.total_doors_evaluated) || 0

  return (
    <article className="threshold-chart-card">
      <header className="threshold-chart-card__head">
        <div>
          <span className="lab-eyebrow">Frota Linha 4</span>
          <h3>Portas em risco por limiar</h3>
        </div>
        <div className="threshold-chart-legend">
          <span className="legend-chip is-official">
            <span className="legend-chip__dot" />
            Operação ({officialDoors} portas)
          </span>
          <span className="legend-chip is-selected">
            <span className="legend-chip__dot" />
            Selecionado ({Number(selectedThreshold).toFixed(2)})
          </span>
        </div>
      </header>

      <div className="threshold-impact-list" role="list" aria-label="Portas em risco por threshold">
        {comparisonItems.map((item) => {
          const isSelected = Math.abs(item.threshold - selectedThreshold) < 0.005
          const alertRate = item.totalDoors ? (item.doorsAtRisk / item.totalDoors) * 100 : 0
          const width = maxDoors ? Math.max(2, (item.doorsAtRisk / maxDoors) * 100) : 2
          return (
            <button
              type="button"
              role="listitem"
              key={`${item.runId}-${item.threshold}`}
              className={`threshold-impact-row${isSelected ? ' is-selected' : ''}${item.isOfficial ? ' is-official' : ''}`}
              onClick={() => onSelectThreshold(item.threshold)}
            >
              <span className="threshold-impact-row__threshold">{item.threshold.toFixed(2)}</span>
              <span className="threshold-impact-row__visual">
                <span className="threshold-impact-row__track"><i style={{ width: `${width}%` }} /></span>
                <span>{item.isOfficial ? 'Oficial' : isSelected ? 'Selecionado' : 'Cenário'}</span>
              </span>
              <strong>{item.doorsAtRisk}</strong>
              <small>{alertRate.toFixed(1)}% da frota</small>
            </button>
          )
        })}
      </div>

      <footer className="threshold-chart-card__footer">
        <span>Linha de referência: operação atual com <strong>{officialDoors}</strong> portas em risco de {officialTotal || '—'} portas avaliadas.</span>
        <span>Clique em uma linha para carregar o threshold correspondente.</span>
      </footer>
    </article>
  )
}

function DeltaCard({ selectedThreshold, officialThreshold, officialRun, experiments, thresholdMetrics }) {
  const metrics = normalizeThresholdMetrics(thresholdMetrics)
  const selectedMetrics = nearestThresholdMetric(metrics, selectedThreshold)
  const officialMetrics = nearestThresholdMetric(metrics, officialThreshold)

  const deltaRecall = selectedMetrics && officialMetrics ? selectedMetrics.recall - officialMetrics.recall : null
  const deltaPrecision = selectedMetrics && officialMetrics ? selectedMetrics.precision - officialMetrics.precision : null
  const deltaF1 = selectedMetrics && officialMetrics ? selectedMetrics.f1 - officialMetrics.f1 : null

  const matchingRun = Array.isArray(experiments)
    ? experiments.find((run) => Math.abs(Number(run.threshold_used) - selectedThreshold) < 0.005)
    : null

  const officialRisk = Number(officialRun?.total_doors_at_risk) || 0
  const selectedRisk = matchingRun ? Number(matchingRun.total_doors_at_risk) || 0 : null
  const deltaDoors = selectedRisk !== null ? selectedRisk - officialRisk : null

  return (
    <div className="threshold-delta-panel">
      <div className="threshold-delta-item">
        <span className="threshold-delta-label">Limiar em simulação</span>
        <strong className="threshold-delta-value">{Number(selectedThreshold).toFixed(2)}</strong>
        <small>Operação ativa: {Number(officialThreshold).toFixed(2)}</small>
      </div>

      <div className="threshold-delta-item">
        <span className="threshold-delta-label">Variação de Recall</span>
        <strong className={`threshold-delta-value ${deltaRecall != null && deltaRecall >= 0 ? 'is-positive' : 'is-negative'}`}>
          {formatDelta(deltaRecall)}
        </strong>
        <small>{formatPercent(selectedMetrics?.recall)} vs {formatPercent(officialMetrics?.recall)}</small>
      </div>

      <div className="threshold-delta-item">
        <span className="threshold-delta-label">Variação de Precisão</span>
        <strong className={`threshold-delta-value ${deltaPrecision != null && deltaPrecision >= 0 ? 'is-positive' : 'is-warning'}`}>
          {formatDelta(deltaPrecision)}
        </strong>
        <small>{formatPercent(selectedMetrics?.precision)} vs {formatPercent(officialMetrics?.precision)}</small>
      </div>

      <div className="threshold-delta-item">
        <span className="threshold-delta-label">Variação de F1-Score</span>
        <strong className={`threshold-delta-value ${deltaF1 != null && deltaF1 >= 0 ? 'is-positive' : 'is-neutral'}`}>
          {formatDelta(deltaF1)}
        </strong>
        <small>{formatPercent(selectedMetrics?.f1)} vs {formatPercent(officialMetrics?.f1)}</small>
      </div>

      <div className="threshold-delta-item">
        <span className="threshold-delta-label">Portas em risco</span>
        <strong className="threshold-delta-value">
          {selectedRisk !== null ? selectedRisk : '—'}
        </strong>
        <small>
          {deltaDoors !== null
            ? `${formatDelta(deltaDoors, false)} versus oficial (${officialRisk})`
            : `Oficial ativo: ${officialRisk}`}
        </small>
      </div>
    </div>
  )
}

export function ThresholdComparisonCharts({
  selectedThreshold,
  officialThreshold,
  officialRun,
  experiments,
  thresholdMetrics,
  onSelectThreshold,
}) {
  return (
    <section className="threshold-comparison-section" aria-label="Comparativo visual de mudanças de threshold">
      <div className="lab-visuals__intro">
        <span className="lab-eyebrow">Análise visual</span>
        <h2>Comparativo de sensibilidade e impacto dos limiares</h2>
        <p>Acompanhe o equilíbrio entre captura de falhas e falsos alertas, e compare o volume de portas sinalizadas em risco na operação.</p>
      </div>

      <DeltaCard
        selectedThreshold={selectedThreshold}
        officialThreshold={officialThreshold}
        officialRun={officialRun}
        experiments={experiments}
        thresholdMetrics={thresholdMetrics}
      />

      <div className="threshold-comparison-grid">
        <TradeoffCurveChart
          selectedThreshold={selectedThreshold}
          officialThreshold={officialThreshold}
          thresholdMetrics={thresholdMetrics}
          onSelectThreshold={onSelectThreshold}
        />

        <OperationalImpactChart
          experiments={experiments}
          officialRun={officialRun}
          officialThreshold={officialThreshold}
          selectedThreshold={selectedThreshold}
          onSelectThreshold={onSelectThreshold}
        />
      </div>
    </section>
  )
}
