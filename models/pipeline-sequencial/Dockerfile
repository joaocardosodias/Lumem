FROM python:3.14-slim@sha256:cad9a2c871761c413caa6fdd6441c783451e740a48aaeba60ae62a8b53525ef6

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    DEBIAN_FRONTEND=noninteractive

WORKDIR /app

# O CatBoost utiliza OpenMP em tempo de execução.
RUN apt-get update \
    && apt-get install --yes --no-install-recommends libgomp1 \
    && rm -rf /var/lib/apt/lists/*

COPY models/pipeline-sequencial/requirements.txt /tmp/requirements.txt
RUN python -m pip install --upgrade pip \
    && python -m pip install --requirement /tmp/requirements.txt

COPY models/modelo-5/painel.parquet models/modelo-5/painel.parquet
COPY models/modelo-7/modelo_catboost.cbm models/modelo-7/modelo_catboost.cbm
COPY models/modelo-7/metadata_modelo.json models/modelo-7/metadata_modelo.json
COPY data/processados/modelo-7/tabela_mestra_psd.csv data/processados/modelo-7/tabela_mestra_psd.csv
COPY ["data/Corretiva Emergencial/processados/ZM01_limpa.xlsx", "data/Corretiva Emergencial/processados/ZM01_limpa.xlsx"]

COPY models/pipeline-sequencial/executar_pipeline.py models/pipeline-sequencial/executar_pipeline.py
COPY models/pipeline-sequencial/pipeline_sequencial_modelos_7_5.ipynb models/pipeline-sequencial/pipeline_sequencial_modelos_7_5.ipynb
COPY models/pipeline-sequencial/artefatos/metadata_modelo_5.json models/pipeline-sequencial/artefatos/metadata_modelo_5.json
COPY models/pipeline-sequencial/artefatos/modelo_5_com_score_modelo_7.joblib models/pipeline-sequencial/artefatos/modelo_5_com_score_modelo_7.joblib
COPY models/pipeline-sequencial/artefatos/metadata_modelo_5_com_score_modelo_7.json models/pipeline-sequencial/artefatos/metadata_modelo_5_com_score_modelo_7.json

ENTRYPOINT ["python", "models/pipeline-sequencial/executar_pipeline.py"]
CMD ["--help"]
