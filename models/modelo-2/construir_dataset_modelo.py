"""Monta o dataset final para o modelo preditivo.

Lê as tabelas já limpas em models/sprint-2/modelo_2/data/ (geradas por
limpeza_consolidacao.py) e produz uma única tabela onde cada linha é uma
porta de uma estação em um dia, com:

- label `falha_7d`: 1 se houve ordem corretiva (ZM01 ou ZM02) nessa porta
  nos próximos 7 dias (d+1 a d+7).
- features de histórico de alarmes, uso (Catracas) e manutenção anterior.
"""

import re
from pathlib import Path

import numpy as np
import pandas as pd

BASE = Path(__file__).resolve().parent / "data"
JANELA_LABEL_DIAS = 7


def extrair_estacao_porta(local):
    if pd.isna(local):
        return None, None
    m_est = re.search(r"PSD-([A-Z]{3})", local)
    estacao = m_est.group(1) if m_est else None
    m_porta = re.search(r"PP(\d+)PDM", local)
    porta = int(m_porta.group(1)) if m_porta else None
    return estacao, porta


def carregar_corretivas():
    zm01 = pd.read_csv(BASE / "zm01_corretiva_emergencial_limpo.csv", parse_dates=["inicio_avaria_datetime"])
    zm02 = pd.read_csv(BASE / "zm02_corretiva_programada_limpo.csv", parse_dates=["inicio_avaria_datetime"])
    zm01["origem"] = "ZM01"
    zm02["origem"] = "ZM02"
    cols = ["ordem", "local_de_instalacao", "inicio_avaria_datetime", "origem"]
    corretivas = pd.concat([zm01[cols], zm02[cols]], ignore_index=True)
    corretivas[["estacao", "porta"]] = corretivas["local_de_instalacao"].apply(
        lambda x: pd.Series(extrair_estacao_porta(x))
    )
    corretivas["data"] = corretivas["inicio_avaria_datetime"].dt.normalize()

    total = len(corretivas)
    com_porta = corretivas["porta"].notna().sum()
    print(f"[Corretivas ZM01+ZM02] {total:,} ordens; {com_porta:,} ({com_porta/total:.1%}) com porta identificavel")

    return corretivas.dropna(subset=["estacao", "porta", "data"]).copy()


def carregar_preventiva():
    zm03 = pd.read_csv(BASE / "zm03_preventiva_limpo.csv", parse_dates=["DT_Inicio"])
    zm03 = zm03.rename(columns={"Estacao": "estacao", "DT_Inicio": "data_hora"})
    zm03["data"] = zm03["data_hora"].dt.normalize()
    return zm03.dropna(subset=["estacao", "data"])[["estacao", "data"]]


def carregar_alarmes():
    alarmes = pd.read_csv(
        BASE / "alarmes_limpo.csv",
        usecols=["horario_alarme", "estacao", "porta", "alarme", "tempo_normalizacao_seg"],
        parse_dates=["horario_alarme"],
    )
    alarmes = alarmes.dropna(subset=["estacao", "porta"]).copy()
    alarmes["porta"] = alarmes["porta"].astype(int)
    alarmes["data"] = alarmes["horario_alarme"].dt.normalize()
    return alarmes


def carregar_catracas():
    catracas = pd.read_csv(BASE / "catracas_limpo.csv", parse_dates=["data"])
    return catracas


def construir_grade(doors, dates):
    grade = doors.merge(pd.DataFrame({"data": dates}), how="cross")
    return grade.sort_values(["estacao", "porta", "data"]).reset_index(drop=True)


def rolling_passado(serie_diaria, janela):
    """Soma dos ultimos `janela` dias anteriores ao dia corrente (exclui o dia atual)."""
    return serie_diaria.rolling(janela, min_periods=1).sum().shift(1).fillna(0)


def dias_desde_ultimo_evento(flag_diario):
    """Para uma serie diaria 0/1 (evento no dia), retorna dias desde o ultimo
    evento ANTES do dia atual (NaN se nunca houve)."""
    idx = np.arange(len(flag_diario))
    ultimo_idx = np.where(flag_diario.values > 0, idx, np.nan)
    ultimo_idx_ffill = pd.Series(ultimo_idx).ffill().shift(1)
    return pd.Series(idx) - ultimo_idx_ffill


def label_falha_futura(flag_diario, janela):
    """1 se houve evento nos proximos `janela` dias (d+1..d+janela)."""
    invertido = flag_diario[::-1].reset_index(drop=True)
    soma_futura = invertido.rolling(janela, min_periods=1).sum().shift(1).fillna(0)
    return soma_futura[::-1].reset_index(drop=True).gt(0).astype(int)


