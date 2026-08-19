#!/usr/bin/env python3
"""
Script de Otimização Massiva de Hiperparâmetros em GPU (CUDA)
Modelo: XGBoost para Manutenção Preditiva de Portas de Plataforma (PSDs)
Projeto Linha 4-Amarela

Rastreia e exporta 3 Campeões:
1. Campeão Geral (Melhor F1-Score / Equilíbrio)
2. Campeão de Precisão (Maior Precisão Operacional)
3. Campeão de Recall (Maior Cobertura de Falhas)
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
    path_mestra = "data/tabela_mestra_bruta.csv"
    
    if not os.path.exists(path_treino) or not os.path.exists(path_teste):
        if not os.path.exists(path_mestra):
            raise FileNotFoundError(f"Nem os datasets prontos nem {path_mestra} foram encontrados. Certifique-se de ter os dados na pasta data/.")
            
        print(f"[{datetime.now().strftime('%H:%M:%S')}] Gerando datasets de modelagem v2 a partir da tabela mestra...")
        df = pd.read_csv(path_mestra)
        df['data_ref'] = pd.to_datetime(df['data_ref'])
        
        # Features 2.0
        df['aceleracao_alarmes_7d'] = df['qtd_alarmes_7d'] - (df['qtd_alarmes_14d'] - df['qtd_alarmes_7d'])
        df['tempo_medio_por_alarme_7d'] = df['tempo_parada_acumulado_7d'] / (df['qtd_alarmes_7d'] + 1)
        df['proporcao_alarmes_graves_7d'] = df['qtd_anomalias_30s_14d'] / (df['qtd_alarmes_14d'] + 1)
        df['faixa_risco_zm01'] = np.where((df['dias_desde_ultima_zm01'] >= 30) & (df['dias_desde_ultima_zm01'] <= 90), 1, 0)
        
        data_corte = pd.to_datetime('2026-01-01')
        df_treino = df[df['data_ref'] < data_corte].copy()
        df_teste = df[df['data_ref'] >= data_corte].copy()
        
        def aplicar_encodings(df_in):
            df_out = df_in.copy()
            df_out['plataforma_bin'] = df_out['plataforma'].map({'P1': 0, 'P2': 1}).fillna(0).astype(int)
            df_out = pd.get_dummies(df_out, columns=['estacao'], prefix='estacao', drop_first=False)
            return df_out

        df_treino_enc = aplicar_encodings(df_treino)
        df_teste_enc = aplicar_encodings(df_teste)
        df_treino_enc, df_teste_enc = df_treino_enc.align(df_teste_enc, join='left', axis=1, fill_value=0)
        
        cols_para_escalonar = [
            'idade_ativo_anos', 'porta',
            'dias_desde_ultima_zm01', 'dias_desde_ultima_zm02', 'dias_desde_ultima_zm03',
            'qtd_alarmes_1d', 'qtd_alarmes_7d', 'qtd_alarmes_14d',
            'aceleracao_alarmes_7d', 'qtd_anomalias_30s_14d',
            'tempo_parada_acumulado_7d', 'max_tempo_normalizacao_7d',
            'tempo_medio_por_alarme_7d', 'proporcao_alarmes_graves_7d'
        ]
        
        from sklearn.preprocessing import RobustScaler
        scaler = RobustScaler().fit(df_treino_enc[cols_para_escalonar])
        cols_scaled = [f"{c}_scaled" for c in cols_para_escalonar]
        df_treino_enc[cols_scaled] = scaler.transform(df_treino_enc[cols_para_escalonar])
        df_teste_enc[cols_scaled] = scaler.transform(df_teste_enc[cols_para_escalonar])
        
        cols_estacoes = [c for c in df_treino_enc.columns if c.startswith('estacao_')]
        features = ['plataforma_bin', 'faixa_risco_zm01'] + cols_estacoes + cols_scaled
        target_col = 'target_falha_7d'
        
        cols_export = ['id_ativo', 'data_ref'] + features + [target_col]
        df_treino_enc[cols_export].to_csv(path_treino, index=False)
        df_teste_enc[cols_export].to_csv(path_teste, index=False)
        print(f"[{datetime.now().strftime('%H:%M:%S')}] ✅ Datasets v2 gerados com sucesso!")
        
        return df_treino_enc[features], df_treino_enc[target_col], df_teste_enc[features], df_teste_enc[target_col], features, df_teste_enc
        
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
    
    tscv = TimeSeriesSplit(n_splits=3)
    
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
        
        print(f"[{datetime.now().strftime('%H:%M:%S')}] Trial {trial.number+1:03d}/{args.n_trials} | Precisão: {mean_prec*100:5.2f}% | Recall: {mean_rec*100:5.2f}% | F1: {mean_f1:.4f} | (depth={params['max_depth']}, lr={params['learning_rate']:.4f}, scale_pos={params['scale_pos_weight']:.2f})")
        
        return mean_f1

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
    
    # -------------------------------------------------------------
    # RASTREAMENTO E EXPORTAÇÃO DOS 3 CAMPEÕES INDIVIDUAIS
    # -------------------------------------------------------------
    trials = study.trials
    
    best_trial_f1 = max(trials, key=lambda t: t.user_attrs.get('f1_score', 0))
    best_trial_prec = max(trials, key=lambda t: t.user_attrs.get('precision', 0))
    best_trial_rec = max(trials, key=lambda t: t.user_attrs.get('recall', 0))
    
    campeoes = [
        ('🏆 CAMPEÃO GERAL (MELHOR F1-SCORE / EQUILÍBRIO)', best_trial_f1, 'f1_equilibrado'),
        ('🎯 CAMPEÃO DE PRECISÃO (MAIOR PRECISÃO OPERACIONAL)', best_trial_prec, 'alta_precisao'),
        ('🚨 CAMPEÃO DE RECALL (MAIOR COBERTURA DE FALHAS)', best_trial_rec, 'alto_recall')
    ]
    
    print("\n" + "=" * 95)
    print(f"🏁 OTIMIZAÇÃO CONCLUÍDA EM {total_time/60:.2f} MINUTOS!")
    print("=" * 95)
    
    relatorios_campeoes = {}
    os.makedirs("data/modelos_artefatos", exist_ok=True)
    
    for titulo, t, slug in campeoes:
        print(f"\n{titulo}:")
        print(f"   -> Validação: Precisão = {t.user_attrs.get('precision', 0)*100:5.2f}% | Recall = {t.user_attrs.get('recall', 0)*100:5.2f}% | F1 = {t.user_attrs.get('f1_score', 0):.4f}")
        print("   -> Hiperparâmetros Ótimos:")
        for k, v in t.params.items():
            print(f"        '{k}': {v},")
            
        params_t = t.params.copy()
        params_t.update({
            'tree_method': 'hist',
            'device': args.device,
            'random_state': 42,
            'eval_metric': 'logloss'
        })
        
        m_t = XGBClassifier(**params_t).fit(X_train, y_train)
        probs_t = m_t.predict_proba(X_test)[:, 1]
        
        df_teste['prob_t'] = probs_t
        df_teste['semana'] = pd.to_datetime(df_teste['data_ref']).dt.isocalendar().week
        
        df_sem = df_teste.groupby(['id_ativo', 'semana']).agg(
            max_prob=('prob_t', 'max'),
            houve_falha=('target_falha_7d', 'max')
        ).reset_index()
        
        df_sem['pred'] = (df_sem['max_prob'] >= args.threshold).astype(int)
        ac = ((df_sem['pred'] == 1) & (df_sem['houve_falha'] == 1)).sum()
        al = (df_sem['pred'] == 1).sum()
        tot = df_sem['houve_falha'].sum()
        
        prec_t = ac / al if al > 0 else 0
        rec_t = ac / tot if tot > 0 else 0
        f1_t = 2 * prec_t * rec_t / (prec_t + rec_t + 1e-10)
        roc_t = roc_auc_score(y_test, probs_t)
        pr_t = average_precision_score(y_test, probs_t)
        
        print(f"   📊 Performance no Teste Real 2026:")
        print(f"        -> Precisão Operacional: {prec_t*100:.2f}% ({ac:,} acertos em {al:,} chamados)")
        print(f"        -> Recall Operacional:    {rec_t*100:.2f}% ({ac:,} de {tot:,} falhas evitadas)")
        print(f"        -> F1-Score Operacional:  {f1_t:.4f} | ROC-AUC: {roc_t:.4f} | PR-AUC: {pr_t:.4f}")
        
        caminho_mod = f"data/modelos_artefatos/modelo_xgboost_{slug}.joblib"
        joblib.dump(m_t, caminho_mod)
        print(f"        -> ✅ Modelo persistido em: {caminho_mod}")
        
        relatorios_campeoes[slug] = {
            'perfil': titulo,
            'hiperparametros': t.params,
            'validacao': {
                'precision': float(t.user_attrs.get('precision', 0)),
                'recall': float(t.user_attrs.get('recall', 0)),
                'f1_score': float(t.user_attrs.get('f1_score', 0))
            },
            'teste_real_2026': {
                'precision_operacional': float(prec_t),
                'recall_operacional': float(rec_t),
                'f1_score': float(f1_t),
                'roc_auc': float(roc_t),
                'pr_auc': float(pr_t),
                'acertos': int(ac),
                'alertas_totais': int(al),
                'falhas_totais': int(tot)
            }
        }
        
    caminho_json = "data/modelos_artefatos/relatorio_tres_campeoes.json"
    with open(caminho_json, 'w', encoding='utf-8') as f:
        json.dump(relatorios_campeoes, f, indent=2, ensure_ascii=False)
        
    print("\n" + "=" * 95)
    print(f"✅ Todos os 3 modelos e o relatório consolidado foram salvos com sucesso!")
    print(f"   -> Relatório JSON Completo: {caminho_json}")
    print("=" * 95)

if __name__ == "__main__":
    main()
