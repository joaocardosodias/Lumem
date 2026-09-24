"""
Funcoes de treino e avaliacao da RNN, compartilhadas pelas etapas da otimizacao.

A logica do baseline e a mesma de `models/modelo-rnn/modelo_rnn.ipynb`, com
os hiperparametros expostos em um dicionario `cfg` para a busca das etapas
seguintes. Com `CFG_BASELINE` o resultado reproduz o notebook original.
"""
import os

# a instalacao do torch pode usar uma OpenMP diferente da do numpy/scikit-learn;
# sem isso o treino pode travar (nao da erro, so nunca avanca)
os.environ.setdefault('KMP_DUPLICATE_LIB_OK', 'TRUE')

from pathlib import Path

import numpy as np
import pandas as pd
import torch
import torch.nn as nn
from sklearn.metrics import (average_precision_score, confusion_matrix, f1_score,
                             precision_score, recall_score, roc_auc_score)
from sklearn.model_selection import TimeSeriesSplit

SEQ_DIR = Path(__file__).resolve().parents[1] / 'modelo-rnn'

# mesmo corte temporal do Modelo 5 e da RNN original
FIM_TREINO, INI_TESTE, INI_VALID = (pd.Timestamp('2025-04-23'), pd.Timestamp('2025-05-01'),
                                    pd.Timestamp('2025-01-31'))

# hiperparametros do notebook original (modelo_rnn.ipynb)
CFG_BASELINE = {
    'cell': 'lstm', 'hidden': 32, 'layers': 1, 'dropout': 0.0,
    'lr': 1e-3, 'batch': 1024, 'weight_decay': 0.0, 'pos_weight_mult': 1.0,
    'max_epocas': 15, 'paciencia': 3,
}

device = torch.device('cpu')


def set_threads(n):
    torch.set_num_threads(n)


def seed_tudo(seed):
    torch.manual_seed(seed)
    np.random.seed(seed)


def carrega_dados():
    """Sequencias [n, 30, 12], alvo, indice (id_porta, Data) e mascaras da particao temporal."""
    d = np.load(SEQ_DIR / 'sequencias.npz')
    X, y = d['X'], d['y']
    idx = pd.read_parquet(SEQ_DIR / 'sequencias_index.parquet')
    data = idx['Data']
    m_treino = (data <= FIM_TREINO).values
    m_teste = (data >= INI_TESTE).values
    m_fit = (m_treino & (data <= INI_VALID - pd.Timedelta(days=8))).values
    m_val = (m_treino & (data >= INI_VALID)).values
    return X, y, idx, {'treino': m_treino, 'teste': m_teste, 'fit': m_fit, 'val': m_val}


def folds_temporais(idx, treino_mask, n_folds=2, folga_dias=7):
    """Janelas expansivas dentro do treino, no mesmo espirito do TimeSeriesSplit usado no
    Modelo 5 (RandomizedSearchCV/GridSearchCV). Cada fold usa um corte de data mais recente
    que o anterior, com uma folga entre fit e val para evitar vazamento (mesma logica da
    secao 3 do modelo_5.ipynb). O teste nunca entra aqui.
    """
    datas = np.sort(idx.loc[treino_mask, 'Data'].unique())
    tscv = TimeSeriesSplit(n_splits=n_folds)
    data_col = idx['Data'].values
    folds = []
    for fit_idx, val_idx in tscv.split(datas):
        fim_fit = pd.Timestamp(datas[fit_idx[-1]])
        ini_val = pd.Timestamp(datas[val_idx[0]])
        fim_val = pd.Timestamp(datas[val_idx[-1]])
        m_fit = treino_mask & (data_col <= fim_fit - pd.Timedelta(days=folga_dias))
        m_val = treino_mask & (data_col >= ini_val) & (data_col <= fim_val)
        folds.append((m_fit, m_val))
    return folds


def estatisticas_norm(X_ref):
    """Media e desvio por feature, calculados so no conjunto de referencia (fit)."""
    mu = X_ref.mean(axis=(0, 1), keepdims=True)
    sd = X_ref.std(axis=(0, 1), keepdims=True) + 1e-6
    return mu, sd


def estatisticas_norm_extra(E_ref):
    """Mesma ideia de `estatisticas_norm`, para as features extra (2D: [n, n_extra]) da Etapa 2."""
    mu = E_ref.mean(axis=0, keepdims=True)
    sd = E_ref.std(axis=0, keepdims=True) + 1e-6
    return mu, sd


class RNNFalha(nn.Module):
    """Com `n_extra=0` (padrao) e a mesma rede das Etapas 0 e 1: so a sequencia.
    Com `n_extra>0` (Etapa 2), features extra (historico ZM/estacao/via, uma por
    observacao, nao por dia) sao concatenadas ao estado escondido final antes da
    camada de saida -- a sequencia continua sendo processada do mesmo jeito.
    """
    def __init__(self, n_feat, cfg, n_extra=0):
        super().__init__()
        rec = nn.LSTM if cfg['cell'] == 'lstm' else nn.GRU
        drop_rec = cfg['dropout'] if cfg['layers'] > 1 else 0.0
        self.rec = rec(n_feat, cfg['hidden'], num_layers=cfg['layers'],
                       dropout=drop_rec, batch_first=True)
        self.drop = nn.Dropout(cfg['dropout'])
        self.n_extra = n_extra
        self.saida = nn.Linear(cfg['hidden'] + n_extra, 1)
        self.lstm = cfg['cell'] == 'lstm'

    def forward(self, x, extra=None):
        _, h = self.rec(x)
        h = h[0] if self.lstm else h       # LSTM devolve (h, c); GRU devolve so h
        z = self.drop(h[-1])               # h[-1]: ultima camada, ultimo dia
        if self.n_extra:
            z = torch.cat([z, extra], dim=-1)
        return self.saida(z).squeeze(-1)


