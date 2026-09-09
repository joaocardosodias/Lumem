import * as XLSX from 'xlsx'

function normalizeKey(key) {
  return String(key)
    .trim()
    .toLowerCase()
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_|_$/g, '')
}

function parseCsv(text) {
  const rows = []
  let row = []
  let cell = ''
  let quoted = false

  for (let i = 0; i < text.length; i += 1) {
    const char = text[i]
    const next = text[i + 1]

    if (quoted) {
      if (char === '"' && next === '"') {
        cell += '"'
        i += 1
        continue
      }
      if (char === '"') {
        quoted = false
        continue
      }
      cell += char
      continue
    }

    if (char === '"') {
      quoted = true
      continue
    }
    if (char === ',') {
      row.push(cell)
      cell = ''
      continue
    }
    if (char === '\n') {
      row.push(cell)
      rows.push(row)
      row = []
      cell = ''
      continue
    }
    if (char !== '\r') cell += char
  }

  if (cell.length || row.length) {
    row.push(cell)
    rows.push(row)
  }

  return rows.filter((item) => item.some((value) => String(value).trim() !== ''))
}

function rowsToObjects(rows) {
  const [headerRow, ...dataRows] = rows
  if (!headerRow) return []

  const headers = headerRow.map((header) => normalizeKey(header))

  return dataRows.map((values) => {
    const entry = {}
    headers.forEach((header, index) => {
      if (!header) return
      entry[header] = values[index] == null ? '' : String(values[index]).trim()
    })
    return entry
  })
}

export async function parseSpreadsheet(file) {
  const name = file.name.toLowerCase()

  if (name.endsWith('.csv') || file.type === 'text/csv') {
    return rowsToObjects(parseCsv(await file.text()))
  }

  if (!name.endsWith('.xlsx') && !name.endsWith('.xls')) {
    throw new Error('Use um arquivo CSV ou XLSX.')
  }

  const workbook = XLSX.read(await file.arrayBuffer(), { type: 'array', cellDates: true })
  const sheet = workbook.Sheets[workbook.SheetNames[0]]
  const rows = XLSX.utils.sheet_to_json(sheet, { defval: '', raw: false })

  return rows.map((row) => {
    const entry = {}
    Object.entries(row).forEach(([key, value]) => {
      entry[normalizeKey(key)] = value == null ? '' : String(value).trim()
    })
    return entry
  })
}

export function pick(row, keys) {
  for (const key of keys) {
    const value = row[key]
    if (value != null && String(value).trim() !== '') return String(value).trim()
  }
  return ''
}

export function stationFromLocation(location) {
  const match = String(location).toUpperCase().match(/PSD[-_]?([A-Z]{3})/)
  return match ? match[1] : ''
}
