import argparse
import hashlib
import json
import os
import re
import shutil
import time
from datetime import datetime, timezone
from pathlib import Path

import joblib
import numpy as np
import pandas as pd
from sklearn.ensemble import RandomForestClassifier
from sklearn.metrics import (
    average_precision_score,
    confusion_matrix,
    f1_score,
    precision_score,
    recall_score,
    roc_auc_score,
)


def load_local_environment():
    env_path = Path(__file__).resolve().parent.parent / 'backend' / '.env'
    if not env_path.is_file():
        return
    for raw_line in env_path.read_text(encoding='utf-8').splitlines():
        line = raw_line.strip()
        if not line or line.startswith('#') or '=' not in line:
            continue
        key, value = line.split('=', 1)
        os.environ.setdefault(key.strip(), value.strip().strip('"').strip("'"))


load_local_environment()

MODULE_DIR = Path(__file__).resolve().parent
if MODULE_DIR.parent.name == 'src':
    DEFAULT_PROJECT_ROOT = MODULE_DIR.parents[1]
    DEFAULT_PANEL = DEFAULT_PROJECT_ROOT / 'models' / 'modelo-5' / 'painel.parquet'
    DEFAULT_MAINTENANCE = DEFAULT_PROJECT_ROOT / 'data' / 'Corretiva Emergencial' / 'processados' / 'ZM01_limpa.xlsx'
else:
    DEFAULT_PANEL = MODULE_DIR / 'training-data' / 'painel.parquet'
    DEFAULT_MAINTENANCE = MODULE_DIR / 'training-data' / 'ZM01_limpa.xlsx'
ARTIFACTS = MODULE_DIR / 'artifacts'

t0 = time.time()


def parse_args():
    parser = argparse.ArgumentParser(description='Treina, avalia e registra o modelo de falhas PSD.')
    parser.add_argument('--version', default=os.getenv('MODEL_VERSION', 'v1.0.0'))
    parser.add_argument('--publish', action='store_true', help='Publica os arquivos no registry S3 configurado.')
    parser.add_argument('--activate-s3', action='store_true', help='Promove a versão publicada no ponteiro ativo do S3.')
    parser.add_argument('--activate-local', action='store_true', help='Atualiza o ponteiro local depois de validar o candidato.')
    parser.add_argument('--evaluate-existing', action='store_true', help='Regenera manifest e curva sem alterar o modelo treinado.')
    return parser.parse_args()


ARGS = parse_args()
VERSION_DIR = ARTIFACTS / 'models' / ARGS.version
SAIDA = VERSION_DIR / 'model.joblib'


def log(msg):
    print(f'[{time.time() - t0:5.0f}s] {msg}', flush=True)


def materialize_training_input(path_variable, default_path, key_variable, filename):
    source = Path(os.getenv(path_variable, default_path))
    if source.is_file():
        return source

    object_key = os.getenv(key_variable)
    if not object_key:
        raise FileNotFoundError(
            f'{source} não existe; informe {path_variable} ou {key_variable} para o snapshot de treinamento'
        )

    import boto3

    target_dir = Path(os.getenv('MODEL_TRAINING_CACHE_DIR', '/tmp/g04-training-data'))
    target_dir.mkdir(parents=True, exist_ok=True)
    target = target_dir / filename
    bucket = os.getenv('MODEL_TRAINING_BUCKET') or os.getenv('MODEL_S3_BUCKET') or os.environ['S3_BUCKET']
    client = boto3.client(
        's3',
        endpoint_url=os.getenv('MODEL_S3_ENDPOINT') or os.getenv('S3_ENDPOINT'),
        region_name=os.getenv('MODEL_S3_REGION') or os.getenv('S3_REGION', 'us-west-2'),
        aws_access_key_id=os.getenv('MODEL_S3_ACCESS_KEY_ID') or os.getenv('S3_ACCESS_KEY_ID'),
        aws_secret_access_key=os.getenv('MODEL_S3_SECRET_ACCESS_KEY') or os.getenv('S3_SECRET_ACCESS_KEY'),
    )
    temporary = target.with_suffix(target.suffix + '.download')
    client.download_file(bucket, object_key, str(temporary))
    temporary.replace(target)
    return target


