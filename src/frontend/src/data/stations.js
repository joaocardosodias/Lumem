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

import { lineStations as stationMetadata } from './stationCatalog.js'
export { mapStation, resolveStation, stationLabel, normalizeMaintenance } from './stationCatalog.js'

const photos = { VSO: imgVso, MBI: imgMbi, BUT: imgBut, PIN: imgPin, FAL: imgFal, FRA: imgFra, OFR: imgOsc, PAU: imgPau, HIG: imgHig, REP: imgRep, LUZ: imgLuz }
export const lineStations = stationMetadata.map((station) => ({ ...station, photo: photos[station.code] }))