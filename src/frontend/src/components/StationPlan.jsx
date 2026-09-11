import './StationPlan.css'

const DOORS_PER_CAR = 4

function DoorSymbol() {
  return (
    <svg viewBox="0 0 28 28" fill="none" stroke="currentColor" strokeWidth="1.4" aria-hidden="true">
      <path d="M3 25V4a1 1 0 0 1 1-1h20a1 1 0 0 1 1 1v21" />
      <path className="station-plan__leaf-left" d="M6 6h7v18H6z" />
      <path className="station-plan__leaf-right" d="M15 6h7v18h-7z" />
    </svg>
  )
}

function PlanSymbol({ type }) {
  const shapes = {
    stairs: <><rect x="6" y="2" width="16" height="24" rx="2" /><path d="M9 6h10M9 10h10M9 14h10M9 18h10M9 22h10M8 3v22m12-22v22" /></>,
    lift: <><rect x="3" y="3" width="22" height="22" rx="3" /><path d="M9 7v5m-2-2 2 2 2-2m8 2V7m-2 2 2-2 2 2M7 15h14v10m-7-10v10" /></>,
    bench: <><rect x="2" y="7" width="24" height="13" rx="2" /><path d="M3 11h22M3 16h22M7 5v18M21 5v18" /></>,
    planter: <><rect x="2" y="7" width="24" height="14" rx="2" /><path d="M10 14c0-5-5-5-5-1 0 4 5 1 5 1Zm0 0c0 5 5 5 5 1 0-4-5-1-5-1Zm8 0c0-5 5-5 5-1 0 4-5 1-5 1ZM10 14l8 1" /></>,
    bin: <><rect x="7" y="5" width="14" height="20" rx="3" /><path d="M7 9h14M11 5V3h6v2M11 13v7m6-7v7" /></>,
  }
  return <svg viewBox="0 0 28 28" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">{shapes[type]}</svg>
}

function PlatformAccess() {
  return (
    <div className="station-plan__access" aria-hidden="true">
      <span className="station-plan__stairs"><PlanSymbol type="stairs" /></span>
      <span className="station-plan__lift"><PlanSymbol type="lift" /></span>
    </div>
  )
}

function Platform({ platform, doors, carCount, reverse, matchesDoor, selectedDoor, onSelectDoor }) {
  const byNumber = new Map(doors.map((door) => [Number(door.door_number), door]))
  const riskCount = doors.filter((door) => door.will_fail).length
  const furniture = ['planter', 'bench', 'bin', 'bench', 'planter', 'bench']

  return (
    <section className={`station-plan__platform${reverse ? ' station-plan__platform--reverse' : ''}`} aria-label={`Plataforma ${platform}`}>
      <header className="station-plan__platform-head">
        <div className="station-plan__platform-name">
          <b aria-hidden="true">{platform}</b>
          <div><h3>Plataforma {platform}</h3><span>Portas de plataforma</span></div>
        </div>
        <span className="station-plan__platform-count">{doors.length} portas{riskCount > 0 && <span> · {riskCount} em risco</span>}</span>
      </header>
      <div className="station-plan__deck">
        <PlatformAccess />
        <div className="station-plan__drawing">
          {Array.from({ length: carCount }, (_, carIndex) => (
            <div className="station-plan__car" key={carIndex}>
              <span className={`station-plan__furniture station-plan__furniture--${furniture[carIndex % furniture.length]}`} aria-hidden="true">
                <PlanSymbol type={furniture[carIndex % furniture.length]} />
              </span>
              <div className="station-plan__doors">
                {Array.from({ length: DOORS_PER_CAR }, (_, position) => {
                  const number = carIndex * DOORS_PER_CAR + position + 1
                  const label = String(number).padStart(2, '0')
                  const door = byNumber.get(number)
                  const matched = door && matchesDoor(door)
                  const selected = Boolean(door && selectedDoor?.platform === platform && Number(selectedDoor?.door_number) === number)
                  const status = !door ? 'sem registro' : door.will_fail ? 'em risco' : 'estável'
                  return (
                    <button
                      key={number}
                      type="button"
                      className={`station-plan__door${door?.will_fail ? ' is-risk' : ''}${selected ? ' is-selected' : ''}${door && !matched ? ' is-dimmed' : ''}`}
                      disabled={!matched}
                      aria-label={`Porta ${label}, plataforma ${platform}, ${status}`}
                      aria-pressed={selected}
                      aria-haspopup={door ? 'dialog' : undefined}
                      title={`Porta ${label} · ${platform} · ${status}${door?.sap_tag ? `\n${door.sap_tag}` : ''}`}
                      onClick={() => onSelectDoor(door)}
                    >
                      <DoorSymbol />
                      <span>{label}</span>
                      {door?.will_fail && <i aria-hidden="true">!</i>}
                    </button>
                  )
                })}
              </div>
            </div>
          ))}
        </div>
        <PlatformAccess />
      </div>
      {!doors.length && <p className="station-plan__platform-empty">Nenhuma porta registrada nesta plataforma.</p>}
    </section>
  )
}

function TrackBed({ carCount }) {
  return (
    <div className="station-plan__tracks" aria-hidden="true">
      <div className="station-plan__track" />
      <div className="station-plan__car-labels">
        {Array.from({ length: carCount }, (_, index) => <span key={index}>Carro {String(index + 1).padStart(2, '0')}</span>)}
      </div>
      <div className="station-plan__track" />
    </div>
  )
}

