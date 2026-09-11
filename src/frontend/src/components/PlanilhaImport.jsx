import './PlanilhaImport.css'

function PlanilhaImport({ file, onFile, accept = '.csv,.xlsx,.xls', hint, error }) {
  function handleChange(event) {
    onFile(event.target.files?.[0] || null)
    event.target.value = ''
  }

  return (
    <label className="planilha-import">
      <span>Arquivo CSV ou XLSX</span>
      <input type="file" accept={accept} onChange={handleChange} />
      <em>{file ? file.name : 'Selecionar planilha'}</em>
      {hint ? <small>{hint}</small> : null}
      {error ? <p>{error}</p> : null}
    </label>
  )
}

export default PlanilhaImport
