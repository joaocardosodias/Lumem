export function exportToCsv(filename, columns, rows) {
  const headerLine = columns.map((col) => `"${String(col.label).replace(/"/g, '""')}"`).join(',')
  const dataLines = rows.map((row) =>
    columns
      .map((col) => {
        const val = typeof col.value === 'function' ? col.value(row) : row[col.key]
        if (val == null) return '""'
        return `"${String(val).replace(/"/g, '""')}"`
      })
      .join(','),
  )
  const csvContent = '\uFEFF' + [headerLine, ...dataLines].join('\r\n')
  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.setAttribute('href', url)
  link.setAttribute('download', filename.endsWith('.csv') ? filename : `${filename}.csv`)
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}
