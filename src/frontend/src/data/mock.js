export const currentUser = {
  id: 'usr-001',
  name: 'Marta Oliveira',
  email: 'operador@viaquatro.com.br',
  role: 'Admin',
}

export const mockAuth = {
  email: currentUser.email,
  password: 'motiva',
}

export const dashboardSummary = {
  total_doors: 526,
  doors_at_risk: 150,
  pending_inspections: 12,
  completed_inspections: 48,
  failure_probability_7d: 1,
  alerts_last_7d: 150,
  station_most_alarms: 'BUT',
  critical_stations: [
    { station_code: 'BUT', station_name: 'Butantã', doors_at_risk: 18 },
    { station_code: 'FAL', station_name: 'Faria Lima', doors_at_risk: 11 },
    { station_code: 'PIN', station_name: 'Pinheiros', doors_at_risk: 9 },
    { station_code: 'PAU', station_name: 'Paulista', doors_at_risk: 7 },
  ],
}

export const latestPredictions = [
  {
    door_id: 109,
    station_name: 'Butantã',
    platform: 'P1',
    door_number: 9,
    sap_tag: 'PSD_BUT_P1_PDM09',
    will_fail: true,
    risk: 0.92,
    last_evaluated_at: '2026-09-08 18:40:00',
  },
  {
    door_id: 204,
    station_name: 'Faria Lima',
    platform: 'P2',
    door_number: 4,
    sap_tag: 'PSD_FAL_P2_PDM04',
    will_fail: true,
    risk: 0.81,
    last_evaluated_at: '2026-09-08 18:40:00',
  },
  {
    door_id: 312,
    station_name: 'Pinheiros',
    platform: 'P1',
    door_number: 12,
    sap_tag: 'PSD_PIN_P1_PDM12',
    will_fail: true,
    risk: 0.74,
    last_evaluated_at: '2026-09-08 18:40:00',
  },
  {
    door_id: 401,
    station_name: 'Paulista',
    platform: 'P2',
    door_number: 1,
    sap_tag: 'PSD_PAU_P2_PDM01',
    will_fail: false,
    risk: 0.41,
    last_evaluated_at: '2026-09-08 18:40:00',
  },
]

export const maintenances = [
  {
    tipo: 'Z1',
    nota: '14943931',
    ordem: '53954586',
    indisponibilidade: 'Sim',
    falha: 'Sim',
    analisada: 'Sim',
    cce: 'MI / NVT1 - AVARIADA',
    perfil: '6PSD',
  },
  {
    tipo: 'ZM01',
    nota: '14944002',
    ordem: '53954610',
    indisponibilidade: 'Não',
    falha: 'Sim',
    analisada: 'Não',
    cce: 'BUT / P1 - SENSOR',
    perfil: '6PSD',
  },
]

export const users = [
  { id: 'usr-001', name: 'João Pedro Gonçalves', role: 'Admin' },
  { id: 'usr-002', name: 'Marta Oliveira', role: 'Operadora' },
]

export const portasResumo = [
  { label: 'Total de portas', value: '24 portas' },
  { label: 'Portas em risco', value: '8 portas' },
]

export const alarms = [
  {
    tag: 'PSD_FAL_FAL2_PDM09_OBS',
    started_at: '12/05/2026 08:16:53',
    station: 'BUTANTÃ',
    alarm: 'OBS',
    normalized_at: '12/05/2026 08:16:54',
  },
  {
    tag: 'PSD_BUT_P1_PDM03_RCE',
    started_at: '08/09/2026 06:22:11',
    station: 'BUTANTÃ',
    alarm: 'RCE',
    normalized_at: '08/09/2026 06:24:40',
  },
  {
    tag: 'PSD_PIN_P2_PDM07_FO',
    started_at: '08/09/2026 17:03:08',
    station: 'PINHEIROS',
    alarm: 'FO',
    normalized_at: '08/09/2026 17:09:21',
  },
  {
    tag: 'PSD_PAU_P1_PDM02_RCE',
    started_at: '08/09/2026 11:42:09',
    station: 'PAULISTA',
    alarm: 'RCE',
    normalized_at: '08/09/2026 11:48:33',
  },
  {
    tag: 'PSD_FAL_P1_PDM11_OBS',
    started_at: '07/09/2026 19:05:14',
    station: 'FARIA LIMA',
    alarm: 'OBS',
    normalized_at: '07/09/2026 19:05:40',
  },
  {
    tag: 'PSD_BUT_P2_PDM08_FO',
    started_at: '07/09/2026 07:18:02',
    station: 'BUTANTÃ',
    alarm: 'FO',
    normalized_at: '07/09/2026 07:22:11',
  },
]
