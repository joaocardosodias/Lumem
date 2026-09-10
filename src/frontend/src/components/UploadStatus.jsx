import './UploadStatus.css'

function statusLabel(status) {
  if (status === 1) return 'Concluído'
  if (status === 2) return 'Erro'
  return 'Processando'
}

function UploadStatus({ upload }) {
  if (!upload) return null

  return (
    <div className="upload-status">
      <p>Status do processamento</p>
      <dl>
        <div>
          <dt>Id</dt>
          <dd>{upload.id}</dd>
        </div>
        <div>
          <dt>Arquivo</dt>
          <dd>{upload.file_name}</dd>
        </div>
        <div>
          <dt>Tipo</dt>
          <dd>{upload.file_type}</dd>
        </div>
        <div>
          <dt>Status</dt>
          <dd>
            {upload.status} · {statusLabel(upload.status)}
          </dd>
        </div>
        <div>
          <dt>Linhas lidas</dt>
          <dd>{upload.rows_processed}</dd>
        </div>
        <div>
          <dt>Linhas gravadas</dt>
          <dd>{upload.rows_inserted}</dd>
        </div>
        <div>
          <dt>Criado em</dt>
          <dd>{upload.created_at}</dd>
        </div>
      </dl>
    </div>
  )
}

export default UploadStatus
