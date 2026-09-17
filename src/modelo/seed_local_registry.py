"""Inicializa o MinIO local com o modelo oficial e os snapshots de treinamento."""

import hashlib
import json
import os
import time
from pathlib import Path

import boto3
from botocore.config import Config
from botocore.exceptions import ClientError, EndpointConnectionError


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open('rb') as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b''):
            digest.update(chunk)
    return digest.hexdigest()


def client_from_environment():
    return boto3.client(
        's3',
        endpoint_url=os.environ['MODEL_S3_ENDPOINT'],
        region_name=os.getenv('MODEL_S3_REGION', 'us-east-1'),
        aws_access_key_id=os.environ['MODEL_S3_ACCESS_KEY_ID'],
        aws_secret_access_key=os.environ['MODEL_S3_SECRET_ACCESS_KEY'],
        config=Config(
            connect_timeout=2,
            read_timeout=5,
            retries={'max_attempts': 1, 'mode': 'standard'},
        ),
    )


def wait_for_storage(client, bucket: str, attempts: int = 20) -> None:
    print(f'aguardando object storage e bucket {bucket}...', flush=True)
    last_error = None
    for attempt in range(attempts):
        try:
            client.head_bucket(Bucket=bucket)
            print('object storage disponível', flush=True)
            return
        except (ClientError, EndpointConnectionError) as error:
            last_error = error
            if attempt == attempts - 1:
                break
            if attempt and attempt % 5 == 0:
                print(f'object storage ainda indisponível ({attempt + 1}/{attempts})', flush=True)
            time.sleep(2)
    raise RuntimeError(
        'object storage indisponível em MODEL_S3_ENDPOINT; '
        'inicie minio e minio-init antes do serviço de ML'
    ) from last_error


def object_exists(client, bucket: str, key: str) -> bool:
    try:
        client.head_object(Bucket=bucket, Key=key)
        return True
    except ClientError as error:
        status = error.response.get('ResponseMetadata', {}).get('HTTPStatusCode')
        code = str(error.response.get('Error', {}).get('Code', ''))
        if status == 404 or code in {'404', 'NoSuchKey', 'NotFound'}:
            return False
        raise


def upload_if_missing(client, bucket: str, key: str, source: Path, content_type: str) -> None:
    if object_exists(client, bucket, key):
        print(f's3://{bucket}/{key} já existe', flush=True)
        return
    print(f'enviando {source.name} para s3://{bucket}/{key}...', flush=True)
    client.upload_file(
        str(source),
        bucket,
        key,
        ExtraArgs={
            'ContentType': content_type,
            'Metadata': {'sha256': sha256_file(source)},
        },
    )
    print(f's3://{bucket}/{key} enviado', flush=True)


def seed_training_data(client) -> None:
    bucket = os.getenv('MODEL_TRAINING_BUCKET', 'training-data')
    sources = (
        (
            os.getenv('MODEL_PANEL_S3_KEY', 'datasets/current/painel.parquet'),
            Path('/bootstrap-training/painel.parquet'),
            'application/octet-stream',
        ),
        (
            os.getenv('MODEL_MAINTENANCE_S3_KEY', 'datasets/current/ZM01_limpa.xlsx'),
            Path('/bootstrap-training/ZM01_limpa.xlsx'),
            'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        ),
    )
    for key, source, content_type in sources:
        if not source.is_file():
            raise FileNotFoundError(f'snapshot inicial não encontrado: {source}')
        upload_if_missing(client, bucket, key, source, content_type)


def seed_active_model(client) -> None:
    bucket = os.getenv('MODEL_S3_BUCKET', 'artefacts')
    prefix = os.getenv('MODEL_S3_PREFIX', 'models/random-forest').strip('/')
    active_key = os.getenv('MODEL_MANIFEST_KEY', f'{prefix}/active/manifest.json')
    active_exists = object_exists(client, bucket, active_key)

    artifact = Path('/app/artifacts/modelo_5.joblib')
    manifest_path = Path('/app/artifacts/manifest.json')
    if not artifact.is_file() or not manifest_path.is_file():
        raise FileNotFoundError('artefato inicial e manifest são obrigatórios para inicializar o registry')

    manifest = json.loads(manifest_path.read_text(encoding='utf-8'))
    expected_hash = manifest.get('model_sha256')
    actual_hash = sha256_file(artifact)
    if expected_hash != actual_hash:
        raise ValueError('SHA-256 do modelo inicial difere do manifest')

    version = manifest['model_version']
    version_prefix = f'{prefix}/{version}'
    artifact_key = f'{version_prefix}/model.joblib'
    manifest_key = f'{version_prefix}/manifest.json'
    metrics_key = f'{version_prefix}/threshold-metrics.json'
    upload_if_missing(client, bucket, artifact_key, artifact, 'application/octet-stream')

    published = dict(manifest)
    published['artifact_uri'] = f's3://{bucket}/{artifact_key}'
    manifest_payload = json.dumps(published, ensure_ascii=False, indent=2).encode()
    client.put_object(Bucket=bucket, Key=manifest_key, Body=manifest_payload, ContentType='application/json')

    metrics_payload = json.dumps(published.get('threshold_metrics', []), ensure_ascii=False).encode()
    client.put_object(Bucket=bucket, Key=metrics_key, Body=metrics_payload, ContentType='application/json')
    if not active_exists:
        client.put_object(Bucket=bucket, Key=active_key, Body=manifest_payload, ContentType='application/json')


def seed_candidates(client) -> None:
    bucket = os.getenv('MODEL_S3_BUCKET', 'artefacts')
    prefix = os.getenv('MODEL_S3_PREFIX', 'models/random-forest').strip('/')
    candidates_root = Path('/app/artifacts/models')
    if not candidates_root.is_dir():
        return

    for version_dir in candidates_root.iterdir():
        artifact = version_dir / 'model.joblib'
        manifest_path = version_dir / 'manifest.json'
        if not version_dir.is_dir() or not artifact.is_file() or not manifest_path.is_file():
            continue

        manifest = json.loads(manifest_path.read_text(encoding='utf-8'))
        version = manifest.get('model_version')
        if version != version_dir.name or manifest.get('model_sha256') != sha256_file(artifact):
            raise ValueError(f'candidato local inválido: {version_dir.name}')

        version_prefix = f'{prefix}/{version}'
        artifact_key = f'{version_prefix}/model.joblib'
        upload_if_missing(client, bucket, artifact_key, artifact, 'application/octet-stream')
        published = dict(manifest)
        published['artifact_uri'] = f's3://{bucket}/{artifact_key}'
        payload = json.dumps(published, ensure_ascii=False, indent=2).encode()
        client.put_object(
            Bucket=bucket,
            Key=f'{version_prefix}/manifest.json',
            Body=payload,
            ContentType='application/json',
        )

        metrics_path = version_dir / 'threshold-metrics.json'
        if metrics_path.is_file():
            upload_if_missing(
                client,
                bucket,
                f'{version_prefix}/threshold-metrics.json',
                metrics_path,
                'application/json',
            )


def main() -> None:
    client = client_from_environment()
    wait_for_storage(client, os.getenv('MODEL_S3_BUCKET', 'artefacts'))
    print('verificando snapshots de treinamento...', flush=True)
    seed_training_data(client)
    print('verificando modelo ativo...', flush=True)
    seed_active_model(client)
    print('verificando modelos candidatos...', flush=True)
    seed_candidates(client)
    print('registry local e snapshots inicializados com sucesso', flush=True)


if __name__ == '__main__':
    main()
