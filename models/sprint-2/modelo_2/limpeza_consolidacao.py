"""Limpeza e consolidação das fontes brutas em data/ para uso no modelo preditivo.

Reaplica a mesma lógica documentada em eda/cleaning/ (dedup exato, remoção de
linhas incompletas/subtotais, parsing de datas e de TAGs) e exporta uma tabela
CSV limpa por fonte em models/sprint-2/modelo_2/data/.

- Alarmes: concatena as 3 extrações brutas (incluindo a nova "Estações -
  Alarmes.xlsx", ainda não incorporada ao pipeline em eda/cleaning), remove
  duplicatas exatas, remove linhas sem TAG/horário do alarme e remove alarmes
  com duração > 30 dias (tickets travados), igual a
  eda/cleaning/limpeza_dados_alarmes.ipynb.
- ZM01 / ZM02: reaproveita as versões já limpas em data/.../processados/,
  só exporta como CSV no novo local.
- ZM03 (Preventiva): reaproveita o CSV já limpo em data/Preventiva/.
- Catracas: reconstrói o cabeçalho mesclado do Excel e reformata para
  formato longo (data, estação, entrada, saída), sem tabela de limpeza
  prévia (nova).
- Materiais: remove linhas de subtotal (sem `Ordem`) e duplicatas exatas do
  catálogo e das reservas, sem tabela de limpeza prévia (nova).
"""

from pathlib import Path
import re
import unicodedata

import numpy as np
import pandas as pd

RAIZ = Path(__file__).resolve().parents[3]
DATA = RAIZ / "data"
SAIDA = Path(__file__).resolve().parent / "data"
SAIDA.mkdir(parents=True, exist_ok=True)


def normalizar_coluna(nome: str) -> str:
    nome = str(nome).strip().lower()
    nome = unicodedata.normalize("NFKD", nome)
    nome = nome.encode("ascii", "ignore").decode("ascii")
    nome = re.sub(r"[^a-z0-9]+", "_", nome)
    return nome.strip("_")

def limpar_alarmes() -> pd.DataFrame:
    pasta = DATA / "Alarmes"
    caminho_estacoes = [p for p in pasta.glob("*.xlsx") if "Esta" in p.name][0]

    df1 = pd.read_excel(pasta / "Alarmes do sistema de PSD.xlsx")
    df2 = pd.read_excel(pasta / "Histórico de alarmes 23-26.xlsx")
    df3 = pd.read_excel(caminho_estacoes, sheet_name="Export")

    df_raw = pd.concat([df1, df2, df3], ignore_index=True)
    total_inicial = len(df_raw)

    df_dedup = df_raw.drop_duplicates().copy()
    df_clean = df_dedup.dropna(subset=["Horário do Alarme", "TAG"]).copy()

    def parse_tempo_normalizacao(val):
        if pd.isna(val):
            return np.nan
        val_str = str(val).strip()
        try:
            if "dias," in val_str:
                dias_part, time_part = val_str.split("dias,")
                dias = int(dias_part.strip().split()[0])
            else:
                dias = 0
                time_part = val_str
            h, m, s = map(int, time_part.strip().split(":"))
            return dias * 86400 + h * 3600 + m * 60 + s
        except Exception:
            return np.nan

    df_clean["tempo_normalizacao_seg"] = df_clean["Tempo de Normalização"].apply(
        parse_tempo_normalizacao
    )
    df_clean["dt_alarme"] = pd.to_datetime(df_clean["Horário do Alarme"], errors="coerce")
    df_clean["dt_normalizacao"] = pd.to_datetime(
        df_clean["Horário da Normalizacao"], errors="coerce"
    )

    def extrair_atributos_tag(tag):
        if pd.isna(tag):
            return pd.Series([np.nan, np.nan, np.nan, np.nan])
        tag_str = str(tag).strip()

        match_p = re.search(r"PSD_P_([A-Z]{3})(\d+)_([A-Z0-9]+)_([A-Z0-9]+)", tag_str)
        if match_p:
            estacao = match_p.group(1)
            plataforma = f"{match_p.group(1)}{match_p.group(2)}"
            porta_match = re.search(r"\d+", match_p.group(3))
            porta = int(porta_match.group(0)) if porta_match else np.nan
            tipo = match_p.group(4)
            return pd.Series([estacao, plataforma, porta, tipo])

        match_std = re.search(
            r"PSD_([A-Z]{3})_([A-Z0-9]+)_(?:PDM|MDU)(\d+)_([A-Z0-9]+)", tag_str
        )
        if match_std:
            estacao = match_std.group(1)
            plataforma = match_std.group(2)
            porta = int(match_std.group(3))
            tipo = match_std.group(4)
            return pd.Series([estacao, plataforma, porta, tipo])

        parts = tag_str.split("_")
        if len(parts) >= 3:
            if parts[1] == "P":
                est = re.sub(r"\d+", "", parts[2])
                plat = parts[2]
            else:
                est = parts[1]
                plat = parts[2]
            porta_match = re.search(r"\d+", parts[3]) if len(parts) > 3 else None
            porta = int(porta_match.group(0)) if porta_match else np.nan
            tipo = parts[-1] if len(parts) > 4 else np.nan
            return pd.Series([est, plat, porta, tipo])

        return pd.Series([np.nan, np.nan, np.nan, np.nan])

    df_clean[["estacao", "plataforma", "porta", "tipo_evento"]] = df_clean["TAG"].apply(
        extrair_atributos_tag
    )

    limite_seg = 30 * 86400
    df_final = df_clean[
        (df_clean["tempo_normalizacao_seg"] >= 0)
        & (df_clean["tempo_normalizacao_seg"] <= limite_seg)
    ].copy()

    df_final = df_final.rename(
        columns={
            "TAG": "tag",
            "Horário do Alarme": "horario_alarme",
            "Estação": "estacao_original",
            "Porta": "porta_original",
            "Alarme": "alarme",
            "Horário da Normalizacao": "horario_normalizacao",
            "Tempo de Normalização": "tempo_normalizacao_texto",
        }
    )

    print(
        f"[Alarmes] bruto={total_inicial:,} -> dedup={len(df_dedup):,} -> "
        f"sem nulos criticos={len(df_clean):,} -> final(<=30d)={len(df_final):,}"
    )
    return df_final

