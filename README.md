# Lumem — Manutenção preditiva de portas de plataforma

<p align="center">
  <a href="https://www.inteli.edu.br/"><img src="https://res.cloudinary.com/dwewomj84/image/upload/v1761355911/inteli_lfuayp.png" alt="Inteli — Instituto de Tecnologia e Liderança" width="320"></a>
</p>

Projeto acadêmico do **Grupo 4 — Subway Surfers**, desenvolvido no Inteli em parceria com a **Motiva**, para apoiar a manutenção das portas de plataforma (*Platform Screen Doors*, PSDs) da **Linha 4–Amarela**.

O Lumem reúne alarmes e ordens de manutenção, estima risco de falha e apresenta os resultados por estação, plataforma e porta. O objetivo é ajudar a equipe a priorizar inspeções e intervenções, aproveitando os dados históricos da operação. Uma classificação de risco indica uma necessidade de avaliação; não determina o dia exato de uma quebra nem substitui o diagnóstico em campo.

[Documentação completa](https://graduacao.pages.git.inteli.edu.br/2026-2a/t16/g04/) · [Serviço de ML](https://git.inteli.edu.br/graduacao/2026-2a/t16/g04/-/blob/main/src/modelo/README.md) · [Frontend](https://git.inteli.edu.br/graduacao/2026-2a/t16/g04/-/blob/main/src/frontend/README.md) · [Ambientes Python](https://git.inteli.edu.br/graduacao/2026-2a/t16/g04/-/blob/main/docs/dependencias-python.md)

## O que a plataforma oferece

| Área | Funcionalidades |
| --- | --- |
| Visão geral | Indicadores operacionais e diagnóstico da rodada oficial. |
| Predições | Portas em risco, filtros e consulta às rodadas publicadas. |
| Portas | Navegação por estação/plataforma, planta da estação, detalhes do ativo, histórico preditivo e calendário da janela de risco. |
| Alarmes e manutenções | Consulta dos registros e ingestão assíncrona de planilhas CSV/XLSX. |
| Inspeções | Abertura e acompanhamento de ordens; reutilização da inspeção aberta quando uma porta volta a ser sinalizada. |
| Engenharia de dados | Laboratório de thresholds, comparação de cenários, configuração oficial, versões de modelos e Model Card. |
| Treinamento | Acompanhamento das etapas/checkpoints, cancelamento e retomada de uma tentativa interrompida. |
| Agendamento | Configuração do intervalo para gerar predições oficiais automaticamente. |

O perfil **operador** acompanha a operação e registra ações. O perfil **engenheiro** acessa também as funções de experimentação e gestão de modelos. O valor interno do papel de engenheiro é `enginner`, conforme o contrato atual do backend. As rotas de negócio da API Go exigem JWT, e as ações de engenharia possuem verificação de papel.

## Arquitetura e tecnologias

| Componente | Tecnologia | Responsabilidade |
| --- | --- | --- |
| Interface | React, Vite e CSS | Telas operacionais e laboratório de modelos. |
| Entrada HTTP | Nginx | Servir a interface e encaminhar API, Swagger e Grafana. |
| API | Go e Gin | Autenticação, casos de uso, contratos REST e decisões operacionais. |
| Persistência | PostgreSQL; ambiente configurado com Supabase | Usuários, ativos, registros, rodadas, inspeções e metadados. |
| Fila de ingestão | River sobre PostgreSQL | Jobs persistentes, retentativas e processamento fora das requisições HTTP. |
| Worker de ingestão | Go | Ler arquivos do storage e persistir os lotes na base operacional. |
| Storage de ingestão | MinIO compatível com S3 | Arquivos recebidos para processamento pelo worker. |
| API de ML | Python 3.11, Flask e Gunicorn | Consultar features e devolver probabilidades da versão carregada. |
| Worker de treinamento | Python | Executar o pipeline pesado com checkpoints e limites de recursos próprios. |
| Registry e snapshots | Outro MinIO compatível com S3 | Artefatos, manifests, métricas e dados de treinamento. |
| Observabilidade | OpenTelemetry, Tempo, Loki, Promtail, Prometheus e Grafana | Traces, logs e métricas para diagnosticar a aplicação. |
| Documentação e qualidade | Astro Starlight e GitLab CI | Portal técnico, testes, builds e verificações de segurança. |

### Fluxos principais

**Ingestão:** o frontend envia a planilha à API Go; a API grava o arquivo no MinIO de ingestão e registra upload/job em uma transação PostgreSQL. O worker River lê o objeto, processa lotes e atualiza o resultado. A limpeza do arquivo é executada após o processamento, com tratamento persistente para falhas conhecidas.

**Predição:** o Go solicita a avaliação das portas ao Flask. O serviço consulta views de leitura no PostgreSQL, carrega o modelo do registry e devolve probabilidades. O Go aplica o threshold, registra a rodada e, na publicação oficial, atualiza os diagnósticos e cria ou reutiliza inspeções preventivas. Cenários experimentais são comparados no laboratório antes de serem usados na operação.

**Treinamento:** uma solicitação cria um job para o worker Python. O worker utiliza os snapshots configurados, executa as etapas e publica uma versão candidata com artefato, manifest e métricas. O engenheiro acompanha os checkpoints e decide sobre a versão. A promoção executa uma nova rodada oficial; se a publicação falhar, o caso de uso restaura a versão anterior. Os checkpoints intermediários permanecem no volume de jobs; modelos e snapshots ficam no MinIO.

A referência inicial do serviço é o **Modelo 5, Random Forest**, com janela de **sete dias**. A versão ativa é identificada pelo manifest do registry; colocar um notebook ou `.joblib` novo em `models/` não o torna automaticamente o modelo da aplicação. Alterar o threshold muda a classificação, não retreina o modelo nem muda o horizonte para o qual ele foi treinado.

Precisão, recall e F1 apresentados no laboratório vêm da avaliação temporal do artefato para cada limiar. Eles não devem ser interpretados como resultados já observados das novas predições em produção. Consulte o [contrato interno do ML](https://git.inteli.edu.br/graduacao/2026-2a/t16/g04/-/blob/main/src/modelo/openapi.yaml) e a [explicação do serviço](https://git.inteli.edu.br/graduacao/2026-2a/t16/g04/-/blob/main/src/modelo/README.md).

## Organização do repositório

```text
.
├── src/
│   ├── frontend/          # Interface React, Vite e imagem Nginx
│   ├── backend/           # API Go, worker River, migrations e observabilidade
│   │   ├── cmd/           # Entradas da API e do worker
│   │   ├── internal/      # Domínio, casos de uso, portas e adaptadores
│   │   ├── migrations/    # Evolução versionada do banco
│   │   └── docs/          # Contrato OpenAPI
│   └── modelo/            # API Flask, worker de treinamento e registry
│       ├── ml_service/    # Domínio, aplicação, portas e adaptadores Python
│       └── artifacts/     # Artefato e manifest usados no bootstrap
├── docs/                  # Portal Astro/Starlight, organizado por sprint
├── eda/                   # Análise exploratória, hipóteses e limpeza de dados
├── models/                # Modelos 1–9, RNNs, ablação do 5.1 e pipeline 7 → 5
├── data/                  # Dados de alarmes, manutenção e outras entradas
├── scripts/               # Utilitários, incluindo auditoria de dependências
├── requirements.txt       # Ambiente principal de EDA/modelagem
├── .gitlab-ci.yml         # Pipeline de qualidade e publicação do portal
└── README.md
```

Os experimentos de `models/` usam ambientes e entradas próprios. Eles não são todos serviços da aplicação nem versões promovidas no registry. Os snapshots e artefatos necessários podem precisar ser obtidos/preparados separadamente; a disponibilidade do código não garante a disponibilidade de todos os dados.

## Requisitos para executar

Escolha o ambiente conforme a atividade:

| Atividade | Requisitos |
| --- | --- |
| Aplicação completa | Git, Docker com containers Linux, Compose v2 e PostgreSQL acessível pelos containers. No Windows, Docker Desktop com WSL 2. |
| Documentação e frontend fora do Docker | Node.js 22.12 ou superior e npm 9.6.5 ou superior. |
| Backend fora do Docker | Go 1.26, conforme `src/backend/go.mod`, e PostgreSQL. |
| Serviço de ML fora do Docker | Python 3.11 e `src/modelo/requirements.txt`. |
| EDA/modelagem | Ambiente Python indicado em `docs/dependencias-python.md`; a referência principal é Python 3.14 em Windows x86_64. |

Os Dockerfiles fornecem os ambientes da aplicação. Não é necessário instalar Go, Node.js e Python no host para a execução via Docker. **Os arquivos Compose não criam PostgreSQL**: configure previamente o banco e suas credenciais. Os exemplos de ambiente usam Supabase, que pode ser substituído por PostgreSQL compatível e acessível.

## Executar a aplicação localmente

Os comandos abaixo partem da **raiz do repositório**, em Bash. Para agentes que executam comandos neste projeto, aplicam-se as [diretrizes de RTK](https://git.inteli.edu.br/graduacao/2026-2a/t16/g04/-/blob/main/AGENTS.md).

### 1. Preparar os arquivos de ambiente

Copie os exemplos somente quando ainda não existir configuração:

```bash
test -f src/backend/.env || cp src/backend/.env.example src/backend/.env
test -f src/modelo/.env || cp src/modelo/.env.example src/modelo/.env
test -f src/frontend/.env || cp src/frontend/.env.example src/frontend/.env
```

No PowerShell, utilize `Copy-Item` apenas para os arquivos que ainda não existem. Substitua os valores ilustrativos antes de iniciar os serviços:

| Arquivo | Valores para execução local |
| --- | --- |
| `src/backend/.env` | `ENV=development`; `DATABASE_URL` e `DB_*` correspondentes ao mesmo banco; `JWT_SECRET` próprio; `RUN_MIGRATIONS=true`. |
| `src/backend/.env` | `VPS_MODEL_URL=http://ml-service:8000`; `BACKEND_BIND_ADDRESS=127.0.0.1`; `HTTP_BIND_ADDRESS=127.0.0.1`. |
| `src/backend/.env` | Credenciais `MINIO_ROOT_*` e `MINIO_APP_*` distintas; `S3_ACCESS_KEY_ID/SECRET_ACCESS_KEY` iguais à conta de aplicação desse MinIO; `S3_ENDPOINT=http://minio:9000`. |
| `src/modelo/.env` | `ENV=development`; `ML_DATABASE_URL` do banco da aplicação, com a conta de leitura das views; `ML_BIND_ADDRESS=127.0.0.1`. |
| `src/modelo/.env` | Credenciais do MinIO do modelo; `MODEL_S3_ACCESS_KEY_ID/SECRET_ACCESS_KEY` iguais à conta de aplicação desse arquivo; `MODEL_S3_ENDPOINT=http://model-minio:9000`. |
| `src/frontend/.env` | `VITE_API_URL=` vazio para usar a mesma origem do Nginx; `FRONTEND_BIND_ADDRESS=127.0.0.1`; `FRONTEND_PORT=3000`. |

Acesse a interface pelo **Nginx principal**, que encaminha `/api`. O container de frontend atende os arquivos estáticos; seu acesso direto em outra porta não equivale ao acesso completo pelo proxy. Nunca use credenciais do banco, do MinIO ou JWT em variáveis `VITE_*`, pois elas são incorporadas ao código entregue ao navegador.

Mantenha os `.env` fora do Git e use credenciais específicas do ambiente. No pooler Supabase, confira porta, modo de pooling, usuário e limite total de conexões. Senhas com caracteres especiais devem estar codificadas corretamente na URL.

### 2. Conferir os arquivos do bootstrap

Antes de iniciar o ML, verifique a disponibilidade de:

- `models/modelo-5/painel.parquet`;
- `data/Corretiva Emergencial/processados/ZM01_limpa.xlsx`;
- `src/modelo/artifacts/modelo_5.joblib` e `src/modelo/artifacts/manifest.json` compatíveis.

O `model-registry-init` valida o hash do artefato e publica os objetos iniciais. Se um caminho montado não existir ou o hash divergir, a inferência não fica pronta. A preparação/exportação é descrita no [README do serviço de ML](https://git.inteli.edu.br/graduacao/2026-2a/t16/g04/-/blob/main/src/modelo/README.md).

### 3. Iniciar frontend e backend

Crie a rede compartilhada se ainda não existir:

```bash
docker network inspect backend-network >/dev/null 2>&1 || docker network create backend-network
```

Inicie os dois componentes:

```bash
docker compose --env-file src/frontend/.env -f src/frontend/docker-compose.yml up -d --build
docker compose --env-file src/backend/.env -f src/backend/docker-compose.yml up -d --build
```

A API aplica as migrations no banco configurado. Aguarde a prontidão e confira os logs antes de preparar o ML:

```bash
docker compose --env-file src/backend/.env -f src/backend/docker-compose.yml ps
docker compose --env-file src/backend/.env -f src/backend/docker-compose.yml logs --tail=100 backend minio-init
```

No primeiro uso, execute o [exemplo de criação da conta SQL de leitura](https://git.inteli.edu.br/graduacao/2026-2a/t16/g04/-/blob/main/src/backend/migrations/create_ml_readonly_role.example.sql) como administrador, adaptando senha e nome do banco. O script cria `g04_ml_reader`, concede acesso às views e **não é uma migration automática**. Utilize essa conta em `ML_DATABASE_URL`; se ela já existir, confira seus grants em vez de tentar recriá-la.

### 4. Iniciar o serviço de ML

```bash
docker compose --env-file src/modelo/.env -f src/modelo/docker-compose.yml up -d --build
docker compose --env-file src/modelo/.env -f src/modelo/docker-compose.yml ps
```

A ordem é MinIO → criação dos buckets → bootstrap do registry → inferência e worker de treinamento. Os inicializadores `minio-init`, `model-minio-init` e `model-registry-init` são tarefas únicas: `Exited (0)` indica conclusão bem-sucedida, não um serviço que precisa ficar executando.

A subida do backend não garante que o ML já esteja disponível. Confirme `/health/ready` do ML antes de executar cenários.

### 5. Acessar e acompanhar

Com as portas padrão:

| Recurso | Endereço local |
| --- | --- |
| Aplicação | `http://localhost` |
| Swagger/OpenAPI | `http://localhost/docs` e `http://localhost/openapi.json` |
| Grafana | `http://localhost/grafana/` |
| Liveness da API | `http://localhost:8080/healthz/live` |
| Readiness da API | `http://localhost:8080/healthz/ready` |
| Readiness do ML | `http://localhost:8000/health/ready` |

O login exige uma conta do banco da aplicação. As migrations incluem contas iniciais para o ambiente do projeto; configure os acessos com a equipe e substitua credenciais de demonstração antes de liberar acesso público.

```bash
docker compose --env-file src/backend/.env -f src/backend/docker-compose.yml logs -f backend worker
docker compose --env-file src/modelo/.env -f src/modelo/docker-compose.yml logs -f model-registry-init ml-service ml-training-worker
```

Execute cada acompanhamento em seu terminal. `Ctrl+C` encerra a leitura dos logs e mantém os serviços em execução.

### 6. Parar sem apagar os dados

```bash
docker compose --env-file src/modelo/.env -f src/modelo/docker-compose.yml down
docker compose --env-file src/backend/.env -f src/backend/docker-compose.yml down
docker compose --env-file src/frontend/.env -f src/frontend/docker-compose.yml down
```

Os volumes persistem. `down -v` remove volumes e não deve ser usado como parada normal. Alterações em parâmetros `VITE_*` exigem reconstruir a imagem frontend; não basta reiniciar o container.

## Desenvolvimento sem Docker

### Frontend

Mantenha o backend disponível e defina `VITE_API_URL=http://localhost:8080` em `src/frontend/.env`. O servidor Vite encaminha `/api` para essa origem durante o desenvolvimento:

```bash
cd src/frontend
npm ci
npm run dev
```

Abra o endereço informado pelo Vite, normalmente `http://localhost:5173`. Para conferir compilação e lint, execute `npm run build` e `npm run lint` nesse diretório.

### Backend e ML

O backend possui dois pontos de entrada: `src/backend/cmd/api` e `src/backend/cmd/worker`. O serviço Python tem ambiente separado do de modelagem. Consulte o [README do ML](https://git.inteli.edu.br/graduacao/2026-2a/t16/g04/-/blob/main/src/modelo/README.md), seu [OpenAPI](https://git.inteli.edu.br/graduacao/2026-2a/t16/g04/-/blob/main/src/modelo/openapi.yaml) e a [documentação técnica do backend](https://git.inteli.edu.br/graduacao/2026-2a/t16/g04/-/blob/main/docs/src/content/docs/sprints/Sprint%203/backend/index.mdx) antes de executar esses processos individualmente: banco, migrations, registry e filas precisam estar disponíveis.

## Produção e atualização nas VPS

A organização utilizada separa a aplicação operacional da carga de ML:

| Host | Componentes |
| --- | --- |
| VPS 1 | Frontend, Nginx, API Go, worker River, MinIO de ingestão e observabilidade. |
| VPS 2 | API de ML, worker de treinamento, MinIO de modelos/snapshots e volumes de checkpoints. |
| Serviço externo | PostgreSQL configurado no Supabase, utilizado pela aplicação e pelas views de leitura do ML. |

Cada VPS compila as imagens a partir do repositório. A implantação é manual; o CI não publica as imagens da aplicação em um registry. Execute `git pull --ff-only` e os comandos Compose correspondentes ao host, com `--build`, para aplicar uma versão revisada.

O tráfego entre VPS utiliza WireGuard: a API Go aponta `VPS_MODEL_URL` para o endereço privado da VPS 2, e o ML publica sua porta no IP da interface do túnel. Os MinIO permanecem acessíveis aos containers de seus respectivos hosts. Confirme as regras de firewall e a conectividade antes de alterar o ambiente.

**Nesta versão da `main`, `ENV=production` não configura domínio, certificado HTTPS ou IPs automaticamente.** O Nginx versionado atende HTTP. Para acesso público, configure uma terminação TLS apropriada e os endereços efetivos. Também ajuste `VITE_API_URL` para a origem pública ou mantenha-o vazio quando frontend e API compartilham a mesma origem. O valor `localhost` no bundle publicado aponta para o computador do usuário, não para a VPS.

Planeje cópias recuperáveis do banco e dos objetos antes de aplicar migrations ou atualizar modelos. Volumes persistentes no host não substituem backup externo.

## Documentação

O portal Astro/Starlight é independente da aplicação; não precisa de PostgreSQL nem ML:

```bash
cd docs
npm ci
npm run dev
```

Acesse `http://localhost:4321/2026-2a/t16/g04/`, ou a porta informada no terminal. Para conferir a versão de publicação:

```bash
npm run build
npm run preview
```

A saída fica em `docs/dist`. O prefixo do site vem de `CI_PAGES_URL` no CI, com `/2026-2a/t16/g04/` como referência local. O README da raiz também é exibido na página **Solução** do portal.

## Notebooks e experimentos

Os notebooks ficam em `eda/` e `models/`. O repositório inclui modelos 1–9, variações RNN/LSTM, o experimento de ablação do Modelo 5.1 e o pipeline sequencial dos modelos 7 e 5. Consulte o notebook e seus caminhos de entrada antes de executar todas as células.

No ambiente de referência Windows, para EDA/modelagem:

```powershell
py -3.14 -m venv .venv
.venv/Scripts/python.exe -m pip install -r requirements.txt
.venv/Scripts/python.exe -m pip check
.venv/Scripts/python.exe -m jupyterlab
```

Em Linux/macOS, crie o ambiente com o Python compatível indicado na [auditoria de dependências](https://git.inteli.edu.br/graduacao/2026-2a/t16/g04/-/blob/main/docs/dependencias-python.md) e utilize `.venv/bin/python`. Construção da base, pré-processamento e treinamento precisam respeitar a ordem esperada pelo experimento.

Não misture o ambiente de notebooks com Python 3.11 do serviço de ML. O Modelo 4 usa `models/modelo-4/requirements.txt` por suas restrições de AutoML; o pipeline sequencial e o Modelo 9 também possuem arquivos de dependências próprios. Um artefato serializado precisa de ambiente compatível e manifest válido antes de ser integrado à aplicação.

## Testes e CI

| Componente | Verificação |
| --- | --- |
| Backend | `go vet ./...` e `go test -race ./...`, em `src/backend`; integrações usam `TEST_DATABASE_URL` de um banco descartável. |
| ML | Instalar `src/modelo/requirements.txt` e `pytest`, validar manifest e executar `python -m pytest tests/`, em `src/modelo`. |
| Frontend | `npm run build` e `npm run lint`, em `src/frontend`. |
| Portal | `npm run build`, em `docs`. |

Os testes de integração escrevem dados e aplicam migrations: não informe o banco operacional como banco de teste. As referências de cobertura e métricas dos modelos devem ser consultadas na execução ou artefato correspondente, sem tratar números históricos como resultados atuais.

O [GitLab CI](https://git.inteli.edu.br/graduacao/2026-2a/t16/g04/-/blob/main/.gitlab-ci.yml) executa formatação, análise Go, testes com PostgreSQL, migrations, análise de vulnerabilidades, builds de containers e scans Trivy. Os jobs de ML/frontend possuem regras por arquivos alterados. O portal é publicado pelo job Pages na branch padrão configurada no projeto; o deploy da aplicação continua manual.

## Problemas comuns

| Sintoma | O que conferir |
| --- | --- |
| Login com erro de rede | Acesso pelo Nginx, `/api` e origem compilada em `VITE_API_URL`; reconstrua o frontend após alteração. |
| Backend não fica pronto | URL do PostgreSQL, senha codificada, migrations e limites do pooler. Consulte os logs da API. |
| `model-registry-init` termina com erro | Snapshots montados, artefato/manifest e hash, existência dos buckets e credenciais S3 corretas. |
| S3 retorna `403` | Usuário de aplicação, senha, política e bucket no MinIO do host correto. |
| Modelo aparece indisponível | Resultado do bootstrap e `/health/ready` do ML; a API viva não significa modelo carregado. |
| Treinamento não utiliza novos registros | Atualização do snapshot configurado em `training-data`; ingerir planilhas no banco não recria automaticamente o painel de treinamento. |
| Porta já ocupada | Variáveis de bind/porta no `.env` e consistência da origem frontend e dos endpoints entre serviços. |

## Entregas por sprint

| Sprint | Escopo |
| --- | --- |
| 1 | Entendimento do negócio, personas, hipóteses, análise exploratória e limpeza de dados. |
| 2 | Engenharia de features, treinamento e comparação de modelos. |
| 3 | Backend, interface, contratos, integração preditiva e infraestrutura inicial. |
| 4 | Evolução do laboratório, MLOps, treinamento, observabilidade e justificativas arquiteturais. |
| 5 | Verificações finais, testes e consolidação da entrega. |

Os documentos de cada sprint registram as decisões e evidências. A presença de uma funcionalidade ou configuração não comprova, por si só, desempenho, conformidade ou conclusão da validação em produção.

## Equipe e orientação

### Integrantes

- <a href="https://www.linkedin.com/in/joseph-marcelo-mansur/">Joseph Mansur</a>
- <a href="https://www.linkedin.com/in/jo%C3%A3o-pedro-gon%C3%A7alves-corr%C3%AAa-araujo/">João Pedro Gonçalves Corrêa Araujo</a>
- <a href="https://www.linkedin.com/in/maria-vit%C3%B3ria-dos-santos/">Maria Vitória dos Santos</a>
- <a href="https://www.linkedin.com/in/maria-arielly/">Maria Arielly</a>
- <a href="https://www.linkedin.com/in/kaian-moura-56b8871b4/">Kaian Moura</a>
- <a href="https://www.linkedin.com/in/leandro-precaro-barankiewicz-filho/">Leandro Precaro Barankiewicz Filho</a>
- <a href="https://www.linkedin.com/in/jo%C3%A3ocardosodias/">João Cardoso Dias</a>

### Professores

#### Orientador(a)

- <a href="https://www.linkedin.com/in/murilo-zanini-de-carvalho-0980415b/">Murilo Zanini de Carvalho</a>

#### Instrutores

- <a href="https://www.linkedin.com/in/diogo-martins-gon%C3%A7alves-de-morais-96404732/">Diogo Martins Gonçalves de Morais</a>
- <a href="https://www.linkedin.com/in/fabiana-iegawa/">Fabiana Iegawa</a>
- <a href="https://www.linkedin.com/in/filipe-gon%C3%A7alves-08a55015b/">Filipe Gonçalves</a>
- <a href="https://www.linkedin.com/in/jefferson-o-silva/">Jefferson Silva</a>
- <a href="https://www.linkedin.com/in/marianavioli/">Mariana Vicente Oliveira</a>
- <a href="https://www.linkedin.com/in/rafaeldonairelima/">Rafael Donaire</a>

## Licença

O projeto **Lumem**, desenvolvido pelo **Grupo 4** no **Inteli — Instituto de Tecnologia e Liderança**, é disponibilizado sob a licença [CC0 1.0 Universal](https://creativecommons.org/publicdomain/zero/1.0/).

### Créditos

- [Joseph Mansur](https://www.linkedin.com/in/joseph-marcelo-mansur/)
- [João Pedro Gonçalves Corrêa Araujo](https://www.linkedin.com/in/jo%C3%A3o-pedro-gon%C3%A7alves-corr%C3%AAa-araujo/)
- [Maria Vitória dos Santos](https://www.linkedin.com/in/maria-vit%C3%B3ria-dos-santos/)
- [Maria Arielly](https://www.linkedin.com/in/maria-arielly/)
- [Kaian Moura](https://www.linkedin.com/in/kaian-moura-56b8871b4/)
- [Leandro Precaro Barankiewicz Filho](https://www.linkedin.com/in/leandro-precaro-barankiewicz-filho/)
- [João Cardoso Dias](https://www.linkedin.com/in/jo%C3%A3ocardosodias/)

Na medida permitida por lei, os autores identificados acima dedicam este trabalho ao domínio público, renunciando aos direitos autorais e direitos conexos sobre ele conforme a declaração CC0 1.0 Universal. O [texto legal completo](https://creativecommons.org/publicdomain/zero/1.0/legalcode) está disponível no site da Creative Commons.
