#!/usr/bin/env python3
"""
Script de Otimização Massiva de Hiperparâmetros em GPU (CUDA)
Modelo: XGBoost para Manutenção Preditiva de Portas de Plataforma (PSDs)
Projeto Linha 4-Amarela

Projetado para execução direta ou via SSH em servidores com GPU NVIDIA.
"""

import argparse
import os
import json
import time
import warnings
import joblib
import numpy as np
import pandas as pd
from datetime import datetime

import xgboost as xgb
from xgboost import XGBClassifier
from sklearn.model_selection import TimeSeriesSplit
from sklearn.metrics import (
    precision_score, recall_score, f1_score,
    roc_auc_score, average_precision_score, confusion_matrix
)
import optuna

warnings.filterwarnings('ignore')
optuna.logging.set_verbosity(optuna.logging.WARNING)

def parse_args():
    parser = argparse.ArgumentParser(description="Otimização Massiva de XGBoost em GPU")
    parser.add_argument('--device', type=str, default='cuda', choices=['cuda', 'cpu'],
                        help="Dispositivo de execução: 'cuda' para GPU NVIDIA ou 'cpu'")
    parser.add_argument('--n-trials', type=int, default=150,
                        help="Número total de tentativas/modelos a testar (padrão: 150)")
    parser.add_argument('--timeout', type=int, default=3600,
                        help="Tempo limite máximo em segundos (padrão: 3600 = 1 hora)")
    parser.add_argument('--threshold', type=float, default=0.65,
                        help="Limiar de decisão operacional de Certeza Absoluta (padrão: 0.65)")
    return parser.parse_args()

def carregar_dados():
    path_treino = "data/treino_pronto_modelagem_v2.csv"
    path_teste = "data/teste_pronto_modelagem_v2.csv"
    
    if not os.path.exists(path_treino) or not os.path.exists(path_teste):
        raise FileNotFoundError("Datasets pré-processados não encontrados em data/. Execute a etapa 2 antes.")
        
    print(f"[{datetime.now().strftime('%H:%M:%S')}] Carregando datasets de modelagem...")
    df_treino = pd.read_csv(path_treino)
    df_teste = pd.read_csv(path_teste)
    
    target_col = 'target_falha_7d'
    cols_ignorar = ['id_ativo', 'data_ref', target_col]
    features = [c for c in df_treino.columns if c not in cols_ignorar]
    
    X_train = df_treino[features]
    y_train = df_treino[target_col]
    X_test = df_teste[features]
    y_test = df_teste[target_col]
    
    print(f"[{datetime.now().strftime('%H:%M:%S')}] ✅ Dados carregados: Treino ({X_train.shape[0]:,} linhas), Teste ({X_test.shape[0]:,} linhas), Features ({len(features)})")
    return X_train, y_train, X_test, y_test, features, df_teste