PAINEL = materialize_training_input(
    'MODEL_PANEL_PATH', DEFAULT_PANEL, 'MODEL_PANEL_S3_KEY', 'painel.parquet'
)
ZM01_PATH = materialize_training_input(
    'MODEL_MAINTENANCE_PATH', DEFAULT_MAINTENANCE, 'MODEL_MAINTENANCE_S3_KEY', 'ZM01_limpa.xlsx'
)


log('lendo painel...')
painel = pd.read_parquet(PAINEL)
painel['Data'] = pd.to_datetime(painel['Data']).astype('datetime64[ns]')

PAT_LOC = re.compile(r'^SP04-6-PSD-([A-Z]{3})(?:-PLT(\d)|-P(\d)S\d)?(?:-PP(\d{2})(PDM|PFX|PSE))?')


def carrega_ordens(caminho, aba):
    df = pd.read_excel(caminho, aba)
    m = df['local_de_instalacao'].apply(lambda s: PAT_LOC.match(s) if isinstance(s, str) else None)
    df['est'] = [x.group(1) if x else None for x in m]
    df['via'] = [(x.group(2) or x.group(3)) if x else None for x in m]
    df['porta'] = [int(x.group(4)) if x and x.group(4) else None for x in m]
    df = df.dropna(subset=['est', 'via', 'porta', 'inicio_avaria_datetime']).copy()
    df['porta'] = df['porta'].astype(int)
    df['id_porta'] = df['est'] + '_' + df['via'] + '_' + df['porta'].astype(str).str.zfill(2)
    df['Data'] = pd.to_datetime(df['inicio_avaria_datetime']).dt.normalize().astype('datetime64[ns]')
    return df


log('recarregando zm01 pras features de ciclo...')
zm01 = carrega_ordens(ZM01_PATH, 'ZM01_limpa')

ev = zm01[['id_porta', 'Data']].dropna().drop_duplicates().sort_values(['id_porta', 'Data']).reset_index(drop=True)
ev['ordem'] = ev.groupby('id_porta').cumcount() + 1
ev['diff'] = ev.groupby('id_porta')['Data'].diff().dt.days
ev['intervalo_medio_ate_aqui'] = ev.groupby('id_porta')['diff'] \
    .apply(lambda s: s.expanding().mean()).reset_index(level=0, drop=True)

p = painel[['id_porta', 'Data']].copy()
p['_idx'] = painel.index
m = pd.merge_asof(p.sort_values('Data'),
                   ev[['id_porta', 'Data', 'ordem', 'intervalo_medio_ate_aqui']].sort_values('Data'),
                   on='Data', by='id_porta', direction='backward', allow_exact_matches=False)
painel['n_corretivas_historico_total'] = pd.Series(m['ordem'].values, index=m['_idx'].values) \
    .reindex(painel.index).fillna(0)
_interv = pd.Series(m['intervalo_medio_ate_aqui'].values, index=m['_idx'].values).reindex(painel.index)
painel['intervalo_medio_zm01'] = _interv.fillna(_interv.median())
painel['razao_atraso_zm01'] = painel['dias_desde_zm01'] / painel['intervalo_medio_zm01']
painel['dias_desde_ultima_corretiva'] = painel[['dias_desde_zm01', 'dias_desde_zm02']].min(axis=1)

painel['rampa_7d'] = painel['n_total_7d'] - (painel['n_total_14d'] - painel['n_total_7d'])
painel['razao_7d_30d'] = painel['n_total_7d'] / (painel['n_total_30d'] / 4 + 1)
painel['frac_tecnicos_30d'] = painel['n_tecnicos_30d'] / (painel['n_total_30d'] + 1)
painel['est'] = painel['est'].astype('category')
painel['est_cod'] = painel['est'].cat.codes
painel['via_cod'] = painel['via'].astype(int)
EST_CATEGORIAS = painel['est'].cat.categories.tolist()

EXCLUI = {'id_porta', 'Data', 'est', 'via', 'id_plat', 'y', 'n_falhas_7d_futuro', 'cobertura_30d', 'mes'}
FEATS = [c for c in painel.columns if c not in EXCLUI]
log(f'{len(FEATS)} features candidatas')

