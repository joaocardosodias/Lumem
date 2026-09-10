# Modelo 5, API de predição

Funciona com o Modelo 5 (Random Forest, `models/modelo-5/modelo_5.ipynb`) atrás de
uma API HTTP, no formato que o backend em Go já espera consumir
(`internal/adapter/mlclient/client.go`).

O RNN (`models/modelo-rnn/`) continua documentado como o experimento de
comparação da Sprint 3, mas quem serve a predição de verdade é o Modelo 5 
ele venceu a comparação (PR-AUC 0,184 contra 0,154 da RNN — números do bundle
real exportado por `exportar_modelo.py`, ver detalhes em `modelo-RNN.mdx`).

## Contrato

`POST /predict`

```json
[{"station_code": "VSO", "platform": "P1", "door_number": 8}]

[{"station_code": "VSO", "platform": "P1", "door_number": 8, "will_fail": true, "probability": 0.62}]
```

`GET /health` é usado pelo healthcheck do container.

## Como funciona

1. A API recebe a lista de portas (só identificadores).
2. Busca no Postgres os alarmes dos últimos 60 dias (`telemetry_alarms`, o
   dobro da maior janela usada no treino, para "esquentar" o filtro de
   decaimento por recência) e o histórico completo de ordens
   (`maintenance_events`) das estações pedidas.
3. Calcula, ao vivo, as mesmas 83 features candidatas do treino (janelas
   móveis, decaimento por recência, histórico de corretivas), ver
   `pipeline.py`.
4. Seleciona as features que o SHAP manteve no treino, roda o modelo e
   aplica o limiar salvo.

## Artefatos

artifacts/modelo_5.joblib é gerado por exportar_modelo.py, ele reproduz as
seções 1 a 6 do `modelo_5.ipynb` (mesmo split temporal, mesmos hiperparâmetros,
mesma seleção de features por SHAP) e salva o bundle:
`{"model", "features", "threshold", "est_categorias"}`.

```bash
python exportar_modelo.py
```

##  Validado contra o banco real (staging, 09/2026)

- order_type em maintenance_events é literalmente `ZM01` (2.700 linhas),
  `ZM02` (52) e `ZM03` (3.021).
- `installation_location`: 41% das 5.773 linhas (2.362) batem com o
  regex e trazem a porta específica (ex:
  `SP04-6-PSD-BUT-PLT1-PP21PDM`),  as features de ciclo/corretiva funcionam
  de verdade pra essas. Os outros ~51% vêm num formato diferente
  (`"PLATAFORMA 1"`, sem tag), que o regex não reconhece, não quebra nada,
  só cai no valor padrão (sem histórico) pra essas linhas.
- Foi rodado o container Docker (`docker build` + `docker run` com a
  DATABASE_URL real) e foi chamado /predict pra várias portas de
  BUT/BUT1. Foi respondido sem erro, com probabilidades
  variando entre portas (0,18 a 0,26).

## Validado de ponta a ponta com o Go (staging, 09/2026)

Foi lançado os dois serviços (backend Go buildado local + o ml-service,
ligados pela rede backend-network) contra o banco de staging real e disparamos
`POST /api/v1/predictions/trigger`, autenticado, pra 528 portas:

- `doors.platform` usa `"P1"`/`"P2"`, diferente de `telemetry_alarms.platform`
  (`"BUT1"`). São convenções diferentes, e `normaliza_via()` já lida com isso (extrai só o número
  final, funciona pros dois formatos).
- O /predict respondeu em 3,3s pras 528 portas, dentro do
  timeout de 5s que o cliente Go usa.

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

## CI/CD

O .gitlab-ci.yml (raiz do repo) cobre este serviço: `test_modelo` roda
`pytest`, `build_container_modelo` builda a imagem, `trivy_scan_modelo`
escaneia vulnerabilidade, e `publish_container_modelo` publica
`g04-ml-service` no Azure Container Registry a cada push na branch
principal.

## docker-compose

Serviço separado do backend, próprio (docker-compose.yml pro build local,
docker-compose.prod.yml pra imagem publicada), ligado na mesma rede
backend-network que o backend usa.

## Deploy

Já rodando na VM da Azure. O
Watchtower assume as atualizações seguintes sozinho a partir daqui.
