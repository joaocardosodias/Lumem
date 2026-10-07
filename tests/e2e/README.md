# Fluxo completo no navegador

O roteiro usa Microsoft Edge controlado pelo Playwright, frontend compilado,
API Go, PostgreSQL, worker River e o artefato Python real. Não há interceptação
de respostas nem substitutos de banco, fila ou inferência.

API e worker usam arquivos reais em um volume compartilhado. O modelo carrega
o registro local e consulta as views do PostgreSQL. Este ambiente não valida
MinIO/S3 nem a rede entre VPSs. As contas, credenciais e dados são exclusivos
do ambiente descartável definido neste diretório.

## Preparação

Requisitos: Docker Compose, Node 22 e Microsoft Edge instalado. Executar na raiz:

```powershell
docker compose -f tests/e2e/compose.yml up -d --build
docker compose -f tests/e2e/compose.yml ps
docker cp tests/e2e/seed.sql g04-rf-browser-postgres-1:/tmp/rf-seed.sql
docker compose -f tests/e2e/compose.yml exec -T postgres psql -U rf_browser -d rf_browser -v ON_ERROR_STOP=1 -f /tmp/rf-seed.sql
```

Aguardar backend, frontend, PostgreSQL e modelo saudáveis; o worker deve estar
em execução. A aplicação fica em `http://127.0.0.1:54173`.

## Execução

Antes de cada execução completa, limpar somente os dados fictícios deste ambiente:

```powershell
docker cp tests/e2e/reset.sql g04-rf-browser-postgres-1:/tmp/rf-reset.sql
docker compose -f tests/e2e/compose.yml exec -T postgres psql -U rf_browser -d rf_browser -v ON_ERROR_STOP=1 -f /tmp/rf-reset.sql
cd tests/e2e
npm ci --no-audit --no-fund
npm test
```

O reset verifica o nome do banco e o usuário antes de remover registros de teste.
Não executar estes scripts em um banco compartilhado. O teste espera uma base
operacional vazia, preservando estações, portas, usuários e configuração do modelo.

Os resultados e capturas ficam em `tests/e2e/output`, ignorado pelo Git. O JSON
registra respostas por caminho/status, sem salvar tokens ou senhas. O teste de
expiração substitui a sessão por um JWT corretamente assinado com a chave local
de teste e expiração no passado; a rejeição é feita pelo backend real.

O roteiro cobre login inválido/válido, recarga, expiração, upload inválido,
CSV/XLSX para alarmes e manutenção, cadastro manual, filtros, predição, inspeções,
dashboard e logout. Quando não há portas em risco, cria uma inspeção manual pela
interface. Esse caminho não comprova geração automática de inspeções com ML real.

## Conferência do modelo e da persistência

Depois do navegador terminar, executar na raiz:

```powershell
docker cp tests/e2e/verify_model.py g04-rf-browser-model-1:/tmp/verify_model.py
docker compose -f tests/e2e/compose.yml exec -T model python /tmp/verify_model.py
```

Essa conferência lê os históricos do banco, verifica os alarmes na janela de
atributos, recalcula probabilidades com o mesmo artefato e compara com as
previsões persistidas. Também informa contagens de registros e jobs River.
Ela complementa o roteiro de navegador e não é uma avaliação de qualidade
preditiva ou execução de toda a suíte Python.

## Encerramento

Na raiz, para encerrar apenas este ambiente, preservando os contêineres para consulta:

```powershell
docker compose -f tests/e2e/compose.yml stop
```

Para retomar: `docker compose -f tests/e2e/compose.yml start`.
