"""Prepara a versão final do dataset para treino: encoding de `estacao`.

Lê dataset_modelo_porta_dia.csv e gera dataset_modelo_final.csv com:
- `data` e `porta` mantidos apenas como identificadores (não entram como
  feature de entrada do modelo: `porta` é praticamente um ID de porta,
  usá-lo direto arrisca o modelo decorar o equipamento em vez de generalizar
  pelo histórico dele, que já está nas features n_corretivas_90d etc.).
- `estacao` (11 categorias) convertida em one-hot (`estacao_BUT`, `estacao_PIN`, ...).
- Demais colunas numéricas mantidas como estão — sem normalização/escala,
  já que o modelo alvo é baseado em árvore (Random Forest / XGBoost /
  LightGBM), que não exige features na mesma escala.
"""

from pathlib import Path

import pandas as pd

BASE = Path(__file__).resolve().parent

df = pd.read_csv(BASE / "dataset_modelo_porta_dia.csv", parse_dates=["data"])

estacao_dummies = pd.get_dummies(df["estacao"], prefix="estacao")

df_final = pd.concat(
    [df[["data", "estacao", "porta", "falha_7d"]], estacao_dummies, df.drop(columns=["data", "estacao", "porta", "falha_7d"])],
    axis=1,
)

saida = BASE / "dataset_modelo_final.csv"
df_final.to_csv(saida, index=False)

print(f"[Dataset final para treino] {len(df_final):,} linhas x {df_final.shape[1]} colunas")
print("Colunas:", list(df_final.columns))
print("Salvo em:", saida)
