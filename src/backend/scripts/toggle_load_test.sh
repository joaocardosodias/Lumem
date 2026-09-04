set -euo pipefail

MODE="${1:-status}"
ENV_FILE="src/backend/.env"

if [ ! -f "$ENV_FILE" ]; then
  echo "Arquivo $ENV_FILE nao encontrado."
  exit 1
fi

if [ "$MODE" = "on" ]; then
  sed -i 's/^LOAD_TEST=.*/LOAD_TEST=true/' "$ENV_FILE"
  sed -i 's/^LOAD_TEST_CPUS=.*/LOAD_TEST_CPUS=0.50/' "$ENV_FILE"
  sed -i 's/^LOAD_TEST_MEMORY=.*/LOAD_TEST_MEMORY=256m/' "$ENV_FILE"
  echo "Modo LOAD_TEST ATIVADO no .env (0.50 CPU, 256MB RAM, Rate Limiter desativado)."
  echo "Reiniciando container local do backend com os limites..."
  docker compose -f src/backend/docker-compose.yml up -d
  docker stats backend-linha4 --no-stream
elif [ "$MODE" = "off" ]; then
  sed -i 's/^LOAD_TEST=.*/LOAD_TEST=false/' "$ENV_FILE"
  sed -i 's/^LOAD_TEST_CPUS=.*/LOAD_TEST_CPUS=0/' "$ENV_FILE"
  sed -i 's/^LOAD_TEST_MEMORY=.*/LOAD_TEST_MEMORY=0/' "$ENV_FILE"
  echo "Modo LOAD_TEST DESATIVADO no .env (Recursos normais/ilimitados, Rate Limiter ativo)."
  echo "Reiniciando container local do backend no modo normal..."
  docker compose -f src/backend/docker-compose.yml up -d
  docker stats backend-linha4 --no-stream
else
  echo "Uso: bash src/backend/scripts/toggle_load_test.sh [on|off]"
  echo "Estado atual no .env:"
  grep -E "^LOAD_TEST" "$ENV_FILE"
fi
