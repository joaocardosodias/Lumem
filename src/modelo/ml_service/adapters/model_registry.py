import hashlib
import json
import os
from pathlib import Path
import shutil
import threading
from urllib.parse import urlparse

from ml_service.ports.model_registry import ModelRegistry


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open('rb') as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b''):
            digest.update(chunk)
    return digest.hexdigest()


class LocalModelRegistry(ModelRegistry):
    def __init__(self, artifact_path: str, manifest_path: str | None = None):
        self.artifact_path = Path(artifact_path).resolve()
        self.manifest_path = Path(manifest_path).resolve() if manifest_path else self.artifact_path.with_name('manifest.json')

    def materialize_active(self) -> tuple[Path, dict]:
        if not self.artifact_path.is_file():
            raise FileNotFoundError(f'artefato não encontrado: {self.artifact_path}')
        manifest = {}
        if self.manifest_path.is_file():
            manifest = json.loads(self.manifest_path.read_text(encoding='utf-8'))
        expected_hash = manifest.get('model_sha256')
        actual_hash = sha256_file(self.artifact_path)
        if expected_hash and expected_hash != actual_hash:
            raise ValueError('SHA-256 do artefato local difere do manifest')
        manifest.setdefault('model_sha256', actual_hash)
        manifest.setdefault('model_version', os.getenv('MODEL_VERSION', 'v1.0.0'))
        manifest.setdefault('artifact_uri', self.artifact_path.as_uri())
        return self.artifact_path, manifest

    def materialize_version(self, version: str) -> tuple[Path, dict]:
        version_dir = self.artifact_path.parent / 'models' / version
        candidate_artifact = version_dir / 'model.joblib'
        candidate_manifest = version_dir / 'manifest.json'
        if not candidate_artifact.is_file() or not candidate_manifest.is_file():
            raise FileNotFoundError(f'versão local não encontrada: {version}')
        manifest = json.loads(candidate_manifest.read_text(encoding='utf-8'))
        if manifest.get('model_version') != version or sha256_file(candidate_artifact) != manifest.get('model_sha256'):
            raise ValueError('candidato local possui manifest ou hash inválido')
        return candidate_artifact, manifest

    def activate(self, version: str) -> dict:
        candidate_artifact, manifest = self.materialize_version(version)
        temporary = self.artifact_path.with_suffix('.next')
        shutil.copyfile(candidate_artifact, temporary)
        temporary.replace(self.artifact_path)
        manifest['artifact_uri'] = self.artifact_path.resolve().as_uri()
        self.manifest_path.write_text(json.dumps(manifest, indent=2, ensure_ascii=False), encoding='utf-8')
        return manifest


class S3ModelRegistry(ModelRegistry):
    def __init__(self):
        import boto3

        self.bucket = os.getenv('MODEL_S3_BUCKET') or os.environ['S3_BUCKET']
        self.manifest_key = os.getenv('MODEL_MANIFEST_KEY', 'models/random-forest/active/manifest.json')
        self.cache_dir = Path(os.getenv('MODEL_CACHE_DIR', '/tmp/g04-model-cache'))
        self.cache_dir.mkdir(parents=True, exist_ok=True)
        self._lock = threading.RLock()
        self._active: tuple[Path, dict] | None = None
        self.client = boto3.client(
            's3',
            endpoint_url=os.getenv('MODEL_S3_ENDPOINT') or os.getenv('S3_ENDPOINT'),
            region_name=os.getenv('MODEL_S3_REGION') or os.getenv('S3_REGION', 'us-west-2'),
            aws_access_key_id=os.getenv('MODEL_S3_ACCESS_KEY_ID') or os.getenv('S3_ACCESS_KEY_ID'),
            aws_secret_access_key=os.getenv('MODEL_S3_SECRET_ACCESS_KEY') or os.getenv('S3_SECRET_ACCESS_KEY'),
        )

    def materialize_active(self) -> tuple[Path, dict]:
        with self._lock:
            if self._active is not None:
                path, manifest = self._active
                if path.is_file() and sha256_file(path) == manifest.get('model_sha256'):
                    return path, dict(manifest)

            manifest_object = self.client.get_object(Bucket=self.bucket, Key=self.manifest_key)
            manifest = json.loads(manifest_object['Body'].read())
            target = self._materialize_manifest(manifest)
            self._active = (target, manifest)
            return target, dict(manifest)

    def _materialize_manifest(self, manifest: dict) -> Path:
        version = manifest.get('model_version')
        expected_hash = manifest.get('model_sha256')
        artifact_uri = manifest.get('artifact_uri')
        if not version or not isinstance(expected_hash, str) or len(expected_hash) != 64 or not artifact_uri:
            raise ValueError('manifest deve informar model_version, model_sha256 e artifact_uri válidos')

        parsed = urlparse(artifact_uri)
        if parsed.scheme != 's3' or parsed.netloc != self.bucket:
            raise ValueError('artifact_uri do manifest não pertence ao bucket configurado')
        artifact_key = parsed.path.lstrip('/')
        target = self.cache_dir / version / 'model.joblib'
        target.parent.mkdir(parents=True, exist_ok=True)

        if not target.is_file() or sha256_file(target) != expected_hash:
            temporary = target.with_suffix('.download')
            self.client.download_file(self.bucket, artifact_key, str(temporary))
            if sha256_file(temporary) != expected_hash:
                temporary.unlink(missing_ok=True)
                raise ValueError('SHA-256 do artefato baixado difere do manifest')
            temporary.replace(target)
        return target

    def materialize_version(self, version: str) -> tuple[Path, dict]:
        with self._lock:
            prefix = os.getenv('MODEL_S3_PREFIX', 'models/random-forest').strip('/')
            version_manifest_key = f'{prefix}/{version}/manifest.json'
            response = self.client.get_object(Bucket=self.bucket, Key=version_manifest_key)
            manifest = json.loads(response['Body'].read())
            if manifest.get('model_version') != version:
                raise ValueError('manifest candidato pertence a outra versão')
            target = self._materialize_manifest(manifest)
            return target, dict(manifest)

    def activate(self, version: str) -> dict:
        with self._lock:
            target, manifest = self.materialize_version(version)
            self.client.put_object(
                Bucket=self.bucket,
                Key=self.manifest_key,
                Body=json.dumps(manifest, ensure_ascii=False).encode(),
                ContentType='application/json',
            )
            self._active = (target, manifest)
            return dict(manifest)


def registry_from_environment() -> ModelRegistry:
    driver = os.getenv('MODEL_REGISTRY_DRIVER', 'local').lower()
    if driver == 's3':
        return S3ModelRegistry()
    root = Path(__file__).resolve().parents[2]
    artifact = os.getenv('MODEL_ARTIFACT_PATH', str(root / 'artifacts' / 'modelo_5.joblib'))
    manifest = os.getenv('MODEL_MANIFEST_PATH', str(root / 'artifacts' / 'manifest.json'))
    return LocalModelRegistry(artifact, manifest)
