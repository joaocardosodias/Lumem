import './PredictionHistory.css'

function parseDate(value) {
  if (!value) return null
  const normalized = String(value).includes('T') ? String(value) : String(value).replace(' ', 'T')
  const date = new Date(normalized)
  return Number.isNaN(date.getTime()) ? null : date
}

function percent(item) {
  const value = Number(item?.probability)
  if (Number.isFinite(value)) return Math.max(0, Math.min(1, value))
  return item?.will_fail ? 1 : 0
}

function formatPercent(value) {
  return `${Math.round(value * 100)}%`
}

function formatDate(value, withTime = false) {
  const date = parseDate(value)
  if (!date) return '—'
  return new Intl.DateTimeFormat('pt-BR', {
    day: '2-digit',
    month: 'short',
    ...(withTime ? { hour: '2-digit', minute: '2-digit' } : {}),
  }).format(date)
}

function RiskChart({ history }) {
  const points = history
    .filter((item) => item.probability != null && Number.isFinite(Number(item.probability)))
    .reverse()
  if (!points.length) {
    return <p className="prediction-chart-empty">O gráfico começará a ser preenchido na próxima execução, quando a probabilidade passar a ser registrada.</p>
  }
  const width = 760
  const height = 190
  const inset = { top: 18, right: 22, bottom: 38, left: 44 }
  const chartWidth = width - inset.left - inset.right
  const chartHeight = height - inset.top - inset.bottom
  const x = (index) => inset.left + (points.length === 1 ? chartWidth / 2 : (index / (points.length - 1)) * chartWidth)
  const y = (value) => inset.top + (1 - value) * chartHeight
  const polyline = points.map((item, index) => `${x(index)},${y(percent(item))}`).join(' ')
  const threshold = Number(points.at(-1)?.threshold_used)
  const safeThreshold = Number.isFinite(threshold) ? threshold : 0.28

  return (
    <div className="prediction-history-chart">
      <svg viewBox={`0 0 ${width} ${height}`} role="img" aria-label="Evolução da probabilidade de falha desta porta">
        {[0, 0.5, 1].map((tick) => (
          <g key={tick}>
            <line x1={inset.left} x2={width - inset.right} y1={y(tick)} y2={y(tick)} className="prediction-chart-grid" />
            <text x={inset.left - 10} y={y(tick) + 4} textAnchor="end" className="prediction-chart-label">{formatPercent(tick)}</text>
          </g>
        ))}
        <line x1={inset.left} x2={width - inset.right} y1={y(safeThreshold)} y2={y(safeThreshold)} className="prediction-chart-threshold" />
        <text x={width - inset.right} y={y(safeThreshold) - 7} textAnchor="end" className="prediction-chart-threshold-label">
          limiar {formatPercent(safeThreshold)}
        </text>
        {points.length > 1 ? <polyline points={polyline} className="prediction-chart-line" /> : null}
        {points.map((item, index) => (
          <g key={item.id || `${item.run_id}-${index}`}>
            <circle cx={x(index)} cy={y(percent(item))} r="5" className={item.will_fail ? 'prediction-chart-point is-risk' : 'prediction-chart-point'} />
            {(index === 0 || index === points.length - 1) ? (
              <text x={x(index)} y={height - 12} textAnchor={index === 0 ? 'start' : 'end'} className="prediction-chart-date">
                {formatDate(item.created_at)}
              </text>
            ) : null}
          </g>
        ))}
      </svg>
    </div>
  )
}