def carregar_zm01() -> pd.DataFrame:
    caminho = DATA / "Corretiva Emergencial" / "processados" / "ZM01_limpa.xlsx"
    df = pd.read_excel(caminho, sheet_name="ZM01_limpa")
    print(f"[ZM01] carregado de processados/: {len(df):,} linhas")
    return df


def carregar_zm02() -> pd.DataFrame:
    caminho = DATA / "Corretiva Programada" / "processados" / "ZM02_limpa.xlsx"
    df = pd.read_excel(caminho, sheet_name="ZM02_limpa")
    print(f"[ZM02] carregado de processados/: {len(df):,} linhas")
    return df


def carregar_zm03() -> pd.DataFrame:
    caminho = DATA / "Preventiva" / "preventiva_zm03_limpa.csv"
    df = pd.read_csv(caminho)
    print(f"[ZM03] carregado ja limpo: {len(df):,} linhas")
    return df

def limpar_catracas() -> pd.DataFrame:
    caminho = DATA / "Catracas" / "Historico L4.xlsx"
    bruto = pd.read_excel(caminho, sheet_name="Linha 4", header=None)
    total_inicial = len(bruto) - 5

    nomes_estacao = list(bruto.iloc[3, :])
    tipo_coluna = list(bruto.iloc[4, :])

    mapa_estacao = {
        " Butantã": "BUT",
        " Pinheiros": "PIN",
        " Faria Lima": "FAL",
        " Paulista": "PTA",
        " Luz": "LUZ",
        "Fradique Coutinho": "FRA",
        "Higienópolis Mackenzie": "HIG",
        "Oscar Freire": "FRE",
        "São Paulo Morumbi": "MBI",
        "Vila Sônia": "VSO",
    }

    dados = bruto.iloc[5:, :].copy()
    dados.columns = range(dados.shape[1])
    dados = dados.rename(columns={1: "data"})
    dados["data"] = pd.to_datetime(dados["data"], errors="coerce")

    registros = []
    for col_idx in range(10, 30, 2):  # bloco "Estações Bloqueios"
        nome_raw = nomes_estacao[col_idx]
        codigo = mapa_estacao.get(nome_raw)
        if codigo is None:
            continue
        entrada = pd.to_numeric(dados[col_idx], errors="coerce")
        saida = pd.to_numeric(dados[col_idx + 1], errors="coerce")
        bloco = pd.DataFrame(
            {
                "data": dados["data"].values,
                "estacao": codigo,
                "passageiros_entrada": entrada.values,
                "passageiros_saida": saida.values,
            }
        )
        registros.append(bloco)

    df_final = pd.concat(registros, ignore_index=True)
    df_final = df_final.dropna(subset=["data"]).drop_duplicates()

    print(
        f"[Catracas] {total_inicial:,} dias x {len(mapa_estacao)} estacoes -> "
        f"{len(df_final):,} linhas (formato longo data x estacao)"
    )
    return df_final

def limpar_materiais_catalogo() -> pd.DataFrame:
    caminho = DATA / "Materiais" / "Lista de materiais do sistema PSD.XLSX"
    df = pd.read_excel(caminho)
    df.columns = [normalizar_coluna(c) for c in df.columns]
    antes = len(df)
    df = df.drop_duplicates()
    print(f"[Materiais/catalogo] {antes:,} -> {len(df):,} linhas (dedup exato)")
    return df


def limpar_materiais_reservas() -> pd.DataFrame:
    caminho = DATA / "Materiais" / "Ordens de reserva de materiais para ordens.xlsx"
    df = pd.read_excel(caminho)
    df.columns = [normalizar_coluna(c) for c in df.columns]
    antes = len(df)

    # linhas de subtotal do export SAP: "Ordem" vazio e "tipo_de_ordem" tipo "ZM01 (1.317)"
    df = df.dropna(subset=["ordem"]).copy()
    apos_subtotais = len(df)

    df = df.drop(columns=["tipo_atividad_manut"], errors="ignore")  # 100% nulo
    df = df.drop_duplicates()

    print(
        f"[Materiais/reservas] {antes:,} -> {apos_subtotais:,} (sem subtotais) -> "
        f"{len(df):,} linhas (dedup exato)"
    )
    return df


if __name__ == "__main__":
    limpar_alarmes().to_csv(SAIDA / "alarmes_limpo.csv", index=False)
    carregar_zm01().to_csv(SAIDA / "zm01_corretiva_emergencial_limpo.csv", index=False)
    carregar_zm02().to_csv(SAIDA / "zm02_corretiva_programada_limpo.csv", index=False)
    carregar_zm03().to_csv(SAIDA / "zm03_preventiva_limpo.csv", index=False)
    limpar_catracas().to_csv(SAIDA / "catracas_limpo.csv", index=False)
    limpar_materiais_catalogo().to_csv(SAIDA / "materiais_catalogo_limpo.csv", index=False)
    limpar_materiais_reservas().to_csv(SAIDA / "materiais_reservas_limpo.csv", index=False)
    print("\nTodas as tabelas limpas foram exportadas para:", SAIDA)