FIM_TREINO, INI_TESTE, INI_VALID = (pd.Timestamp('2025-04-23'), pd.Timestamp('2025-05-01'),
                                     pd.Timestamp('2025-01-31'))
treino = painel[painel['Data'] <= FIM_TREINO]
teste = painel[painel['Data'] >= INI_TESTE]
tr_fit = treino[treino['Data'] <= INI_VALID - pd.Timedelta(days=8)]
tr_val = treino[treino['Data'] >= INI_VALID]
log(f'treino: {len(treino):,} | teste: {len(teste):,}')


def melhor_limiar(y, prob):
    grid = np.unique(np.quantile(prob, np.linspace(0.50, 0.999, 200)))
    return max(((f1_score(y, (prob >= t).astype(int), zero_division=0), t) for t in grid))[1]


def treina_avalia(feats):
    mod = RandomForestClassifier(n_estimators=400, max_depth=14, min_samples_leaf=40,
                                  class_weight='balanced', n_jobs=-1, random_state=42)
    mod.fit(tr_fit[feats], tr_fit['y'])
    lim = melhor_limiar(tr_val['y'], mod.predict_proba(tr_val[feats])[:, 1])
    mod.fit(treino[feats], treino['y'])
    return mod, lim


def metricas_classificacao(y_true, probabilities, threshold):
    predicted = (probabilities >= threshold).astype(int)
    tn, fp, fn, tp = confusion_matrix(y_true, predicted, labels=[0, 1]).ravel()
    alerts = int(predicted.sum())
    return {
        'threshold': round(float(threshold), 4),
        'true_positives': int(tp),
        'false_positives': int(fp),
        'false_negatives': int(fn),
        'true_negatives': int(tn),
        'precision': float(precision_score(y_true, predicted, zero_division=0)),
        'recall': float(recall_score(y_true, predicted, zero_division=0)),
        'f1_score': float(f1_score(y_true, predicted, zero_division=0)),
        'alert_count': alerts,
        'alert_rate': float(alerts / len(predicted)) if len(predicted) else 0.0,
    }


def hash_dataframe(frame):
    digest = hashlib.sha256()
    digest.update(pd.util.hash_pandas_object(frame, index=True).values.tobytes())
    digest.update('|'.join(frame.columns).encode())
    return digest.hexdigest()


def hash_file(path):
    digest = hashlib.sha256()
    with Path(path).open('rb') as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b''):
            digest.update(chunk)
    return digest.hexdigest()


def reference_distribution(frame):
    result = {}
    for column in frame.columns:
        values = pd.to_numeric(frame[column], errors='coerce').to_numpy(dtype=float)
        finite = values[np.isfinite(values)]
        if finite.size == 0:
            continue
        internal = np.unique(np.quantile(finite, np.linspace(0.1, 0.9, 9)))
        bins = np.concatenate(([-1e308], internal, [1e308]))
        counts, _ = np.histogram(finite, bins=bins)
        result[column] = {
            'bins': [float(value) for value in bins],
            'proportions': [float(value) for value in counts / max(counts.sum(), 1)],
            'missing_ratio': float(1 - finite.size / max(values.size, 1)),
            'min': float(finite.min()),
            'max': float(finite.max()),
            'records': int(values.size),
        }
    return result


