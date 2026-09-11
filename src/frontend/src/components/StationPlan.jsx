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
            {platforms.map((platform) => {
              const platformDoors = doors.filter((door) => door.platform === platform)
              const byNumber = new Map(platformDoors.map((door) => [Number(door.door_number), door]))
              const carCount = Math.ceil(Math.max(24, ...byNumber.keys()) / DOORS_PER_CAR)
              const platformRisk = platformDoors.filter((door) => door.will_fail).length
              return (
                <section className="station-plan__platform" key={platform} aria-label={`Plataforma ${platform}`}>
                  <header className="station-plan__platform-head">
                    <span className="station-plan__platform-name"><b>{platform}</b>Plataforma {platform}</span>
                    <span className="station-plan__platform-count">{platformDoors.length} portas{platformRisk > 0 && <span> · {platformRisk} em risco</span>}</span>
                  </header>
                  {!platformDoors.length ? (
                    <p className="station-plan__platform-empty">Nenhuma porta registrada nesta plataforma.</p>
                  ) : (
                    <div className="station-plan__scroll" tabIndex={0} role="region" aria-label={`Portas da plataforma ${platform}; role horizontalmente para ver todas`}>
                      <div className="station-plan__drawing" style={{ '--car-count': carCount }}>
                        {Array.from({ length: carCount }, (_, carIndex) => (
                          <div className="station-plan__car" key={carIndex}>
                            <span className="station-plan__car-label">Carro {String(carIndex + 1).padStart(2, '0')}</span>
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
                    </div>
                  )}
                </section>
              )
            })}
          </div>
        </>
      )}

      <footer className="station-plan__foot">
        <span>Vista esquemática · sem escala</span>
        <span className="station-plan__scroll-hint">Deslize para ver todas as portas <span aria-hidden="true">↔</span></span>
      </footer>
    </section>
  )
}

export default StationPlan
