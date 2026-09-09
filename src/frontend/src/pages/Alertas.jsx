import { useState } from 'react'
import Button from '../components/Button'
import CadastrarButton from '../components/CadastrarButton'
import Card from '../components/Card'
import PlanilhaImport from '../components/PlanilhaImport'
import UploadStatus from '../components/UploadStatus'
import { api, errorMessage } from '../lib/api'
import { pollUpload } from '../lib/runUpload'
import './Alertas.css'

function Alertas() {
  const [open, setOpen] = useState(false)
  const [file, setFile] = useState(null)
  const [importError, setImportError] = useState('')
  const [importing, setImporting] = useState(false)
  const [upload, setUpload] = useState(null)
  const [lastUpload, setLastUpload] = useState(null)

  async function handleImport(event) {
    event.preventDefault()
    if (!file) {
      setImportError('Selecione um arquivo CSV ou XLSX.')
      return
    }

    setImporting(true)
    setImportError('')

    try {
      const result = await api.uploadAlarms(file)
      const done = await pollUpload(result, setUpload)
      if (done.status === 2) {
        throw new Error('A API recusou o processamento deste arquivo.')
      }
      setLastUpload(done)
      setFile(null)
      setUpload(null)
      setOpen(false)
    } catch (error) {
      setImportError(errorMessage(error))
    } finally {
      setImporting(false)
    }
  }

  function openCadastro() {
    setImportError('')
    setFile(null)
    setUpload(null)
    setOpen(true)
  }

  return (
    <section className="alertas">
      <div className="page-head">
        <h1>Alertas</h1>
        <CadastrarButton onClick={openCadastro}>Cadastrar Alarmes +</CadastrarButton>
      </div>

      <div className="alertas__hero">
        <Card
          layout="banner"
          label="Linhas gravadas no último upload"
          value={lastUpload?.rows_inserted ?? '—'}
          hint={
            lastUpload?.file_name
              ? lastUpload.file_name
              : 'A API não lista alarmes. Envie a planilha de telemetria para ingestão.'
          }
        />
      </div>

      <div className="alertas__list">
        <p className="page-empty">
          A API não lista alarmes. Os registros entram só pelo envio da planilha de telemetria.
        </p>
      </div>

      {open ? (
        <div className="alertas-modal" onClick={() => !importing && setOpen(false)}>
          <form onClick={(event) => event.stopPropagation()} onSubmit={handleImport}>
            <h2>Enviar planilha de alarmes</h2>
            <PlanilhaImport
              file={file}
              onFile={(next) => {
                setFile(next)
                setImportError('')
                setUpload(null)
              }}
              hint="Planilha de alarmes de telemetria em CSV ou XLSX."
              error={importError}
            />
            {upload ? <UploadStatus upload={upload} /> : null}
            <Button type="submit" variant="light">
              {importing ? 'Processando upload…' : 'Enviar planilha'}
            </Button>
          </form>
        </div>
      ) : null}
    </section>
  )
}

export default Alertas
