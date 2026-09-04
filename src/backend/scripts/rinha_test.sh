set -euo pipefail

TARGET_URL="${1:-${TARGET_URL:-http://localhost}}"
echo "Iniciando bateria estilo Rinha de Backend contra: $TARGET_URL"
echo "Condicoes: Limite de 0.50 CPU, 256MB RAM, Rate Limiter DESATIVADO (0% de rejeicao por 429)"

which hey >/dev/null 2>&1 || {
  echo "Instalando ferramenta hey..."
  if which go >/dev/null 2>&1; then
    go install github.com/rakyll/hey@latest
    export PATH="$PATH:$(go env GOPATH)/bin"
  elif which docker >/dev/null 2>&1; then
    docker run --rm -v /tmp:/out golang:1.26-alpine sh -c "go install github.com/rakyll/hey@latest && cp /go/bin/hey /out/hey"
    export PATH="$PATH:/tmp"
  fi
}

if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^backend-linha4$"; then
  echo "Estado dos recursos antes do estresse:"
  docker stats backend-linha4 --no-stream
fi

echo "Fase 1: Aquecimento (Warm-up) - 1000 leituras concorrentes"
hey -n 1000 -c 30 "$TARGET_URL/healthz/live"

echo "Fase 2: Batalha de Escrita Concorrente (ACID Writes no PostgreSQL) - 500 inserts de inspecao"
hey -n 500 -c 20 -m POST \
  -H "Content-Type: application/json" \
  -d '{"station_code":"PIN","platform":"P1","door_number":1,"scheduled_for":"2026-09-15","priority":"critical","notes":"rinha_stress_test"}' \
  "$TARGET_URL/api/v1/inspections"

echo "Fase 3: Batalha de Escrita Concorrente em Manutencao - 500 inserts de ordens ZM03"
hey -n 500 -c 20 -m POST \
  -H "Content-Type: application/json" \
  -d '{"order_type":"ZM03","station_code":"PIN","installation_location":"PLATAFORMA 1","start_timestamp":"2026-09-04T12:00:00Z","duration_hours":2.0,"description":"rinha_stress_order"}' \
  "$TARGET_URL/api/v1/maintenance"

echo "Fase 4: Leituras Analiticas Concorrentes com Joins e Paginacao - 1000 consultas no banco"
hey -n 1000 -c 30 "$TARGET_URL/api/v1/stations/PIN/doors/status"
hey -n 1000 -c 30 "$TARGET_URL/api/v1/inspections?limit=50"

echo "Fase 5: Estresse Misto Simultaneo (Alta Carga de Leitura + Escrita)"
hey -n 2000 -c 50 "$TARGET_URL/api/v1/dashboard/summary"

if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^backend-linha4$"; then
  echo "Estado dos recursos do container apos a Rinha:"
  docker stats backend-linha4 --no-stream
fi

echo "Verificando se o backend continua vivo e responsivo:"
curl -f "$TARGET_URL/healthz/live"
echo ""
echo "Rinha de Backend finalizada com sucesso."
