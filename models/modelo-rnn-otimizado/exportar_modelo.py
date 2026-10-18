"""
Etapa 4: exporta o artefato final da RNN otimizada + extras (Etapa 2), no
mesmo espirito de `src/modelo/exportar_modelo.py` para o Random Forest --
treina no treino completo com uma semente fixa e salva tudo que e preciso
para servir o modelo depois (pesos, config, normalizacao, limiar e a lista
de features extra, na ordem que o modelo espera).

Isto NAO coloca o modelo em producao -- so deixa o artefato pronto, caso a
equipe decida integrar a RNN na API futuramente (ver README de `src/modelo/`
para o que isso exigiria).

Uso (a partir de models/modelo-rnn-otimizado/):
    python exportar_modelo.py
"""
import json
from pathlib import Path

import numpy as np
import torch

import rnn_utils as R

SEMENTE = 42
SAIDA = Path(__file__).resolve().parent / 'artefatos'
SAIDA.mkdir(exist_ok=True)

R.set_threads(1)
R.seed_tudo(SEMENTE)

with open('etapa2_resultado.json', encoding='utf-8') as f:
    etapa2 = json.load(f)

X, y, idx, m = R.carrega_dados()
norma = R.estatisticas_norm(X[m['fit']])
E = np.load('extras.npz')['E']
norma_extra = R.estatisticas_norm_extra(E[m['fit']])
cols_extra = Path('extras_features.txt').read_text().splitlines()

modelo = R.treina_fixo(X[m['treino']], y[m['treino']], norma, etapa2['cfg'], etapa2['melhor_epoca'],
                        Etr=E[m['treino']], norma_extra=norma_extra)

artefato = {
    'state_dict': modelo.state_dict(),
    'cfg': etapa2['cfg'],
    'n_extra': E.shape[-1],
    'colunas_extra': cols_extra,
    'norma_sequencia': {'mu': norma[0], 'sd': norma[1]},
    'norma_extra': {'mu': norma_extra[0], 'sd': norma_extra[1]},
    'limiar': etapa2['limiar'],
    'semente': SEMENTE,
    'seq_len': X.shape[1],
    'colunas_sequencia': Path('../modelo-rnn/sequencias_features.txt').read_text().splitlines(),
}
torch.save(artefato, SAIDA / 'rnn_otimizada_extras.pt')
print(f'salvo em {SAIDA / "rnn_otimizada_extras.pt"}')
