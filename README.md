# Inteli - Instituto de Tecnologia e Liderança

<p align="center">
<a href= "https://www.inteli.edu.br/"><img src="https://res.cloudinary.com/dwewomj84/image/upload/v1761355911/inteli_lfuayp.png" alt="Inteli - Instituto de Tecnologia e Liderança" border="0" width=40% height=40%></a>
</p>

<br>

# Lumen - Manutenção Preditiva de Portas de Plataforma (PSDs)

## Subway Surfers

## Integrantes:

- <a href="https://www.linkedin.com/in/joseph-marcelo-mansur/">Joseph Mansur</a>
- <a href="https://www.linkedin.com/in/jo%C3%A3o-pedro-gon%C3%A7alves-corr%C3%AAa-araujo/">João Pedro Gonçalves Corrêa Araujo</a>
- <a href="https://www.linkedin.com/in/maria-vit%C3%B3ria-dos-santos/">Maria Vitória dos Santos</a>
- <a href="https://www.linkedin.com/in/maria-arielly/">Maria Arielly</a>
- <a href="https://www.linkedin.com/in/kaian-moura-56b8871b4/">Kaian Moura</a>
- <a href="https://www.linkedin.com/in/leandro-precaro-barankiewicz-filho/">Leandro Precaro Barankiewicz Filho</a>
- <a href="https://www.linkedin.com/in/jo%C3%A3ocardosodias/">João Cardoso Dias</a>

## Professores:

### Orientador(a)

- <a href="https://www.linkedin.com/in/murilo-zanini-de-carvalho-0980415b/">Murilo Zanini de Carvalho</a>

### Instrutores

- <a href="https://www.linkedin.com/in/diogo-martins-gon%C3%A7alves-de-morais-96404732/">Diogo Martins Gonçalves de Morais</a>
- <a href="https://www.linkedin.com/in/fabiana-iegawa/">Fabiana Iegawa</a>
- <a href="https://www.linkedin.com/in/filipe-gon%C3%A7alves-08a55015b/">Filipe Gonçalves</a>
- <a href="https://www.linkedin.com/in/jefferson-o-silva/">Jefferson Silva</a>
- <a href="https://www.linkedin.com/in/marianavioli/">Mariana Vicente Oliveira</a>
- <a href="https://www.linkedin.com/in/rafaeldonairelima/">Rafael Donaire</a>

## Descrição

&emsp; Este projeto, desenvolvido em parceria com a **Motiva** (maior empresa de infraestrutura de mobilidade do Brasil), tem como objetivo construir uma solução de **Manutenção Preditiva** para as Portas de Plataforma (*Platform Screen Doors* — PSDs) operadas nas linhas de metrô sob concessão da concessionária.<br><br>&emsp; Historicamente, a manutenção de ativos ferroviários e metroviários atua de forma preventiva periódica e corretiva emergencial quando ocorrem falhas ou alarmes em operação. O objetivo deste trabalho é realizar a virada de chave para a manutenção preditiva orientada a dados: através do tratamento de dados históricos de logs de alarmes, ordens do SAP (ZM01, ZM02 e ZM03) e movimentação de passageiros, o modelo de Inteligência Artificial identifica padrões de degradação nos componentes mecânicos e elétricos (como sensores `LSD` e roletes), antecipando falhas iminentes. A solução visa otimizar as janelas noturnas de manutenção (4 horas), aumentar a segurança operacional e reduzir paradas não programadas no transporte público.

## Documentação Adicional (Deploy do Astro / Starlight)

