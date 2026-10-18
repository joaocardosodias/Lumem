FROM python:3.11-slim

WORKDIR /app

RUN apt-get update && \
    apt-get upgrade -y && \
    rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt 'msgpack>=1.2.1' && \
    pip uninstall -y setuptools wheel

COPY app.py database.py pipeline.py exportar_modelo.py seed_local_registry.py training_worker.py ./
COPY ml_service ./ml_service
COPY openapi.yaml ./openapi.yaml
COPY artifacts ./artifacts

RUN useradd -u 10001 -m appuser
RUN mkdir -p /var/lib/g04/training-jobs && \
    chown -R appuser:appuser /var/lib/g04 /app/artifacts
USER 10001

EXPOSE 8000

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8000/health/ready')"

CMD ["gunicorn", "--bind", "0.0.0.0:8000", "--workers", "1", "--threads", "4", "--timeout", "60", "app:app"]
