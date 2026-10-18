import './DataPageSkeleton.css'

function DataPageSkeleton({ metrics = 4, label = 'Carregando dados' }) {
  return (
    <div className="data-page-skeleton" role="status" aria-label={label}>
      <span className="data-page-skeleton__sr">{label}…</span>

      <div
        className="data-page-skeleton__metrics"
        style={{ '--skeleton-metrics': Math.min(Math.max(metrics, 2), 5) }}
        aria-hidden="true"
      >
        {Array.from({ length: metrics }, (_, index) => (
          <span key={index} className="data-page-skeleton__metric">
            <i />
            <strong />
            <small />
          </span>
        ))}
      </div>

      <div className="data-page-skeleton__panels" aria-hidden="true">
        <span><i /><strong /></span>
        <span><i /><strong /></span>
      </div>

      <div className="data-page-skeleton__table" aria-hidden="true">
        <span className="data-page-skeleton__table-title" />
        <span className="data-page-skeleton__table-filter" />
        {Array.from({ length: 4 }, (_, index) => (
          <span key={index} className="data-page-skeleton__row" />
        ))}
      </div>
    </div>
  )
}

export default DataPageSkeleton
