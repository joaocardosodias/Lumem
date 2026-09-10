set -eu

MODE="${1:-status}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

if [ -f "$BACKEND_DIR/.env" ]; then
  ENV_FILE="$BACKEND_DIR/.env"
elif [ -f ".env" ]; then
  ENV_FILE=".env"
else
  echo "Arquivo .env nao encontrado."
  exit 1
fi

COMPOSE_FILE="$BACKEND_DIR/docker-compose.yml"
if [ -f "$BACKEND_DIR/docker-compose.prod.yml" ] && docker ps 2>/dev/null | grep -q "watchtower"; then
  COMPOSE_FILE="$BACKEND_DIR/docker-compose.prod.yml"
fi

if [ "$MODE" = "on" ]; then
  sed -i 's/^LOAD_TEST=.*/LOAD_TEST=true/' "$ENV_FILE"
  sed -i 's/^LOAD_TEST_CPUS=.*/LOAD_TEST_CPUS=0.50/' "$ENV_FILE"
  sed -i 's/^LOAD_TEST_MEMORY=.*/LOAD_TEST_MEMORY=256m/' "$ENV_FILE"
  echo "Modo LOAD_TEST ATIVADO no .env (0.50 CPU, 256MB RAM, Rate Limiter desativado)."
  echo "Reiniciando container do backend com os limites..."
  docker compose -f "$COMPOSE_FILE" up -d
  docker stats backend-linha4 --no-stream
elif [ "$MODE" = "off" ]; then
  sed -i 's/^LOAD_TEST=.*/LOAD_TEST=false/' "$ENV_FILE"
  sed -i 's/^LOAD_TEST_CPUS=.*/LOAD_TEST_CPUS=0/' "$ENV_FILE"
  sed -i 's/^LOAD_TEST_MEMORY=.*/LOAD_TEST_MEMORY=0/' "$ENV_FILE"
  echo "Modo LOAD_TEST DESATIVADO no .env (Recursos normais/ilimitados, Rate Limiter ativo)."
  echo "Reiniciando container do backend no modo normal..."
  docker compose -f "$COMPOSE_FILE" up -d
  docker stats backend-linha4 --no-stream
else
  echo "Uso: bash src/backend/scripts/toggle_load_test.sh [on|off]"
  echo "Estado atual no .env ($ENV_FILE):"
  grep -E "^LOAD_TEST" "$ENV_FILE"
fi