function StationPlan({
  station,
  platforms,
  doors,
  loading,
  error,
  filter,
  onFilter,
  search,
  onSearch,
  matchesDoor,
  selectedDoor,
  onSelectDoor,
}) {
  const riskCount = doors.filter((door) => door.will_fail).length
  const matchingCount = doors.filter(matchesDoor).length
  const isFiltered = filter !== 'all' || Boolean(search.trim())
  const carCount = Math.ceil(Math.max(24, ...doors.map((door) => Number(door.door_number))) / DOORS_PER_CAR)
  const platformPairs = Array.from({ length: Math.ceil(platforms.length / 2) }, (_, index) => platforms.slice(index * 2, index * 2 + 2))

  return (
    <section className="station-plan" aria-labelledby="station-plan-title">
      <header className="station-plan__head">
        <div>
          <p className="station-plan__eyebrow">Distribuição por plataforma</p>
          <h2 id="station-plan-title">Mapa de portas</h2>
          <p className="station-plan__description">Selecione uma porta para consultar os detalhes e o histórico.</p>
        </div>
        <span className="station-plan__station">{station.code}<span>{platforms.length} plataformas</span></span>
      </header>

      <div className="station-plan__toolbar">
        <div className="station-plan__filters" role="group" aria-label="Filtrar portas por diagnóstico">
          {[
            ['all', 'Todas', doors.length],
            ['risk', 'Em risco', riskCount],
            ['ok', 'Estáveis', doors.length - riskCount],
          ].map(([value, label, count]) => (
            <button key={value} type="button" aria-pressed={filter === value} onClick={() => onFilter(value)}>
              {label}<span>{loading ? '—' : count}</span>
            </button>
          ))}
        </div>
        <label className="station-plan__search">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true">
            <circle cx="10.5" cy="10.5" r="6.5" /><path d="m16 16 4 4" />
          </svg>
          <input type="search" aria-label="Buscar porta ou TAG SAP" placeholder="Buscar porta ou TAG SAP" value={search} onChange={(event) => onSearch(event.target.value)} />
        </label>
      </div>

      <div className="station-plan__legend" aria-label="Legenda da planta">
        <span><i className="station-plan__key" />Estável</span>
        <span><i className="station-plan__key station-plan__key--risk">!</i>Em risco</span>
        <span><i className="station-plan__key station-plan__key--selected" />Selecionada</span>
        <span><i className="station-plan__key station-plan__key--missing" />Sem registro</span>
      </div>

      {loading ? (
        <div className="station-plan__state" role="status">Carregando as portas da estação…</div>
      ) : !doors.length ? (
        <div className="station-plan__state" role="status">
          <DoorSymbol />
          <strong>{error ? 'Mapa indisponível' : 'Nenhuma porta cadastrada'}</strong>
          <span>{error ? 'Atualize os dados para tentar novamente.' : 'As portas aparecerão aqui quando houver registros para esta estação.'}</span>
        </div>
      ) : (
        <>
          {isFiltered && (
            <div className="station-plan__results" role="status">
              <span>{matchingCount ? `${matchingCount} ${matchingCount === 1 ? 'porta encontrada' : 'portas encontradas'}` : 'Nenhuma porta encontrada.'}</span>
              <button type="button" onClick={() => { onFilter('all'); onSearch('') }}>Limpar filtros</button>
            </div>
          )}
          <div className="station-plan__platforms">
            <div className="station-plan__scroll" tabIndex={0} role="region" aria-label={`Planta de ${station.name}; role horizontalmente para ver todas as portas`}>
              <div className="station-plan__map" style={{ '--car-count': carCount }}>
                {platformPairs.map((pair) => (
                  <div className="station-plan__pair" key={pair[0]}>
                    {pair.map((platform, index) => (
                      <div className="station-plan__platform-group" key={platform}>
                        {index > 0 && <TrackBed carCount={carCount} />}
                        <Platform
                          platform={platform}
                          doors={doors.filter((door) => door.platform === platform)}
                          carCount={carCount}
                          reverse={index > 0}
                          matchesDoor={matchesDoor}
                          selectedDoor={selectedDoor}
                          onSelectDoor={onSelectDoor}
                        />
                      </div>
                    ))}
                    {pair.length === 1 && <TrackBed carCount={carCount} />}
                  </div>
                ))}
              </div>
            </div>
          </div>
          <div className="station-plan__equipment-legend" aria-label="Elementos ilustrativos da planta">
            <span><i className="station-plan__safety-key" />Linha de segurança</span>
            {[
              ['stairs', 'Escada'], ['lift', 'Elevador'], ['bench', 'Banco'], ['planter', 'Jardineira'], ['bin', 'Lixeira'],
            ].map(([type, label]) => <span key={type}><PlanSymbol type={type} />{label}</span>)}
          </div>
        </>
      )}

      <footer className="station-plan__foot">
        <span>Vista esquemática, sem escala. Acessos e mobiliário ilustrativos.</span>
        <span className="station-plan__scroll-hint">Deslize para ver todas as portas <span aria-hidden="true">↔</span></span>
      </footer>
    </section>
  )
}

export default StationPlan