function RiskCalendar({ history }) {
  const latestPrediction = history[0]
  const latestDate = parseDate(latestPrediction?.created_at) || new Date()
  const year = latestDate.getFullYear()
  const month = latestDate.getMonth()
  const firstOfMonth = new Date(year, month, 1)
  const calendarStart = new Date(year, month, 1 - firstOfMonth.getDay())
  const cells = Array.from({ length: 42 }, (_, index) => {
    const day = new Date(calendarStart)
    day.setDate(calendarStart.getDate() + index)
    return day
  })
  const calendarEnd = cells.at(-1)
  const monthFormatter = new Intl.DateTimeFormat('pt-BR', { month: 'long', year: 'numeric' })
  const monthLabel = calendarEnd.getMonth() === month
    ? monthFormatter.format(latestDate)
    : `${new Intl.DateTimeFormat('pt-BR', { month: 'long' }).format(latestDate)} — ${monthFormatter.format(calendarEnd)}`
  const latestWindowStart = parseDate(latestPrediction?.created_at)
  const latestWindowEnd = parseDate(latestPrediction?.window_ends_at) || latestWindowStart

  function dayState(day) {
    if (!day) return { strength: 0, predictions: 0 }
    const timestamp = day.setHours(12, 0, 0, 0)
    let predictions = 0
    history.forEach((item) => {
      const start = parseDate(item.created_at)
      if (!start) return
      const startDay = new Date(start).setHours(0, 0, 0, 0)
      if (startDay === new Date(day).setHours(0, 0, 0, 0)) predictions += 1
    })

    let strength = 0
    if (latestPrediction?.will_fail && latestWindowStart) {
      const startDay = new Date(latestWindowStart).setHours(0, 0, 0, 0)
      const endDay = new Date(latestWindowEnd).setHours(23, 59, 59, 999)
      if (timestamp >= startDay && timestamp <= endDay) strength = percent(latestPrediction)
    }
    return { strength, predictions }
  }

  return (
    <div className="prediction-calendar">
      <div className="prediction-calendar__month">{monthLabel}</div>
      <div className="prediction-calendar__weekdays" aria-hidden="true">
        {['D', 'S', 'T', 'Q', 'Q', 'S', 'S'].map((day, index) => <span key={`${day}-${index}`}>{day}</span>)}
      </div>
      <div className="prediction-calendar__grid">
        {cells.map((day) => {
          const state = dayState(day)
          const level = state.strength >= 0.67 ? 'is-high' : state.strength > 0 ? 'is-risk' : ''
          return (
            <span
              key={day.toISOString()}
              className={`${level}${state.predictions ? ' has-prediction' : ''}${day.getMonth() !== month ? ' is-outside' : ''}`}
              title={state.strength ? `Dentro de uma janela de risco (${formatPercent(state.strength)})` : undefined}
            >
              {day.getDate()}
            </span>
          )
        })}
      </div>
      <div className="prediction-calendar__legend">
        <span><i className="is-window" /> Janela de risco</span>
        <span><i className="is-run" /> Dia analisado</span>
      </div>
    </div>
  )
}

export default function PredictionHistory({ history = [], loading, error }) {
  if (loading) {
    return <div className="prediction-history-state"><span />Carregando histórico preditivo...</div>
  }
  if (error) return <p className="prediction-history-empty is-error">{error}</p>
  if (!history.length) {
    return <p className="prediction-history-empty">Esta porta ainda não possui análises preditivas registradas.</p>
  }

  const latest = history[0]
  const previous = history[1]
  const latestRisk = percent(latest)
  const delta = previous ? latestRisk - percent(previous) : null

  return (
    <div className="prediction-history">
      <div className="prediction-history__summary">
        <div><span>Risco atual</span><strong>{latest.probability == null ? (latest.will_fail ? 'Em risco' : 'Estável') : formatPercent(latestRisk)}</strong></div>
        <div><span>Variação</span><strong className={delta > 0 ? 'is-up' : ''}>{delta == null ? '—' : `${delta > 0 ? '+' : ''}${Math.round(delta * 100)} p.p.`}</strong></div>
        <div><span>Análises registradas</span><strong>{history.length}</strong></div>
        <div><span>Último modelo</span><strong>{latest.model_version || 'Não registrado'}</strong></div>
      </div>

      <div className="prediction-history__visuals">
        <section>
          <header><div><h5>Evolução do risco</h5><p>Cada ponto representa uma execução do modelo.</p></div></header>
          <RiskChart history={history} />
        </section>
        <section>
          <header><div><h5>Janela prevista</h5><p>Os dias destacados representam somente a predição oficial mais recente.</p></div></header>
          <RiskCalendar history={history} />
        </section>
      </div>

      <details className="prediction-history__runs">
        <summary>Consultar todas as rodadas <span>{history.length}</span></summary>
        <div className="prediction-history__table-wrap">
          <table>
            <thead><tr><th>Data</th><th>Probabilidade</th><th>Diagnóstico</th><th>Horizonte</th><th>Modelo</th><th>Rodada</th></tr></thead>
            <tbody>
              {history.map((item) => (
                <tr key={item.id}>
                  <td>{formatDate(item.created_at, true)}</td>
                  <td>{item.probability == null ? 'Não registrada' : formatPercent(percent(item))}</td>
                  <td><span className={item.will_fail ? 'prediction-run-status is-risk' : 'prediction-run-status'}>{item.will_fail ? 'Falha prevista' : 'Estável'}</span></td>
                  <td>{item.horizon_days} dias</td>
                  <td>{item.model_version || '—'}</td>
                  <td><code title={item.run_id}>{String(item.run_id).slice(0, 8)}</code></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </details>
    </div>
  )
}