def _tensor(a):
    return torch.tensor(a, dtype=torch.float32)


def _monta(cfg, ytr, n_feat, n_extra=0):
    modelo = RNNFalha(n_feat, cfg, n_extra=n_extra).to(device)
    pos_weight = torch.tensor([cfg['pos_weight_mult'] * float((ytr == 0).sum()) / max(float((ytr == 1).sum()), 1.0)],
                              dtype=torch.float32, device=device)
    perda_fn = nn.BCEWithLogitsLoss(pos_weight=pos_weight)
    otim = torch.optim.Adam(modelo.parameters(), lr=cfg['lr'], weight_decay=cfg['weight_decay'])
    return modelo, perda_fn, otim


def _epoca(modelo, perda_fn, otim, Xt, yt, lote, Et=None):
    modelo.train()
    n = len(Xt)
    perm = torch.randperm(n)
    perda_total = 0.0
    for i in range(0, n, lote):
        b = perm[i:i + lote]
        xb, yb = Xt[b].to(device), yt[b].to(device)
        eb = Et[b].to(device) if Et is not None else None
        otim.zero_grad()
        perda = perda_fn(modelo(xb, extra=eb), yb)
        perda.backward()
        otim.step()
        perda_total += perda.item() * len(b)
    return perda_total / n


def prediz(modelo, Xa, norma, Ea=None, norma_extra=None):
    mu, sd = norma
    xt = _tensor((Xa - mu) / sd).to(device)
    et = None
    if Ea is not None:
        mu_e, sd_e = norma_extra
        et = _tensor((Ea - mu_e) / sd_e).to(device)
    modelo.eval()
    with torch.no_grad():
        return torch.sigmoid(modelo(xt, extra=et)).cpu().numpy()


def treina(Xtr, ytr, Xval, yval, norma, cfg, verboso=True, Etr=None, Eval=None, norma_extra=None):
    """Treina com early stopping no PR-AUC de validacao. Devolve (melhor_epoca, melhor_prauc, historico).
    `Etr`/`Eval`/`norma_extra` sao opcionais (Etapa 2): features extra por observacao.
    """
    mu, sd = norma
    n_extra = Etr.shape[-1] if Etr is not None else 0
    modelo, perda_fn, otim = _monta(cfg, ytr, Xtr.shape[-1], n_extra=n_extra)
    Xtr_t, ytr_t = _tensor((Xtr - mu) / sd), _tensor(ytr)
    Etr_t = None
    if Etr is not None:
        mu_e, sd_e = norma_extra
        Etr_t = _tensor((Etr - mu_e) / sd_e)

    melhor_prauc, melhor_epoca, sem_melhora, hist = -1, 0, 0, []
    for epoca in range(1, cfg['max_epocas'] + 1):
        perda = _epoca(modelo, perda_fn, otim, Xtr_t, ytr_t, cfg['batch'], Et=Etr_t)
        prauc = average_precision_score(yval, prediz(modelo, Xval, norma, Ea=Eval, norma_extra=norma_extra))
        hist.append({'epoca': epoca, 'perda': perda, 'pr_auc_val': prauc})
        if verboso:
            print(f'epoca {epoca:2d} | perda={perda:.4f} | PR-AUC val={prauc:.4f}')
        if prauc > melhor_prauc:
            melhor_prauc, melhor_epoca, sem_melhora = prauc, epoca, 0
        else:
            sem_melhora += 1
            if sem_melhora >= cfg['paciencia']:
                if verboso:
                    print(f'parando cedo na epoca {epoca} (melhor foi {melhor_epoca})')
                break
    return melhor_epoca, melhor_prauc, hist


def treina_fixo(Xtr, ytr, norma, cfg, n_epocas, Etr=None, norma_extra=None):
    """Treina por um numero fixo de epocas, sem early stopping (usado no refit final)."""
    mu, sd = norma
    n_extra = Etr.shape[-1] if Etr is not None else 0
    modelo, perda_fn, otim = _monta(cfg, ytr, Xtr.shape[-1], n_extra=n_extra)
    Xtr_t, ytr_t = _tensor((Xtr - mu) / sd), _tensor(ytr)
    Etr_t = None
    if Etr is not None:
        mu_e, sd_e = norma_extra
        Etr_t = _tensor((Etr - mu_e) / sd_e)
    for _ in range(n_epocas):
        _epoca(modelo, perda_fn, otim, Xtr_t, ytr_t, cfg['batch'], Et=Etr_t)
    return modelo


def melhor_limiar(y, prob):
    grid = np.unique(np.quantile(prob, np.linspace(0.50, 0.999, 200)))
    return max(((f1_score(y, (prob >= t).astype(int), zero_division=0), t) for t in grid))[1]


def avalia(nome, yte, pred, prob):
    tn, fp, fn, tp = confusion_matrix(yte, pred).ravel()
    return {'modelo': nome, 'precisao': precision_score(yte, pred, zero_division=0),
            'recall': recall_score(yte, pred, zero_division=0), 'f1': f1_score(yte, pred, zero_division=0),
            'pr_auc': average_precision_score(yte, prob), 'roc_auc': roc_auc_score(yte, prob),
            'alertas': int(pred.sum()), 'tp': int(tp), 'fp': int(fp), 'fn': int(fn)}
