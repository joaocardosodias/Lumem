set -euo pipefail

TARGET_URL="${1:-http://localhost:8080}"
echo "Iniciando bateria massiva de testes de carga contra: $TARGET_URL"
echo "Estado dos recursos do container antes do teste:"
docker stats backend-linha4 --no-stream

which hey >/dev/null 2>&1 || {
  echo "Instalando ferramenta hey..."
  go install github.com/rakyll/hey@latest
}

echo "Passo 1: Baseline de alta concorrencia no Healthcheck (2000 requisicoes, 50 concorrentes)"
hey -n 2000 -c 50 "$TARGET_URL/healthz/live"

echo "Passo 2: Autenticando para obter token JWT"
TOKEN=$(curl -s -X POST "$TARGET_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"operador@motiva.com.br","password":"operador123"}' | jq -r '.token // empty')

if [ -z "$TOKEN" ]; then
  echo "Falha ao obter token JWT de autenticacao"
  exit 1
fi

echo "Autenticado com sucesso. Token obtido."

echo "Passo 3: Teste de carga massivo no sumario do dashboard (3000 requisicoes, 50 concorrentes)"
sleep 1
hey -n 3000 -c 50 -H "Authorization: Bearer $TOKEN" "$TARGET_URL/api/v1/dashboard/summary"

echo "Passo 4: Teste de carga no banco de dados para portas da estacao PIN (1500 requisicoes, 25 concorrentes)"
sleep 1
hey -n 1500 -c 25 -H "Authorization: Bearer $TOKEN" "$TARGET_URL/api/v1/stations/PIN/doors/status"

echo "Passo 5: Estresse severo e saturacao do Rate Limiter (5000 requisicoes, 100 concorrentes)"
hey -n 5000 -c 100 -H "Authorization: Bearer $TOKEN" "$TARGET_URL/api/v1/dashboard/summary"

echo "Estado dos recursos do container apos o teste:"
docker stats backend-linha4 --no-stream

echo "Bateria massiva finalizada com sucesso."
