import { api, uploadIdFrom } from './api'

export function wait(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

export async function pollUpload(result, onTick) {
  const initial = result?.data && typeof result.data === 'object' ? result.data : null
  if (initial?.id) onTick(initial)
  if (initial && (initial.status === 'completed' || initial.status === 'failed')) return initial

  const id = uploadIdFrom(result) || initial?.id
  if (!id) {
    throw new Error('A API aceitou o arquivo, mas não devolveu o id do upload.')
  }

  const started = Date.now()
  while (Date.now() - started < 90000) {
    const status = await api.uploadStatus(id)
    onTick(status)
    if (status.status === 'completed' || status.status === 'failed') return status
    await wait(800)
  }

  throw new Error('Tempo esgotado no processamento do arquivo.')
}