Clique [aqui](https://graduacao.pages.git.inteli.edu.br/2026-2a/t16/g04/) para acessar a documentação completa do projeto.

## Estrutura de pastas

```
├── data/ --> Bases de dados brutas, tratadas e modelos serializados.
│   ├── Alarmes/
│   ├── Catracas/
│   ├── Corretiva Emergencial/
│   ├── Corretiva Programada/
│   ├── Preventiva/
│   └── modelos_artefatos/ --> Modelos treinados (.joblib, .cbm, .json).
├── docs/ --> Código-fonte do portal de documentação estática (Astro / Starlight).
│   ├── src/
│   │   ├── assets/
│   │   └── content/docs/
│   └── astro.config.mjs
├── eda/ --> Notebooks de Análise Exploratória de Dados, teste de hipóteses e scripts de limpeza.
│   ├── cleaning/ --> Pipelines de limpeza de dados da Sprint 1.
│   └── hipoteses/ --> Validação estatística das 7 hipóteses de falha.
├── models/ --> Diretório centralizado e padronizado dos modelos preditivos (Sprint 2).
│   ├── modelo-1/ --> Modelo 1: XGBoost & Ensembles.
│   ├── modelo-2/ --> Modelo 2: LightGBM & Random Forest Baseline.
│   ├── modelo-3/ --> Modelo 3: Árvore de Decisão CRISP-DM.
│   ├── modelo-5/ --> Modelo 5: Random Forest por Evento e Ciclos.
│   ├── modelo-6/ --> Modelo 6: XGBoost por Evento/Estação.
│   └── modelo-7/ --> Modelo 7: CatBoostClassifier (Plataforma × Dia).
├── requirements.txt --> Dependências Python do projeto.
└── README.md --> Arquivo guia de apresentação do projeto (o mesmo que você está lendo agora).
```

## Requisitos

O projeto possui três formas de execução: a aplicação Lumen, o portal de documentação e os notebooks de modelagem. Instale apenas as ferramentas necessárias ao fluxo escolhido.

| Ferramenta ou recurso | Quando é necessário |
| --- | --- |
| Git | Obter e atualizar o repositório |
| Docker com suporte a contêineres Linux e Docker Compose v2 | Executar a aplicação completa; no Windows, utilizar Docker Desktop com WSL 2 |
| PostgreSQL acessível pelos contêineres | Banco da aplicação; os exemplos usam Supabase. Os arquivos Compose não criam um servidor PostgreSQL |
| Node.js 22.12 ou superior e npm 9.6.5 ou superior | Executar o portal Astro e desenvolver o frontend React fora do Docker |
| Python 3.14 | Executar os notebooks com as dependências principais fixadas no `requirements.txt` da raiz |
| Go 1.26.0 | Apenas para executar ou testar o backend fora do Docker |
| Python 3.11 | Apenas para executar o serviço de ML fora do Docker; a imagem já fornece essa versão |

Os contêineres compilam o backend e o frontend e instalam as dependências do serviço de ML. Portanto, Node.js, Go e Python não precisam estar instalados no host para executar a aplicação via Docker.

## Como Rodar a Aplicação

Execute os comandos desta seção **na raiz do repositório**. A aplicação reúne o frontend React, a API Go, os workers, o serviço de ML, o armazenamento MinIO e os serviços de observabilidade.

### 1. Preparar as variáveis de ambiente

Na primeira configuração, crie os arquivos a partir dos exemplos. Preserve os arquivos `.env` que já estiverem configurados.

No PowerShell:

```powershell
Copy-Item src/backend/.env.example src/backend/.env
Copy-Item src/modelo/.env.example src/modelo/.env
Copy-Item src/frontend/.env.example src/frontend/.env
```

No Linux ou macOS, utilize `cp` no lugar de `Copy-Item`.

Preencha os arquivos antes de iniciar os contêineres:

| Arquivo | Configuração necessária |
| --- | --- |
| `src/backend/.env` | Configure `DATABASE_URL` e os campos `DB_*` para o mesmo PostgreSQL, além de `JWT_SECRET`. Mantenha `RUN_MIGRATIONS=true` para aplicar as migrações e `VPS_MODEL_URL=http://ml-service:8000` para comunicação local entre contêineres |
| `src/backend/.env` | Defina as credenciais `MINIO_ROOT_*` e `MINIO_APP_*`; faça `S3_ACCESS_KEY_ID` e `S3_SECRET_ACCESS_KEY` corresponderem às credenciais da aplicação no MinIO |
| `src/modelo/.env` | Configure `ML_DATABASE_URL` para o banco da aplicação, com o usuário de leitura preparado conforme `src/backend/migrations/create_ml_readonly_role.example.sql` |
| `src/modelo/.env` | Defina as credenciais do MinIO do modelo; `MODEL_S3_ACCESS_KEY_ID` e `MODEL_S3_SECRET_ACCESS_KEY` devem corresponder a `MINIO_APP_USER` e `MINIO_APP_PASSWORD` desse arquivo. Mantenha `MODEL_S3_ENDPOINT=http://model-minio:9000` |
| `src/frontend/.env` | Para acessar a aplicação pelo Nginx, mantenha `VITE_API_URL=http://localhost`, `FRONTEND_BIND_ADDRESS=127.0.0.1` e `FRONTEND_PORT=3000` |

Para execução local, use `ENV=development` e mantenha os binds do backend e do ML em `127.0.0.1`. Defina também `HTTP_BIND_ADDRESS=127.0.0.1` no backend para acessar o Nginx apenas pelo próprio computador. Os valores de exemplo de senhas e banco precisam ser substituídos; não versione os arquivos `.env`.

O primeiro boot do serviço de ML utiliza `models/modelo-5/painel.parquet`, `data/Corretiva Emergencial/processados/ZM01_limpa.xlsx` e os artefatos em `src/modelo/artifacts/` para inicializar o registry. Confira se os dados e o modelo treinado estão disponíveis. A preparação e exportação estão descritas em `src/modelo/README.md`; um checkout sem os arquivos binários necessários não basta para iniciar a inferência.

### 2. Criar a rede e iniciar os serviços

Crie a rede compartilhada uma única vez:

```bash
docker network create backend-network
```

Se ela já existir, prossiga. Inicie o frontend e o backend:

```bash
docker compose --env-file src/frontend/.env -f src/frontend/docker-compose.yml up -d --build
docker compose --env-file src/backend/.env -f src/backend/docker-compose.yml up -d --build
```

No primeiro uso do banco, aguarde as migrações do backend e execute `src/backend/migrations/create_ml_readonly_role.example.sql` como administrador no PostgreSQL, adaptando a senha e o nome do banco. Esse script não é uma migração automática: ele cria `g04_ml_reader` e concede leitura às views do ML. Use esse mesmo usuário em `ML_DATABASE_URL` no lugar do nome ilustrativo do `.env.example`. Se o usuário já existir, confira suas permissões em vez de recriá-lo.

Depois, inicie o serviço de ML:

```bash
docker compose --env-file src/modelo/.env -f src/modelo/docker-compose.yml up -d --build
```

O backend aplica as migrações; os inicializadores do MinIO criam os buckets e o serviço de ML publica os artefatos iniciais. Aguarde esses processos e os healthchecks. Os serviços `minio-init`, `model-minio-init` e `model-registry-init` são tarefas de inicialização e podem aparecer como `Exited (0)` após concluírem com sucesso.

### 3. Acessar e conferir a execução

Com as portas padrão dos exemplos:

| Serviço | Endereço |
| --- | --- |
| Aplicação Lumen pelo Nginx | `http://localhost` |
| Frontend diretamente | `http://localhost:3000` |
| Swagger da API | `http://localhost/docs` |
| Prontidão do backend | `http://localhost:8080/healthz/ready` |
| Prontidão do serviço de ML | `http://localhost:8000/health/ready` |
| Grafana | `http://localhost/grafana/` |

O login usa uma conta cadastrada no banco da aplicação. As migrações incluem usuários iniciais; consulte a configuração de acesso da equipe.

Para conferir o estado e acompanhar os logs:

```bash
docker compose --env-file src/backend/.env -f src/backend/docker-compose.yml ps
docker compose --env-file src/modelo/.env -f src/modelo/docker-compose.yml ps
docker compose --env-file src/backend/.env -f src/backend/docker-compose.yml logs -f backend worker
```

Para investigar o ML, use `logs -f model-registry-init ml-service ml-training-worker` no comando Compose de `src/modelo`. `Ctrl+C` encerra o acompanhamento dos logs, mantendo os contêineres ativos.

### 4. Parar a aplicação

```bash
docker compose --env-file src/frontend/.env -f src/frontend/docker-compose.yml down
docker compose --env-file src/backend/.env -f src/backend/docker-compose.yml down
docker compose --env-file src/modelo/.env -f src/modelo/docker-compose.yml down
```

Esses comandos preservam os volumes de dados. Se uma porta estiver ocupada, ajuste a variável correspondente no `.env`; ao alterar a porta HTTP do Nginx, atualize também `VITE_API_URL` e reconstrua o frontend.

### Desenvolvimento do frontend sem Docker

Mantenha o backend e o serviço de ML disponíveis. Em `src/frontend/.env`, defina `VITE_API_URL=http://localhost:8080` para utilizar diretamente a API local. Execute, a partir da raiz:

```bash
cd src/frontend
npm ci
npm run dev
```

Acesse o endereço informado pelo Vite, normalmente `http://localhost:5173`. Para gerar a versão de distribuição, execute `npm run build`; a saída fica em `src/frontend/dist`. Mudanças em `VITE_API_URL` exigem reiniciar o servidor de desenvolvimento ou reconstruir a imagem do frontend.

## Como Rodar a Documentação

O portal Astro é independente da aplicação e não exige banco, Docker ou serviço de ML. A partir da raiz:

```bash
cd docs
npm ci
npm run dev
```

Acesse `http://localhost:4321/2026-2a/t16/g04/`. Se a porta estiver ocupada, utilize a porta exibida no terminal. Para encerrar, pressione `Ctrl+C`.

Para compilar e conferir a versão de publicação, execute dentro de `docs`:

```bash
npm run build
npm run preview
```

O build gera `docs/dist`. Use o endereço informado pelo preview, mantendo o prefixo `/2026-2a/t16/g04/`. No GitLab CI, o prefixo pode ser definido por `CI_PAGES_URL`.

## Como Rodar os Notebooks

Os notebooks estão em `eda/` e `models/`. Para o ambiente principal de modelagem, execute na raiz, usando PowerShell:

```powershell
py -3.14 -m venv .venv
.venv/Scripts/python.exe -m pip install -r requirements.txt
.venv/Scripts/python.exe -m pip check
.venv/Scripts/python.exe -m jupyterlab
```

No Linux ou macOS, crie o ambiente com `python3.14 -m venv .venv` e substitua `.venv/Scripts/python.exe` por `.venv/bin/python` nos comandos seguintes. As versões do ambiente principal foram validadas em Windows; confira a disponibilidade das dependências no seu sistema.

Abra o notebook desejado no JupyterLab. Nos modelos com etapas numeradas, execute construção da base, pré-processamento e treinamento nessa ordem. Os arquivos de entrada esperados ficam em `data/` e nas pastas dos modelos.

O Modelo 4 (FLAML) requer um ambiente separado, definido em `models/modelo-4/requirements.txt`, devido à versão do XGBoost. O serviço de inferência usa Python 3.11 e seu próprio `src/modelo/requirements.txt`; não misture esses ambientes com o da raiz. Os arquivos fixam as dependências principais, e o pip resolve as transitivas automaticamente. Consulte `docs/dependencias-python.md` para a auditoria e os comandos específicos de cada ambiente.

## Histórico de lançamentos

- 0.1.0 - 14/08/2026
  - [sprint 1] Entendimento do negócio (Motiva), alinhamento com ODS, definição de personas (Ronaldo e Marta), formulação e validação estatística de 7 hipóteses de falha, pipelines de limpeza de dados (Alarmes, ZM01, ZM02, ZM03) e definição das features candidatas.
- 0.2.0 - 27/08/2026
  - [sprint 2] Consolidação da Tabela Mestra Contínua, engenharia de features (32+ atributos), blindagem contra vazamento temporal (*data leakage*), treinamento de 5 arquiteturas de Machine Learning (Árvores de Decisão, Random Forest, XGBoost, LightGBM, CatBoost e Ensembles), calibração de hiperparâmetros, avaliação sob métricas estritas de manutenção e explicabilidade global com SHAP.
- 0.3.0 - 11/09/2026
  - [sprint 3] Refinamento dos modelos, otimização de limiares operacionais e avaliação de impacto financeiro.
- 0.4.0 - 25/09/2026
  - [sprint 4] Integração do modelo com a interface visual e relatórios operacionais.
- 0.5.0 - 09/10/2026
  - [sprint 5] Entrega final do sistema preditivo e validação de impacto operacional.

## Licença

<img style="height:22px!important;margin-left:3px;vertical-align:text-bottom;" src="https://mirrors.creativecommons.org/presskit/icons/cc.svg?ref=chooser-v1"><img style="height:22px!important;margin-left:3px;vertical-align:text-bottom;" src="https://mirrors.creativecommons.org/presskit/icons/by.svg?ref=chooser-v1"><p xmlns:cc="http://creativecommons.org/ns#" xmlns:dct="http://purl.org/dc/terms/">GRUPO 4 by [Inteli](https://www.inteli.edu.br/), [Joseph Mansur](https://www.linkedin.com/in/joseph-marcelo-mansur/), [João Pedro Gonçalves Corrêa Araujo](https://www.linkedin.com/in/jo%C3%A3o-pedro-gon%C3%A7alves-corr%C3%AAa-araujo/), [Maria Vitória dos Santos](https://www.linkedin.com/in/maria-vit%C3%B3ria-dos-santos/), [Maria Arielly](https://www.linkedin.com/in/maria-arielly/), [Kaian Moura](https://www.linkedin.com/in/kaian-moura-56b8871b4/), [Leandro Precaro Barankiewicz Filho](https://www.linkedin.com/in/leandro-precaro-barankiewicz-filho/), [João Cardoso Dias](https://www.linkedin.com/in/jo%C3%A3ocardosodias/) is licensed under <a href="http://creativecommons.org/licenses/by/4.0/?ref=chooser-v1" target="_blank" rel="license noopener noreferrer" style="display:inline-block;">Attribution 4.0 International</a>.</p>

