"""
Busca de hiperparametros da RNN com Optuna.

Mantem as 12 features atuais e a janela de 30 dias. O
objetivo de cada trial e o PR-AUC medio em 2 folds temporais expansivos
dentro do treino (mesmo espirito do TimeSeriesSplit usado no Modelo 5), sem
tocar no teste.

Uso:
    python optuna_search.py [--trials N] [--jobs N] [--epocas-busca N]

Grava:
    optuna_study.db      -- storage sqlite, permite retomar/inspecionar depois
    optuna_trials.csv    -- todos os trials, parametros e resultado
    optuna_best.json     -- melhor configuracao encontrada
"""
import argparse
import json
import time

import optuna
from optuna.samplers import TPESampler

import rnn_utils as R

optuna.logging.set_verbosity(optuna.logging.WARNING)


def espaco_busca(trial):
    layers = trial.suggest_categorical('layers', [1, 2])
    return {
        'cell': trial.suggest_categorical('cell', ['lstm', 'gru']),
        'hidden': trial.suggest_categorical('hidden', [16, 32, 64]),
        'layers': layers,
        # o dropout so atua na saida quando layers=1 (ver RNNFalha.forward); com 2 camadas
        # tambem atua entre as camadas recorrentes
        'dropout': trial.suggest_float('dropout', 0.0, 0.4),
        'lr': trial.suggest_float('lr', 1e-4, 5e-3, log=True),
        'batch': trial.suggest_categorical('batch', [512, 1024]),
        'weight_decay': trial.suggest_categorical('weight_decay', [0.0, 1e-5, 1e-4, 1e-3]),
        'pos_weight_mult': trial.suggest_float('pos_weight_mult', 0.5, 2.0),
    }


def faz_objetivo(X, y, folds, max_epocas, paciencia):
    def objetivo(trial):
        cfg = espaco_busca(trial)
        cfg['max_epocas'], cfg['paciencia'] = max_epocas, paciencia

        pr_aucs = []
        for i, (m_fit, m_val) in enumerate(folds):
            R.seed_tudo(42)
            norma = R.estatisticas_norm(X[m_fit])
            _, prauc, _ = R.treina(X[m_fit], y[m_fit], X[m_val], y[m_val], norma, cfg, verboso=False)
            pr_aucs.append(prauc)

            trial.report(sum(pr_aucs) / len(pr_aucs), step=i)
            if trial.should_prune():
                raise optuna.TrialPruned()

        return sum(pr_aucs) / len(pr_aucs)
    return objetivo


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--trials', type=int, default=40)
    ap.add_argument('--jobs', type=int, default=4)
    ap.add_argument('--folds', type=int, default=2)
    ap.add_argument('--epocas-busca', type=int, default=8)
    ap.add_argument('--paciencia-busca', type=int, default=3)
    args = ap.parse_args()

    R.set_threads(1)  # 1 thread por trial -- a paralelizacao vem do n_jobs do optuna
    X, y, idx, m = R.carrega_dados()
    folds = R.folds_temporais(idx, m['treino'], n_folds=args.folds)
    print(f'{len(folds)} folds temporais dentro do treino (teste nao usado nesta etapa)')

    study = optuna.create_study(
        study_name='rnn_falha_porta',
        direction='maximize',
        sampler=TPESampler(seed=42),
        pruner=optuna.pruners.MedianPruner(n_warmup_steps=1),
        storage='sqlite:///optuna_study.db',
        load_if_exists=True,
    )

    ja_rodados = len(study.trials)
    faltam = max(args.trials - ja_rodados, 0)
    print(f'trials ja no storage: {ja_rodados} | faltam: {faltam}')

    if faltam > 0:
        t0 = time.time()
        study.optimize(
            faz_objetivo(X, y, folds, args.epocas_busca, args.paciencia_busca),
            n_trials=faltam, n_jobs=args.jobs, show_progress_bar=False,
        )
        print(f'busca concluida em {time.time() - t0:.0f}s')

    df = study.trials_dataframe()
    df.to_csv('optuna_trials.csv', index=False)

    melhor = {
        'n_trials_total': len(study.trials),
        'n_trials_completos': len(study.get_trials(states=(optuna.trial.TrialState.COMPLETE,))),
        'n_trials_podados': len(study.get_trials(states=(optuna.trial.TrialState.PRUNED,))),
        'best_value_pr_auc_cv': study.best_value,
        'best_params': study.best_params,
        'folds': args.folds,
        'epocas_busca': args.epocas_busca,
        'paciencia_busca': args.paciencia_busca,
    }
    with open('optuna_best.json', 'w', encoding='utf-8') as f:
        json.dump(melhor, f, indent=2, ensure_ascii=False)

    print(json.dumps(melhor, indent=2, ensure_ascii=False))


if __name__ == '__main__':
    main()
