# Dependencias Python

## Escopo da auditoria

A auditoria examina os imports por AST em todos os scripts `.py` e nas celulas
de codigo dos notebooks `.ipynb` versionados ou ainda nao ignorados pelo Git.
Nao executa notebooks nem considera imports presentes apenas em saidas antigas.
Tambem verifica imports dinamicos com nomes literais e os motores de Excel
(`openpyxl`) e Parquet (`pyarrow`) usados pelo pandas.

Foram examinados 60 arquivos, incluindo o proprio auditor, e 768 celulas de
codigo/scripts. Modulos da biblioteca padrao e os modulos locais `app` e
`ml_service` nao sao pacotes a instalar. Celulas que nao podem ser analisadas
geram erro, em vez de serem silenciosamente descartadas.

O arquivo anterior da raiz era um congelamento de um ambiente de uso geral:
continha pacotes de sistema como `btrfsutil`, `cupshelpers` e `ufw`, ferramentas
de audio/video e uma referencia `TBB @ file:///build/...`. Esses itens nao foram
encontrados no codigo e foram removidos. As dependencias do projeto agora usam
versoes exatas (`==`).

## Ambientes

| Arquivo | Uso | Python de referencia |
| --- | --- | --- |
| `requirements.txt` | EDA e modelos, exceto AutoML do Modelo 4 | 3.14, Windows x86_64 |
| `models/modelo-4/requirements.txt` | Modelo 4 com FLAML AutoML | 3.14, Windows x86_64 |
| `models/pipeline-sequencial/requirements.txt` | Subconjunto para o pipeline 7 + 5 | 3.14 |
| `src/modelo/requirements.txt` | Servico de inferencia e exportacao | 3.11, Linux/Docker |
| `src/modelo/requirements-test.txt` | Dependencias do servico mais pytest | 3.11, Linux/Docker |

O FLAML 2.7.0 declara `xgboost<3.0.0` no extra `automl`; o extra `blendsearch`
declara `optuna<=3.6.1`. Por isso o Modelo 4 usa XGBoost 2.1.4 e Optuna 3.6.1
em ambiente separado. A raiz preserva XGBoost 3.4.0, utilizado na reexecucao do
Modelo 1. Nao instale os dois arquivos no mesmo ambiente.

O servico permanece com scikit-learn 1.4.2, pandas 2.2.2 e NumPy 1.26.4,
conforme seu ambiente existente. A modelagem usa scikit-learn 1.9.0, pandas
3.0.5 e NumPy 2.5.2, alinhado ao pipeline sequencial. Esta auditoria nao migra
artefatos serializados entre essas versoes nem altera o ambiente de producao.

## Dependencias e evidencias

| Dependencias | Uso observado |
| --- | --- |
| pandas, numpy | Preparacao de dados e modelagem em EDA, modelos e servico |
| scipy | Hipoteses estatisticas e engenharia de atributos |
| scikit-learn | Arvores, Random Forest, ExtraTrees, HistGradientBoosting, ensembles e metricas |
| xgboost | Modelos 1 e 6; estimadores do AutoML no Modelo 4 |
| lightgbm | Modelo 2 e AutoML |
| catboost | Modelo 7 e pipeline sequencial |
| flaml[automl,blendsearch] | AutoML do Modelo 4, incluindo suporte ao otimizador |
| shap | Explicabilidade nos modelos 1, 5 e 7 e na exportacao do servico |
| optuna | Incluido conforme solicitado; tambem requerido pelo extra blendsearch do FLAML. Nao ha import direto no codigo atual |
| torch | Notebooks RNN/LSTM e arquitetura LSTM exportada |
| matplotlib, seaborn | Graficos exploratorios e de avaliacao |
| openpyxl, pyarrow | Leitura/escrita de Excel e Parquet |
| joblib | Serializacao e carregamento de modelos |
| ipython | display e utilitarios usados pelos notebooks |
| ipykernel, jupyterlab, nbformat, nbclient, nbconvert | Kernel, interface e execucao dos notebooks |
| flask, gunicorn | API e processo servidor no Docker |
| sqlalchemy, psycopg2-binary | Acesso ao PostgreSQL, incluindo driver carregado pela URL |
| boto3, botocore | Registro de modelos e armazenamento de objetos |
| pytest | Testes do servico; disponivel tambem no ambiente de modelagem |
| packaging | Leitura e verificacao dos requisitos pelo auditor |

As versoes disponiveis no ambiente local e os arquivos existentes serviram de
referencia. Dependencias ausentes foram resolvidas pelo pip no PyPI. Os pins
definem uma base de instalacao; nao comprovam qual versao foi usada em todos os
experimentos historicos, que nao possuem registro completo do ambiente.

## Instalacao

Na raiz, em PowerShell, para modelagem:

```powershell
py -3.14 -m venv .venv
.venv/Scripts/python.exe -m pip install -r requirements.txt
.venv/Scripts/python.exe -m pip check
.venv/Scripts/python.exe -m jupyterlab
```

Para o Modelo 4, use um ambiente separado dentro da pasta ignorada `.venv`:

```powershell
py -3.14 -m venv .venv/modelo4
.venv/modelo4/Scripts/python.exe -m pip install -r models/modelo-4/requirements.txt
.venv/modelo4/Scripts/python.exe -m ipykernel install --user --name g04-modelo4
```

Selecione o kernel `g04-modelo4` para esse notebook. Para o servico, use o
Dockerfile de `src/modelo`, que instala seu proprio arquivo de dependencias.

Os arquivos de requisitos fixam apenas as dependencias principais de cada
ambiente. O pip instala automaticamente as dependencias transitivas exigidas
por elas; suas versoes podem variar entre instalacoes.

## Repetir a auditoria

```powershell
.venv/Scripts/python.exe scripts/auditar_dependencias.py
.venv/Scripts/python.exe scripts/auditar_dependencias.py --json
```

O modo JSON mostra os arquivos e as celulas que usam cada dependencia. O
comando retorna codigo diferente de zero quando encontra import sem pacote
declarado, requisito sem versao exata, pins conflitantes ou erro de analise.
O auditor verifica cobertura estatica; a instalacao e `pip check` verificam
as dependencias declaradas pelos pacotes. Esses controles nao substituem a
execucao dos experimentos e a validacao das metricas de cada modelo.

## Validacao realizada

- Auditoria estatica: 60 arquivos e 768 celulas/scripts, sem imports descobertos
  fora dos requisitos do respectivo ambiente.
- Ambiente principal instalado do zero em `.venv`; `pip check` sem conflitos.
- Imports das bibliotecas principais, treinamento e inferencia pequenos com
  XGBoost, LightGBM e CatBoost, explicacao SHAP, dois trials do Optuna e uma
  passagem por uma LSTM do PyTorch concluidos.
- Seis testes da avaliacao de limiares do Modelo 1 aprovados no novo ambiente.
- Modelo 4: resolucao de todas as dependencias com `pip install --dry-run
  --ignore-installed --only-binary=:all:` aprovada, sem treinamento do AutoML.
- Servico: resolucao com Python 3.11 e wheels Linux manylinux2014/manylinux_2_28
  aprovada. Esta verificacao cruzada nao equivale a executar o container.

Os notebooks completos nao foram reexecutados nesta auditoria de dependencias.