def publish_to_s3(manifest, metrics_path):
    import boto3
    from botocore.exceptions import ClientError

    bucket = os.getenv('MODEL_S3_BUCKET') or os.environ['S3_BUCKET']
    prefix = os.getenv('MODEL_S3_PREFIX', 'models/random-forest').strip('/')
    version_prefix = f"{prefix}/{manifest['model_version']}"
    client = boto3.client(
        's3',
        endpoint_url=os.getenv('MODEL_S3_ENDPOINT') or os.getenv('S3_ENDPOINT'),
        region_name=os.getenv('MODEL_S3_REGION') or os.getenv('S3_REGION', 'us-west-2'),
        aws_access_key_id=os.getenv('MODEL_S3_ACCESS_KEY_ID') or os.getenv('S3_ACCESS_KEY_ID'),
        aws_secret_access_key=os.getenv('MODEL_S3_SECRET_ACCESS_KEY') or os.getenv('S3_SECRET_ACCESS_KEY'),
    )
    artifact_key = f'{version_prefix}/model.joblib'
    manifest_key = f'{version_prefix}/manifest.json'
    metrics_key = f'{version_prefix}/threshold-metrics.json'
    published_manifest = dict(manifest)
    published_manifest['artifact_uri'] = f's3://{bucket}/{artifact_key}'
    manifest_payload = json.dumps(published_manifest, indent=2, ensure_ascii=False).encode()

    def upload_immutable(path, key, content_type):
        checksum = hash_file(path)
        try:
            existing = client.head_object(Bucket=bucket, Key=key)
            remote_checksum = (existing.get('Metadata') or {}).get('sha256')
            if remote_checksum == checksum:
                return
            raise FileExistsError(f'o objeto imutável s3://{bucket}/{key} já existe com outro conteúdo')
        except ClientError as error:
            status = error.response.get('ResponseMetadata', {}).get('HTTPStatusCode')
            code = str(error.response.get('Error', {}).get('Code', ''))
            if status != 404 and code not in {'404', 'NoSuchKey', 'NotFound'}:
                raise
        extra = {'Metadata': {'sha256': checksum}}
        if content_type:
            extra['ContentType'] = content_type
        client.upload_file(str(path), bucket, key, ExtraArgs=extra)

    def upload_immutable_bytes(payload, key, content_type):
        checksum = hashlib.sha256(payload).hexdigest()
        try:
            existing = client.head_object(Bucket=bucket, Key=key)
            remote_checksum = (existing.get('Metadata') or {}).get('sha256')
            if remote_checksum == checksum:
                return
            raise FileExistsError(f'o objeto imutável s3://{bucket}/{key} já existe com outro conteúdo')
        except ClientError as error:
            status = error.response.get('ResponseMetadata', {}).get('HTTPStatusCode')
            code = str(error.response.get('Error', {}).get('Code', ''))
            if status != 404 and code not in {'404', 'NoSuchKey', 'NotFound'}:
                raise
        client.put_object(
            Bucket=bucket,
            Key=key,
            Body=payload,
            ContentType=content_type,
            Metadata={'sha256': checksum},
        )

    upload_immutable(SAIDA, artifact_key, 'application/octet-stream')
    upload_immutable(metrics_path, metrics_key, 'application/json')
    upload_immutable_bytes(manifest_payload, manifest_key, 'application/json')
    if ARGS.activate_s3:
        active_key = os.getenv('MODEL_MANIFEST_KEY', f'{prefix}/active/manifest.json')
        client.put_object(
            Bucket=bucket,
            Key=active_key,
            Body=json.dumps(published_manifest, ensure_ascii=False).encode(),
            ContentType='application/json',
        )
        log(f'versão {published_manifest["model_version"]} promovida no ponteiro s3://{bucket}/{active_key}')
    log(f'artefato e manifest publicados em s3://{bucket}/{version_prefix}/')


VERSION_DIR.mkdir(parents=True, exist_ok=True)
if ARGS.evaluate_existing:
    source_artifact = ARTIFACTS / 'modelo_5.joblib'
    bundle = joblib.load(source_artifact)
    mod_final = bundle['model']
    mantidas = list(bundle['features'])
    lim_final = float(bundle.get('threshold', 0.5))
    shutil.copyfile(source_artifact, SAIDA)
    log(f'avaliando artefato existente com {len(mantidas)} features e limiar {lim_final:.4f}')
else:
    import shap

    log('treinando com todas as features...')
    mod_full, _ = treina_avalia(FEATS)

    log('calculando SHAP (TreeExplainer, amostra de 4000 linhas do teste)...')
    amostra = teste.sample(n=4000, random_state=42)
    explainer = shap.TreeExplainer(mod_full)
    sv = explainer.shap_values(amostra[FEATS])
    if isinstance(sv, list):
        sv = sv[1]
    if sv.ndim == 3:
        sv = sv[:, :, 1]

    imp = pd.DataFrame({'feature': FEATS, 'shap_abs_medio': np.abs(sv).mean(axis=0)}) \
        .sort_values('shap_abs_medio', ascending=False).reset_index(drop=True)
    limiar_corte = imp['shap_abs_medio'].iloc[0] * 0.01
    mantidas = imp[imp['shap_abs_medio'] >= limiar_corte]['feature'].tolist()
    log(f'mantidas: {len(mantidas)} de {len(FEATS)} (corte SHAP em {limiar_corte:.5f})')

    log('treinando modelo final com as features selecionadas...')
    mod_final, lim_final = treina_avalia(mantidas)
    log(f'limiar final: {lim_final:.4f}')
    joblib.dump({'model': mod_final, 'features': mantidas, 'threshold': float(lim_final),
                 'est_categorias': EST_CATEGORIAS, 'version': ARGS.version}, SAIDA)

