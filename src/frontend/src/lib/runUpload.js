export function wait(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

export async function runUploadProgress(job, rowCount, onTick) {
  const total = Math.max(rowCount, 1)
  onTick({
    id: job.id,
    file_name: job.file_name,
    file_type: job.file_type,
    status: 0,
    rows_processed: 0,
    rows_inserted: 0,
    created_at: job.created_at,
  })

  for (let index = 1; index <= total; index += 1) {
    await wait(32)
    onTick({
      id: job.id,
      file_name: job.file_name,
      file_type: job.file_type,
      status: 0,
      rows_processed: index,
      rows_inserted: 0,
      created_at: job.created_at,
    })
  }

  const done = {
    id: job.id,
    file_name: job.file_name,
    file_type: job.file_type,
    status: 1,
    rows_processed: total,
    rows_inserted: total,
    created_at: job.created_at,
  }
  onTick(done)
  await wait(400)
  return done
}
