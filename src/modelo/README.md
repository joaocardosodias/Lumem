# Serviço de Machine Learning

Este serviço executa o Modelo 5, um `RandomForestClassifier` treinado para estimar a probabilidade de uma porta apresentar falha nos próximos sete dias. Ele ficou responsável somente pela parte matemática do fluxo: montar as mesmas features do treinamento e devolver probabilidades. O backend Go escolhe o threshold, classifica as portas e executa as regras operacionais, como registrar a rodada e abrir inspeções.

## Arquitetura

O código foi separado em domínio, aplicação, portas e adaptadores:

```text
ml_service/
  domain/       validação do artefato, ordem das features e inferência
  application/  casos de uso de predição, treinamento e monitoramento
  ports/        contratos de repositório de features e registry de modelo
  adapters/     PostgreSQL e registry local/S3
app.py          adaptador HTTP Flask
training_worker.py consumidor dedicado da fila de treinamentos
```

O `ml-service` atende inferências e registra novos jobs no volume compartilhado. O container `ml-training-worker` reivindica um job por vez e executa o pipeline pesado em processo separado, com limites próprios de CPU e memória. Dessa forma, SHAP e Random Forest não disputam diretamente os recursos reservados às respostas de inferência.

O domínio e os casos de uso não conhecem Flask, SQLAlchemy, nomes de tabelas ou AWS SDK. O adaptador PostgreSQL consulta apenas as views versionadas `ml_telemetry_v1`, `ml_maintenance_v1` e `ml_active_doors_v1`. Em produção, `ML_DATABASE_URL` é obrigatória e deve pertencer ao usuário somente leitura criado a partir de `src/backend/migrations/create_ml_readonly_role.example.sql`.

## Contrato interno v1

O backend usa exclusivamente as rotas versionadas descritas em `openapi.yaml`:

- `POST /v1/predictions`: recebe `{ "doors": [...] }` e devolve `model_version`, `reference_date` e probabilidades. O payload não recebe threshold e a resposta não contém `will_fail`.
- `GET /v1/models/active`: devolve o manifest da versão carregada.
- `GET /v1/models/{version}/metrics`: devolve a curva de thresholds produzida no teste temporal.
- `POST /v1/models/{version}/activate`: valida o candidato e altera o ponteiro ativo.
- `POST /v1/training-jobs` e `GET /v1/training-jobs/{id}`: iniciam e acompanham treinamento real assíncrono. A resposta inclui `progress.stages`, com checkpoints, horários, durações e a etapa exata de uma eventual falha.
- `POST /v1/training-jobs/{id}/resume`: cria uma nova tentativa para um job com falha ou cancelado, reutiliza as saídas persistidas das etapas anteriores e recomeça exatamente na etapa interrompida.
- `GET /v1/models/{version}/drift`: compara as features atuais com a distribuição de referência do treinamento.
- `GET /health/live` e `GET /health/ready`: distinguem processo vivo de modelo realmente carregado.

As rotas `/predict`, `/metadata`, `/drift`, `/retrain` e `/health` permanecem temporariamente para clientes antigos e devolvem o cabeçalho `Deprecation: true`.

## Métricas reproduzíveis

`exportar_modelo.py` usa o split temporal original e gera, junto do artefato:

- `manifest.json`, com hashes SHA-256 do modelo e do conjunto de teste, versão, target, períodos, features, hiperparâmetros e distribuição de referência;
- `threshold-metrics.json`, com 91 pontos entre 0,05 e 0,95;
- TP, FP, FN, TN, precisão, recall, F1, quantidade e taxa de alertas em cada ponto.

O threshold é escolhido sobre validação. As métricas apresentadas como desempenho final são calculadas sobre teste temporal intocado. `validate_manifest.py` recalcula as relações da matriz de confusão e impede a publicação de um manifest inconsistente.

Para conferir o artefato atual sem retreiná-lo:

```bash
python exportar_modelo.py --version v1.0.0 --evaluate-existing --activate-local
python validate_manifest.py artifacts/manifest.json artifacts/modelo_5.joblib
```

Para criar uma candidata sem alterar a versão ativa:

```bash
python exportar_modelo.py --version v1.1.0
```

Em `MODEL_REGISTRY_DRIVER=s3`, a candidata é publicada no bucket privado `artefacts`, sob o prefixo imutável `models/random-forest/<versão>/`. A promoção valida o manifest e o SHA-256 antes de atualizar `models/random-forest/active/manifest.json`.

## Dados de treinamento

No primeiro boot, `model-registry-init` envia `models/modelo-5/painel.parquet` e `data/Corretiva Emergencial/processados/ZM01_limpa.xlsx` ao bucket `training-data`. O treinamento baixa as chaves configuradas por `MODEL_PANEL_S3_KEY` e `MODEL_MAINTENANCE_S3_KEY` para `MODEL_TRAINING_CACHE_DIR`. O mesmo inicializador publica o modelo `v1.0.0`, seu manifest e a curva de thresholds em `artefacts` caso ainda não exista um ponteiro ativo.

Os jobs e seus checkpoints intermediários ficam no volume persistente montado em `/var/lib/g04/training-jobs`, enquanto modelos e datasets persistem no MinIO. Se o processo reiniciar no meio de um job, o registro é recuperado como `failed`, preservando a causa e evitando um estado `running` eterno. A retomada reaproveita snapshots, painéis processados, seleção de features, modelos e avaliação já concluídos; somente a etapa com falha e as posteriores são executadas novamente. Retomadas sucessivas mantêm a mesma raiz de checkpoints. Um job na fila ou em execução também pode ser cancelado por `POST /v1/training-jobs/{id}/cancel`; o grupo de processos é encerrado e os checkpoints concluídos permanecem no volume. Apenas um treinamento pode ser executado por instância de serviço.

## Execução local

O serviço possui seu próprio MinIO para modelos, datasets e checkpoints. Crie a rede compartilhada uma vez, prepare o arquivo de ambiente e suba a stack:

```bash
docker network inspect backend-network >/dev/null 2>&1 || docker network create backend-network
cd src/modelo
cp .env.example .env
docker compose up -d --build
```

Em desenvolvimento, mantenha `ML_BIND_ADDRESS=127.0.0.1`; o backend alcança o serviço pelo alias `http://ml-service:8000` na rede Docker. Na VPS 2, configure `ENV=production` e `ML_BIND_ADDRESS=10.50.0.2`; a VPS 1 passa a usar `VPS_MODEL_URL=http://10.50.0.2:8000` pelo WireGuard.

```bash
pip install -r requirements.txt
export ML_DATABASE_URL='postgresql://postgres.PROJECT_REF:senha@pooler.supabase.com:6543/postgres?sslmode=require&default_query_exec_mode=simple_protocol'
gunicorn --bind 0.0.0.0:8000 --workers 1 --threads 4 app:app
```

```bash
python validate_manifest.py artifacts/manifest.json artifacts/modelo_5.joblib
python -m pytest tests/ -v
```

O pipeline do GitLab valida manifest, testes, build e vulnerabilidades. O deploy é manual: cada VPS executa `git pull`, `docker compose build` e `docker compose up -d`; nenhuma imagem da aplicação é publicada em registry.
