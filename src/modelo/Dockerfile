FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt 'msgpack>=1.2.1' && \
    pip uninstall -y setuptools wheel

COPY app.py database.py pipeline.py ./
COPY artifacts ./artifacts

RUN useradd -u 10001 -m appuser
USER 10001

EXPOSE 8000

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8000/health')"

CMD ["gunicorn", "--bind", "0.0.0.0:8000", "--workers", "2", "--timeout", "10", "app:app"]
