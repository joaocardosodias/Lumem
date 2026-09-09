import { pick, stationFromLocation } from './spreadsheet'

function asNumber(value) {
  if (value == null || value === '') return null
  const parsed = Number(String(value).replace(',', '.'))
  return Number.isFinite(parsed) ? parsed : null
}

export function mapMaintenanceRows(rows, orderType) {
  return rows
    .map((row, index) => {
      const location = pick(row, ['installation_location', 'local_de_instalacao', 'local'])
      const station =
        pick(row, ['station_code', 'estacao', 'station']).toUpperCase() || stationFromLocation(location)

      return {
        id: `mnt-upload-${Date.now()}-${index}`,
        order_type: (pick(row, ['order_type', 'tipo_da_ordem', 'tipo_de_nota', 'tipo']) || orderType).toUpperCase(),
        sap_order_number: pick(row, ['sap_order_number', 'ordem', 'numero_da_ordem_sap', 'nota']),
        installation_location: location,
        station_code: station,
        start_timestamp: pick(row, [
          'start_timestamp',
          'inicio_avaria_datetime',
          'inicio',
          'data_de_inicio',
          'dt_inicio',
        ]),
        end_timestamp: pick(row, ['end_timestamp', 'fim_avaria_datetime', 'fim', 'data_de_fim', 'dt_fim']),
        duration_hours: asNumber(pick(row, ['duration_hours', 'duracao_avaria_horas', 'duracao', 'duracao_horas'])),
        description: pick(row, ['description', 'descricao']),
        symptom: pick(row, ['symptom', 'sintoma']),
        source: 'upload',
      }
    })
    .filter((row) => row.station_code || row.sap_order_number || row.start_timestamp)
}

export function mapAlarmRows(rows) {
  return rows
    .map((row) => ({
      tag: pick(row, ['tag', 'sap_tag', 'alarme_tag']),
      started_at: pick(row, ['started_at', 'start_timestamp', 'inicio', 'data_de_inicio', 'data_e_hora']),
      station: pick(row, ['station', 'estacao', 'station_code', 'station_name']).toUpperCase(),
      alarm: pick(row, ['alarm', 'tipo_de_alarme', 'tipo', 'alarme']),
      normalized_at: pick(row, [
        'normalized_at',
        'end_timestamp',
        'fim',
        'data_de_normalizacao',
        'normalizacao',
      ]),
    }))
    .filter((row) => row.tag || row.alarm || row.started_at)
}
