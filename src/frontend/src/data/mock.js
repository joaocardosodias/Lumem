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
    station_code: 'BUT',
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
    station_code: 'FAL',
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
    station_code: 'PIN',
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
    station_code: 'PAU',
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
    id: 'mnt-001',
    order_type: 'ZM01',
    sap_order_number: '53954586',
    installation_location: 'SP04-6-PSD-BUT-P1-PP09',
    station_code: 'BUT',
    start_timestamp: '2026-09-03 14:00:00',
    end_timestamp: '2026-09-03 15:30:00',
    duration_hours: 1.5,
    description: 'Ajuste do sensor da porta',
    symptom: 'Laço de PDM aberto',
    source: 'upload',
  },
  {
    id: 'mnt-002',
    order_type: 'ZM02',
    sap_order_number: '53954610',
    installation_location: 'SP04-6-PSD-FAL-P1-PP04',
    station_code: 'FAL',
    start_timestamp: '2026-09-04 09:00:00',
    end_timestamp: '2026-09-04 10:15:00',
    duration_hours: 1.25,
    description: 'Troca do rolete da folha esquerda',
    symptom: 'Ruído na abertura',
    source: 'manual',
  },
  {
    id: 'mnt-003',
    order_type: 'ZM01',
    sap_order_number: '53954702',
    installation_location: 'SP04-6-PSD-PIN-P2-PP12',
    station_code: 'PIN',
    start_timestamp: '2026-09-05 22:10:00',
    end_timestamp: '2026-09-05 23:40:00',
    duration_hours: 1.5,
    description: 'Correção de falha no LSD',
    symptom: 'Porta não fecha',
    source: 'upload',
  },
  {
    id: 'mnt-004',
    order_type: 'ZM03',
    sap_order_number: '53954818',
    installation_location: 'SP04-6-PSD-PAU-P1-PP02',
    station_code: 'PAU',
    start_timestamp: '2026-09-06 01:00:00',
    end_timestamp: '2026-09-06 03:00:00',
    duration_hours: 2,
    description: 'Inspeção preventiva do rack',
    symptom: 'Rotina preventiva',
    source: 'upload',
  },
  {
    id: 'mnt-005',
    order_type: 'ZM01',
    sap_order_number: '53954944',
    installation_location: 'SP04-6-PSD-BUT-P2-PP03',
    station_code: 'BUT',
    start_timestamp: '2026-09-07 06:20:00',
    end_timestamp: '2026-09-07 07:05:00',
    duration_hours: 0.75,
    description: 'Reaperto da correia de tração',
    symptom: 'Correia de tração danificada',
    source: 'manual',
  },
  {
    id: 'mnt-006',
    order_type: 'ZM02',
    sap_order_number: '53955001',
    installation_location: 'SP04-6-PSD-FAL-P2-PP08',
    station_code: 'FAL',
    start_timestamp: '2026-09-08 02:30:00',
    end_timestamp: '2026-09-08 04:00:00',
    duration_hours: 1.5,
    description: 'Substituição do sensor LSD',
    symptom: 'Sensor LSD instável',
    source: 'upload',
  },
]

export const users = [
  { id: 'usr-001', name: 'João Pedro Gonçalves', role: 'Admin' },
  { id: 'usr-002', name: 'Marta Oliveira', role: 'Operadora' },
]

import imgVso from '../assets/estacoes/VilaSonia.jpeg'
import imgMbi from '../assets/estacoes/Morumbi.jpg'
import imgBut from '../assets/estacoes/butanta.jpg'
import imgPin from '../assets/estacoes/pinheiros.jpg'
import imgFal from '../assets/estacoes/farialima.jpg'
import imgFra from '../assets/estacoes/FradiqueCoutinho.jpg'
import imgOsc from '../assets/estacoes/OscarFreire.jpg'
import imgPau from '../assets/estacoes/Paulista.jpg'
import imgHig from '../assets/estacoes/higienopolis.jpg'
import imgRep from '../assets/estacoes/Republica.jpg'
import imgLuz from '../assets/estacoes/Luz.jpg'

export const lineStations = [
  { code: 'VSO', name: 'Vila Sônia', photo: imgVso },
  { code: 'MBI', name: 'São Paulo-Morumbi', photo: imgMbi },
  { code: 'BUT', name: 'Butantã', photo: imgBut },
  { code: 'PIN', name: 'Pinheiros', photo: imgPin },
  { code: 'FAL', name: 'Faria Lima', photo: imgFal },
  { code: 'FRA', name: 'Fradique Coutinho', photo: imgFra },
  { code: 'OSC', name: 'Oscar Freire', photo: imgOsc },
  { code: 'PAU', name: 'Paulista', photo: imgPau },
  { code: 'HIG', name: 'Higienópolis-Mackenzie', photo: imgHig },
  { code: 'REP', name: 'República', photo: imgRep },
  { code: 'LUZ', name: 'Luz', photo: imgLuz },
]

const stationRiskSeed = {
  VSO: 0.28,
  MBI: 0.33,
  BUT: 0.78,
  PIN: 0.54,
  FAL: 0.61,
  FRA: 0.41,
  OSC: 0.36,
  PAU: 0.42,
  HIG: 0.31,
  REP: 0.38,
  LUZ: 0.35,
}

export function getStationDoors(stationCode) {
  const base = stationRiskSeed[stationCode] ?? 0.3

  return Array.from({ length: 24 }, (_, index) => {
    const doorNumber = index + 1
    const platform = doorNumber <= 12 ? 'P1' : 'P2'
    const prediction = latestPredictions.find(
      (item) => item.station_code === stationCode && item.door_number === doorNumber,
    )
    const wave = ((doorNumber * 17) % 45) / 100
    const risk = prediction?.risk ?? Math.min(0.97, Math.round((base * (0.55 + wave)) * 100) / 100)
    const willFail = prediction?.will_fail ?? risk >= 0.55

    return {
      door_id: Number(`${stationCode.charCodeAt(0)}${stationCode.charCodeAt(1)}${String(doorNumber).padStart(2, '0')}`),
      station_code: stationCode,
      platform,
      door_number: doorNumber,
      sap_tag: `PSD_${stationCode}_${platform}_PDM${String(doorNumber).padStart(2, '0')}`,
      will_fail: willFail,
      risk,
      alarms: willFail ? 6 + (doorNumber % 8) : doorNumber % 4,
      last_maintenance_days: willFail ? 8 + (doorNumber % 12) : 18 + (doorNumber % 20),
      status: willFail ? 'at_risk' : 'stable',
      last_evaluated_at: prediction?.last_evaluated_at ?? '2026-09-08 18:40:00',
    }
  })
}

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
