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
  { code: 'VSO', name: 'Vila Sônia', photo: imgVso, platforms: ['P1', 'P2'] },
  { code: 'MBI', name: 'São Paulo-Morumbi', photo: imgMbi, platforms: ['P1', 'P2'] },
  { code: 'BUT', name: 'Butantã', photo: imgBut, platforms: ['P1', 'P2'] },
  { code: 'PIN', name: 'Pinheiros', photo: imgPin, platforms: ['P1', 'P2', 'P3', 'P4'] },
  { code: 'FAL', name: 'Faria Lima', photo: imgFal, platforms: ['P1', 'P2'] },
  { code: 'FRA', name: 'Fradique Coutinho', photo: imgFra, platforms: ['P1', 'P2'] },
  { code: 'OSC', name: 'Oscar Freire', photo: imgOsc, platforms: ['P1', 'P2'] },
  { code: 'PAU', name: 'Paulista', photo: imgPau, platforms: ['P1', 'P2'] },
  { code: 'HIG', name: 'Higienópolis-Mackenzie', photo: imgHig, platforms: ['P1', 'P2'] },
  { code: 'REP', name: 'República', photo: imgRep, platforms: ['P1', 'P2', 'P3', 'P4'] },
  { code: 'LUZ', name: 'Luz', photo: imgLuz, platforms: ['P1', 'P2', 'P3', 'P4'] },
]

export function stationLabel(code) {
  return lineStations.find((item) => item.code === code)?.name ?? code
}