def main():
    args = parse_args()
    
    print("=" * 85)
    print("🚀 OTIMIZAÇÃO MASSIVA DE HIPERPARÂMETROS EM GPU (CUDA)")
    print(f"   -> Dispositivo Selecionado:  {args.device.upper()}")
    print(f"   -> Tentativas Máximas:       {args.n_trials} trials")
    print(f"   -> Tempo Limite Máximo:      {args.timeout} segundos ({args.timeout/60:.1f} minutos)")
    print(f"   -> Limiar Operacional:       p >= {args.threshold:.2f}")
    print("=" * 85)
    
    X_train, y_train, X_test, y_test, features, df_teste = carregar_dados()
    
    # Validação Cruzada Temporal (3 Janelas Sequenciais)
    tscv = TimeSeriesSplit(n_splits=3)
    
    # -------------------------------------------------------------
    # FUNÇÃO OBJETIVO: MAXIMIZAR PRECISÃO + RECALL NA GPU
    # -------------------------------------------------------------
    def objective(trial):
        params = {
            'n_estimators': trial.suggest_int('n_estimators', 150, 500, step=25),
            'max_depth': trial.suggest_int('max_depth', 3, 8),
            'learning_rate': trial.suggest_float('learning_rate', 0.012, 0.08, log=True),
            'min_child_weight': trial.suggest_int('min_child_weight', 8, 45),
            'subsample': trial.suggest_float('subsample', 0.60, 0.95),
            'colsample_bytree': trial.suggest_float('colsample_bytree', 0.60, 0.95),
            'reg_alpha': trial.suggest_float('reg_alpha', 0.05, 6.0, log=True),
            'reg_lambda': trial.suggest_float('reg_lambda', 0.5, 12.0, log=True),
            'scale_pos_weight': trial.suggest_float('scale_pos_weight', 5.0, 15.0),
            'tree_method': 'hist',
            'device': args.device,
            'random_state': 42,
            'eval_metric': 'logloss'
        }
        
        prec_list, rec_list, f1_list = [], [], []
        
        for train_idx, val_idx in tscv.split(X_train):
            X_tr, X_val = X_train.iloc[train_idx], X_train.iloc[val_idx]
            y_tr, y_val = y_train.iloc[train_idx], y_train.iloc[val_idx]
            
            clf = XGBClassifier(**params)
            clf.fit(X_tr, y_tr)
            
            probs = clf.predict_proba(X_val)[:, 1]
            preds = (probs >= args.threshold).astype(int)
            
            p = precision_score(y_val, preds, zero_division=0)
            r = recall_score(y_val, preds, zero_division=0)
            f = 2 * (p * r) / (p + r + 1e-10)
            
            prec_list.append(p)
            rec_list.append(r)
            f1_list.append(f)
            
        mean_prec = np.mean(prec_list)
        mean_rec = np.mean(rec_list)
        mean_f1 = np.mean(f1_list)
        
        trial.set_user_attr('precision', mean_prec)
        trial.set_user_attr('recall', mean_rec)
        trial.set_user_attr('f1_score', mean_f1)
        
        # Log ao vivo para o terminal SSH
        print(f"[{datetime.now().strftime('%H:%M:%S')}] Trial {trial.number+1:03d}/{args.n_trials} | Precisão: {mean_prec*100:5.2f}% | Recall: {mean_rec*100:5.2f}% | F1: {mean_f1:.4f} | (depth={params['max_depth']}, lr={params['learning_rate']:.4f}, scale_pos={params['scale_pos_weight']:.2f})")
        
        return mean_f1

    # Iniciar Estudo Optuna
    study = optuna.create_study(
        direction='maximize',
        sampler=optuna.samplers.TPESampler(seed=42)
    )
    
    start_time = time.time()
    study.optimize(
        objective,
        n_trials=args.n_trials,
        timeout=args.timeout,
        show_progress_bar=False
    )
    total_time = time.time() - start_time
    
    print("\n" + "=" * 85)
    print(f"🏁 OTIMIZAÇÃO CONCLUÍDA EM {total_time/60:.2f} MINUTOS!")
    print(f"🏆 Melhor F1-Score Médio na Validação: {study.best_value:.4f}")
    melhor_trial = study.best_trial
    print(f"   -> Precisão na Validação: {melhor_trial.user_attrs['precision']*100:.2f}%")
    print(f"   -> Recall na Validação:    {melhor_trial.user_attrs['recall']*100:.2f}%")
    print("=" * 85)
    
    # -------------------------------------------------------------
    # TREINAR O MODELO FINAL NO DATASET COMPLETO DE TREINO
    # -------------------------------------------------------------
    print(f"\n[{datetime.now().strftime('%H:%M:%S')}] Treinando o modelo campeão com todos os dados de treino na GPU...")
    best_params = study.best_params.copy()
    best_params.update({
        'tree_method': 'hist',
        'device': args.device,
        'random_state': 42,
        'eval_metric': 'logloss'
    })
    
    final_model = XGBClassifier(**best_params)
    final_model.fit(X_train, y_train)
    
    # -------------------------------------------------------------
    # AVALIAÇÃO FINAL NO TESTE CEGO DE PRODUÇÃO (2026)
    # -------------------------------------------------------------
    print(f"[{datetime.now().strftime('%H:%M:%S')}] Avaliando performance no conjunto de teste cego (2026)...")
    probs_test = final_model.predict_proba(X_test)[:, 1]
    
    roc_auc = roc_auc_score(y_test, probs_test)
    pr_auc = average_precision_score(y_test, probs_test)
    
    df_teste['prob_falha'] = probs_test
    df_teste['semana'] = pd.to_datetime(df_teste['data_ref']).dt.isocalendar().week
    
    df_sem = df_teste.groupby(['id_ativo', 'semana']).agg(
        max_prob=('prob_falha', 'max'),
        houve_falha=('target_falha_7d', 'max')
    ).reset_index()
    
    df_sem['pred'] = (df_sem['max_prob'] >= args.threshold).astype(int)
    acertos = ((df_sem['pred'] == 1) & (df_sem['houve_falha'] == 1)).sum()
    alertas_totais = (df_sem['pred'] == 1).sum()
    falhas_totais = df_sem['houve_falha'].sum()
    
    prec = acertos / alertas_totais if alertas_totais > 0 else 0
    rec = acertos / falhas_totais if falhas_totais > 0 else 0
    f1 = 2 * prec * rec / (prec + rec + 1e-10)
    
    print("\n" + "=" * 85)
    print("📊 RESULTADOS OPERACIONAIS FINAIS NO TESTE CEGO REAL (2026):")
    print("=" * 85)
    print(f"  -> ROC-AUC Score Global:      {roc_auc:.4f}")
    print(f"  -> PR-AUC (Average Precision): {pr_auc:.4f}")
    print(f"  -> Precisão Operacional:      {prec*100:.2f}% ({acertos:,} acertos em {alertas_totais:,} chamados)")
    print(f"  -> Recall Operacional:         {rec*100:.2f}% ({acertos:,} de {falhas_totais:,} falhas evitadas)")
    print(f"  -> F1-Score Operacional:       {f1:.4f}")
    print("=" * 85)
    
    # -------------------------------------------------------------
    # SALVAMENTO DOS ARTEFATOS
    # -------------------------------------------------------------
    os.makedirs("data/modelos_artefatos", exist_ok=True)
    caminho_modelo = "data/modelos_artefatos/modelo_xgboost_gpu_otimizado.joblib"
    joblib.dump(final_model, caminho_modelo)
    
    relatorio = {
        'timestamp': datetime.now().isoformat(),
        'device': args.device,
        'tempo_execucao_segundos': total_time,
        'n_trials_executados': len(study.trials),
        'melhores_hiperparametros': study.best_params,
        'metricas_teste_2026': {
            'roc_auc': float(roc_auc),
            'pr_auc': float(pr_auc),
            'precision_operacional': float(prec),
            'recall_operacional': float(rec),
            'f1_score': float(f1),
            'acertos_falha': int(acertos),
            'alertas_totais': int(alertas_totais),
            'falhas_totais': int(falhas_totais)
        }
    }
    
    caminho_relatorio = "data/modelos_artefatos/relatorio_otimizacao_gpu.json"
    with open(caminho_relatorio, 'w', encoding='utf-8') as f:
        json.dump(relatorio, f, indent=2, ensure_ascii=False)
        
    print(f"\n✅ Modelo Campeão salvo em:    {caminho_modelo}")
    print(f"✅ Relatório JSON salvo em:      {caminho_relatorio}")
    print("=" * 85)

if __name__ == "__main__":
    main()
