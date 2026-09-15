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
```

O domínio e os casos de uso não conhecem Flask, SQLAlchemy, nomes de tabelas ou AWS SDK. O adaptador PostgreSQL consulta apenas as views versionadas `ml_telemetry_v1`, `ml_maintenance_v1` e `ml_active_doors_v1`. Em produção, `ML_DATABASE_URL` é obrigatória e deve pertencer ao usuário somente leitura criado a partir de `src/backend/migrations/create_ml_readonly_role.example.sql`.

## Contrato interno v1

O backend usa exclusivamente as rotas versionadas descritas em `openapi.yaml`:

- `POST /v1/predictions`: recebe `{ "doors": [...] }` e devolve `model_version`, `reference_date` e probabilidades. O payload não recebe threshold e a resposta não contém `will_fail`.
- `GET /v1/models/active`: devolve o manifest da versão carregada.
- `GET /v1/models/{version}/metrics`: devolve a curva de thresholds produzida no teste temporal.
- `POST /v1/models/{version}/activate`: valida o candidato e altera o ponteiro ativo.
- `POST /v1/training-jobs` e `GET /v1/training-jobs/{id}`: iniciam e acompanham treinamento real assíncrono.
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

Em `MODEL_REGISTRY_DRIVER=s3`, a candidata é publicada automaticamente no prefixo imutável `models/random-forest/<versão>/`. A promoção posterior valida o manifest e o SHA-256 do binário antes de atualizar `models/random-forest/active/manifest.json`.

## Dados de treinamento

No repositório local, o pipeline lê `models/modelo-5/painel.parquet` e `data/Corretiva Emergencial/processados/ZM01_limpa.xlsx`. Esses arquivos não entram na imagem. Em produção, configure caminhos montados com `MODEL_PANEL_PATH` e `MODEL_MAINTENANCE_PATH`, ou chaves privadas do S3 com `MODEL_PANEL_S3_KEY` e `MODEL_MAINTENANCE_S3_KEY`. O download é feito para `MODEL_TRAINING_CACHE_DIR` antes do treinamento.

Os jobs ficam em `/var/lib/g04/training-jobs`, montado como volume persistente nos arquivos Compose. Se o processo reiniciar no meio de um job, o registro é recuperado como `failed`, preservando a causa e evitando um estado `running` eterno. Apenas um treinamento pode ser executado por instância de serviço.

## Execução local

```bash
pip install -r requirements.txt
export ML_DATABASE_URL=postgresql://ml_reader:senha@host:5432/postgres
gunicorn --bind 0.0.0.0:8000 --workers 1 --threads 4 app:app
```

```bash
python validate_manifest.py artifacts/manifest.json artifacts/modelo_5.joblib
python -m pytest tests/ -v
```

O container de produção expõe a porta 8000 apenas na rede Docker compartilhada com o backend. O pipeline do GitLab valida manifest e testes, constrói a imagem, executa Trivy e publica `g04-ml-service` no Azure Container Registry quando as regras da branch principal forem atendidas.
