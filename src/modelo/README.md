# Modelo 5, API de predição

Funciona com o Modelo 5 (Random Forest, `models/modelo-5/modelo_5.ipynb`) atrás de
uma API HTTP, no formato que o backend em Go já espera consumir
(`internal/adapter/mlclient/client.go`).

O RNN (`models/modelo-rnn/`) continua documentado como o experimento de
comparação da Sprint 3, mas quem serve a predição de verdade é o Modelo 5 
ele venceu a comparação (PR-AUC 0,185 contra 0,154 da RNN).

## Contrato

`POST /predict`

```json
// request
[{"station_code": "BUT", "platform": "BUT1", "door_number": 8}]

// response
[{"station_code": "BUT", "platform": "BUT1", "door_number": 8, "will_fail": true, "probability": 0.62}]
```

`probability` é extra (o Go de hoje só lê `will_fail`), incluído porque o
enunciado pede pra devolver a probabilidade e a decisão.

`GET /health` — usado pelo healthcheck do container.

## Como funciona

1. Recebe a lista de portas (só identificadores — nenhum dado de telemetria
   trafega pela rede).
2. Busca no Postgres os alarmes dos últimos 60 dias (`telemetry_alarms` — o
   dobro da maior janela usada no treino, só pra "esquentar" o filtro de
   decaimento por recência) e o histórico completo de ordens
   (`maintenance_events`) das estações pedidas.
3. Calcula, ao vivo, as mesmas 83 features candidatas do treino (janelas
   móveis, decaimento por recência, histórico de corretivas) — ver
   `pipeline.py`.
4. Seleciona as features que o SHAP manteve no treino, roda o modelo e
   aplica o limiar salvo.

## Artefatos

`artifacts/modelo_5.joblib` é gerado por `exportar_modelo.py` — reproduz as
seções 1 a 6 do `modelo_5.ipynb` (mesmo split temporal, mesmos hiperparâmetros,
mesma seleção de features por SHAP) e salva o bundle:
`{"model", "features", "threshold", "est_categorias"}`.

```bash
python exportar_modelo.py
```

##  Validado contra o banco real (staging, 09/2026)

- `order_type` em `maintenance_events` é literalmente `ZM01` (2.700 linhas),
  `ZM02` (52) e `ZM03` (3.021) — bate exatamente com o assumido.
- `installation_location`: **41% das 5.773 linhas** (2.362) batem com o
  regex e trazem a porta específica (ex:
  `SP04-6-PSD-BUT-PLT1-PP21PDM`) — as features de ciclo/corretiva funcionam
  de verdade pra essas. Os outros ~51% vêm num formato totalmente diferente
  (`"PLATAFORMA 1"`, sem tag), que o regex não reconhece — não quebra nada,
  só cai no valor padrão (sem histórico) pra essas linhas.
- Rodei o container Docker de verdade (`docker build` + `docker run` com a
  `DATABASE_URL` real) e chamei `/predict` pra várias portas de
  `BUT`/`BUT1` — respondeu formato certo, sem erro, com probabilidades
  variando entre portas (0,18 a 0,26).

##  O que ainda não foi possível confirmar

- **Formato de `platform` na tabela `doors`** (de onde o Go monta a
  requisição): confirmamos que `telemetry_alarms.platform` usa `"BUT1"`,
  mas não temos acesso à tabela `doors` em si pra saber se ela usa a mesma
  convenção ou algo como `"P1"`. O código não assume nenhum dos dois —
  extrai só o número final da string (`normaliza_via`) — mas vale conferir
  com o time do backend.
- **PDM vs. MDU em `telemetry_alarms`**: não existe coluna de componente
  nesta tabela. Assumimos `door_number` preenchido = alarme de porta (PDM) e
  `door_number` nulo = alarme do módulo da plataforma (MDU) — plausível pelo
  schema, mas não confirmado com um exemplo real de linha MDU.

Tudo o resto (cálculo das features, carregamento do modelo, formato da
resposta) também foi testado de ponta a ponta com dado falso no lugar do
banco — ver `tests/test_predict.py`.

## Rodar localmente

```bash
pip install -r requirements.txt
export DATABASE_URL=postgresql://...
gunicorn --bind 0.0.0.0:8000 app:app
```

## Testes

```bash
python -m pytest tests/ -v -s
```

## docker-compose

Serviço a adicionar em `src/backend/docker-compose.prod.yml` (mesma VM do
backend, conforme decidido com o time):

```yaml
  ml-service:
    build:
      context: ../modelo
      dockerfile: Dockerfile
    image: ${ACR_REGISTRY}/g04-ml-service:latest
    container_name: ml-linha4
    restart: unless-stopped
    expose:
      - "8000"
    environment:
      DATABASE_URL: ${DATABASE_URL}
    healthcheck:
      test: ["CMD", "python", "-c", "import urllib.request; urllib.request.urlopen('http://localhost:8000/health')"]
      interval: 15s
      timeout: 3s
      retries: 3
      start_period: 5s
```

E no serviço `backend`, confirmar:

```yaml
    environment:
      VPS_MODEL_URL: http://ml-service:8000
```

(usa o nome do serviço Docker, não `localhost`, porque os dois containers
vão estar na mesma rede interna do compose, não na mesma máquina host
diretamente.)