def main():
    corretivas = carregar_corretivas()
    preventiva = carregar_preventiva()
    alarmes = carregar_alarmes()
    catracas = carregar_catracas()

    doors = alarmes[["estacao", "porta"]].drop_duplicates()

    data_inicio = max(alarmes["data"].min(), catracas["data"].min(), corretivas["data"].min())
    data_fim = min(alarmes["data"].max(), catracas["data"].max(), corretivas["data"].max()) - pd.Timedelta(
        days=JANELA_LABEL_DIAS
    )
    dates = pd.date_range(data_inicio, data_fim, freq="D")
    print(f"[Painel] {len(doors)} portas x {len(dates)} dias ({data_inicio.date()} a {data_fim.date()})")

    painel = construir_grade(doors, dates)

    alarmes_diario = (
        alarmes.groupby(["estacao", "porta", "data"])
        .agg(n_alarmes=("alarme", "size"), n_alarmes_obs=("alarme", lambda s: (s == "OBS").sum()),
             tempo_normalizacao_medio=("tempo_normalizacao_seg", "mean"))
        .reset_index()
    )
    painel = painel.merge(alarmes_diario, on=["estacao", "porta", "data"], how="left")
    for col in ["n_alarmes", "n_alarmes_obs"]:
        painel[col] = painel[col].fillna(0)

    corretivas_diario = (
        corretivas.groupby(["estacao", "porta", "data"]).size().reset_index(name="n_corretivas_dia")
    )
    painel = painel.merge(corretivas_diario, on=["estacao", "porta", "data"], how="left")
    painel["n_corretivas_dia"] = painel["n_corretivas_dia"].fillna(0)

    preventiva_diario = preventiva.groupby(["estacao", "data"]).size().reset_index(name="n_preventivas_dia")
    painel = painel.merge(preventiva_diario, on=["estacao", "data"], how="left")
    painel["n_preventivas_dia"] = painel["n_preventivas_dia"].fillna(0)

    painel = painel.merge(
        catracas[["estacao", "data", "passageiros_entrada", "passageiros_saida"]],
        on=["estacao", "data"],
        how="left",
    )

    def features_por_porta(grupo):
        grupo = grupo.sort_values("data").reset_index(drop=True)

        grupo["n_alarmes_3d"] = rolling_passado(grupo["n_alarmes"], 3)
        grupo["n_alarmes_7d"] = rolling_passado(grupo["n_alarmes"], 7)
        grupo["n_alarmes_14d"] = rolling_passado(grupo["n_alarmes"], 14)
        grupo["n_alarmes_obs_7d"] = rolling_passado(grupo["n_alarmes_obs"], 7)
        grupo["tempo_medio_normalizacao_7d"] = (
            grupo["tempo_normalizacao_medio"].rolling(7, min_periods=1).mean().shift(1)
        )
        grupo["dias_desde_ultimo_alarme"] = dias_desde_ultimo_evento(grupo["n_alarmes"].gt(0).astype(int))

        grupo["n_corretivas_90d"] = rolling_passado(grupo["n_corretivas_dia"], 90)
        grupo["dias_desde_ultima_corretiva"] = dias_desde_ultimo_evento(grupo["n_corretivas_dia"].gt(0).astype(int))

        grupo["dias_desde_ultima_preventiva"] = dias_desde_ultimo_evento(grupo["n_preventivas_dia"].gt(0).astype(int))

        grupo["passageiros_media_7d"] = (
            grupo["passageiros_entrada"].rolling(7, min_periods=1).mean().shift(1)
        )

        grupo["falha_7d"] = label_falha_futura(grupo["n_corretivas_dia"].gt(0).astype(int), JANELA_LABEL_DIAS)

        return grupo

    painel = (
        painel.groupby(["estacao", "porta"], group_keys=True)
        .apply(features_por_porta)
        .reset_index(level=[0, 1])
        .reset_index(drop=True)
    )

    colunas_finais = [
        "data", "estacao", "porta", "falha_7d",
        "n_alarmes_3d", "n_alarmes_7d", "n_alarmes_14d", "n_alarmes_obs_7d",
        "tempo_medio_normalizacao_7d", "dias_desde_ultimo_alarme",
        "passageiros_entrada", "passageiros_saida", "passageiros_media_7d",
        "dias_desde_ultima_corretiva", "dias_desde_ultima_preventiva", "n_corretivas_90d",
    ]
    painel_final = painel[colunas_finais].sort_values(["estacao", "porta", "data"]).reset_index(drop=True)

    saida = BASE.parent / "dataset_modelo_porta_dia.csv"
    painel_final.to_csv(saida, index=False)

    print(f"\n[Dataset final] {len(painel_final):,} linhas x {len(colunas_finais)} colunas")
    print(f"Taxa de positivos (falha_7d=1): {painel_final['falha_7d'].mean():.2%}")
    print("Salvo em:", saida)


if __name__ == "__main__":
    main()