test_probabilities = mod_final.predict_proba(teste[mantidas])[:, 1]
train_probabilities = mod_final.predict_proba(treino[mantidas])[:, 1]
threshold_metrics = [
    metricas_classificacao(teste['y'].to_numpy(), test_probabilities, threshold)
    for threshold in np.round(np.arange(0.05, 0.951, 0.01), 2)
]
recommended = min(threshold_metrics, key=lambda item: abs(item['threshold'] - lim_final))
model_sha256 = hash_file(SAIDA)
dataset_columns = ['Data', 'y', *mantidas]
dataset_sha256 = hash_dataframe(teste[dataset_columns])
feature_importance = []
if hasattr(mod_final, 'feature_importances_'):
    for feature, importance in sorted(
        zip(mantidas, mod_final.feature_importances_), key=lambda item: item[1], reverse=True
    ):
        feature_importance.append({'feature': feature, 'importance': float(importance)})

manifest = {
    'schema_version': 1,
    'model_version': ARGS.version,
    'model_sha256': model_sha256,
    'dataset_sha256': dataset_sha256,
    'artifact_uri': f'local://artifacts/models/{ARGS.version}/model.joblib',
    'algorithm': 'RandomForestClassifier',
    'hyperparameters': mod_final.get_params(),
    'features': mantidas,
    'target': 'falha_nos_proximos_7_dias',
    'train_end': str(FIM_TREINO.date()),
    'test_start': str(teste['Data'].min().date()),
    'test_end': str(teste['Data'].max().date()),
    'test_rows': int(len(teste)),
    'positive_rows': int(teste['y'].sum()),
    'generated_at': datetime.now(timezone.utc).isoformat(),
    'recommended_threshold': float(recommended['threshold']),
    'train_metrics': {
        'roc_auc': float(roc_auc_score(treino['y'], train_probabilities)),
        'pr_auc': float(average_precision_score(treino['y'], train_probabilities)),
        **metricas_classificacao(treino['y'].to_numpy(), train_probabilities, lim_final),
    },
    'test_metrics': {
        'roc_auc': float(roc_auc_score(teste['y'], test_probabilities)),
        'pr_auc': float(average_precision_score(teste['y'], test_probabilities)),
        **recommended,
    },
    'threshold_metrics': threshold_metrics,
    'feature_importance': feature_importance,
    'reference_distribution': reference_distribution(treino[mantidas]),
}

metrics_path = VERSION_DIR / 'threshold-metrics.json'
metrics_path.write_text(json.dumps(threshold_metrics, indent=2), encoding='utf-8')
manifest_path = VERSION_DIR / 'manifest.json'
manifest_path.write_text(
    json.dumps(manifest, indent=2, ensure_ascii=False), encoding='utf-8'
)

if ARGS.activate_local:
    active_artifact = ARTIFACTS / 'modelo_5.joblib'
    active_manifest = ARTIFACTS / 'manifest.json'
    shutil.copyfile(SAIDA, active_artifact)
    active_manifest_data = dict(manifest)
    active_manifest_data['artifact_uri'] = 'local://artifacts/modelo_5.joblib'
    active_manifest.write_text(json.dumps(active_manifest_data, indent=2, ensure_ascii=False), encoding='utf-8')
    log(f'versão {ARGS.version} ativada no registry local')

if ARGS.publish or ARGS.activate_s3 or os.getenv('MODEL_REGISTRY_DRIVER', 'local').lower() == 's3':
    publish_to_s3(manifest, metrics_path)
log(f'bundle salvo em {SAIDA}')
log(f"curva salva com {len(threshold_metrics)} thresholds; SHA-256 {model_sha256[:12]}…")
