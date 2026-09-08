"""Valida os artefatos gerados pelo pipeline sequencial.

O validador não treina nem executa os modelos. Ele confere o contrato entre as
saídas e termina com código diferente de zero quando encontra inconsistências.
"""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path

import numpy as np
import pandas as pd


DIRETORIO_PIPELINE = Path(__file__).resolve().parent
RAIZ_REPOSITORIO = DIRETORIO_PIPELINE.parents[1]

ARQUIVOS_OBRIGATORIOS = {
    "previsoes": "previsoes_modelo_7_pipeline.csv",
    "intermediaria": "saida_intermediaria_modelo_7.csv",
    "mapa": "mapa_chaves_portas.csv",
    "portas": "saida_portas_modelo_5.csv",
    "priorizadas": "saida_final_portas_priorizadas.csv",
    "status": "status_plataformas_pipeline.csv",
    "recomendacoes": "recomendacoes_manutencao.csv",
    "metricas": "metricas_pipeline_teste.json",
    "resumo": "resumo_execucao_pipeline.json",
}


def exigir(condicao: bool, mensagem: str) -> None:
    if not condicao:
        raise AssertionError(mensagem)


def sha256(caminho: Path) -> str:
    return hashlib.sha256(caminho.read_bytes()).hexdigest()


def ler_csv(caminho: Path, datas: tuple[str, ...] = ()) -> pd.DataFrame:
    return pd.read_csv(caminho, parse_dates=list(datas))


def validar_probabilidade(tabela: pd.DataFrame, coluna: str) -> None:
    exigir(tabela[coluna].notna().all(), f"{coluna} contém valores ausentes.")
    exigir(
        tabela[coluna].between(0, 1).all(),
        f"{coluna} contém valores fora do intervalo [0, 1].",
    )


def validar(pasta: Path) -> dict[str, int]:
    caminhos = {nome: pasta / arquivo for nome, arquivo in ARQUIVOS_OBRIGATORIOS.items()}
    ausentes = [str(caminho) for caminho in caminhos.values() if not caminho.is_file()]
    exigir(not ausentes, "Arquivos obrigatórios ausentes: " + ", ".join(ausentes))

    previsoes = ler_csv(caminhos["previsoes"], ("data_referencia",))
    intermediaria = ler_csv(caminhos["intermediaria"], ("data_referencia",))
    mapa = ler_csv(caminhos["mapa"])
    portas = ler_csv(caminhos["portas"], ("data_referencia",))
    priorizadas = ler_csv(caminhos["priorizadas"], ("data_referencia",))
    status = ler_csv(caminhos["status"], ("data_referencia",))
    recomendacoes = ler_csv(
        caminhos["recomendacoes"],
        ("data_referencia", "inicio_horizonte_previsao", "fim_horizonte_previsao"),
    )
    metricas = json.loads(caminhos["metricas"].read_text(encoding="utf-8"))
    resumo = json.loads(caminhos["resumo"].read_text(encoding="utf-8"))

    chave_plataforma = ["data_referencia", "id_psd"]
    chave_porta = ["data_referencia", "id_porta_pipeline"]

    exigir(not previsoes.duplicated(chave_plataforma).any(), "Previsões de plataforma duplicadas.")
    exigir(not intermediaria.duplicated(chave_plataforma).any(), "Saída intermediária duplicada.")
    exigir(not status.duplicated(chave_plataforma).any(), "Status de plataforma duplicado.")
    exigir(not portas.duplicated(chave_porta).any(), "Scores de porta duplicados.")
    exigir(not priorizadas.duplicated(chave_porta).any(), "Portas priorizadas duplicadas.")
    exigir(not recomendacoes.duplicated(chave_porta).any(), "Recomendações duplicadas.")
    exigir(not mapa["id_porta_pipeline"].duplicated().any(), "Mapa contém porta canônica duplicada.")
    exigir(not mapa["id_porta_modelo_5"].duplicated().any(), "Mapa contém porta do Modelo 5 duplicada.")

    validar_probabilidade(previsoes, "probabilidade_falha_7d")
    if not portas.empty:
        validar_probabilidade(portas, "probabilidade_porta")
    if not recomendacoes.empty:
        validar_probabilidade(recomendacoes, "probabilidade_plataforma")
        validar_probabilidade(recomendacoes, "probabilidade_porta")

    limiar_7 = float(resumo["modelos"]["modelo_7"]["limiar"])
    decisao_7 = previsoes["probabilidade_falha_7d"].ge(limiar_7).astype(int)
    exigir(
        decisao_7.equals(previsoes["predicao_falha_7d"].astype(int)),
        "Decisões do Modelo 7 não correspondem ao limiar registrado.",
    )
    exigir(
        set(map(tuple, previsoes[chave_plataforma].to_numpy()))
        == set(map(tuple, intermediaria[chave_plataforma].to_numpy()))
        == set(map(tuple, status[chave_plataforma].to_numpy())),
        "As saídas não preservam todas as plataformas e datas previstas.",
    )

    colunas_proibidas = {"y", "target", "n_falhas_7d_futuro"}
    for nome, tabela in {
        "previsões": previsoes,
        "saída intermediária": intermediaria,
        "scores de porta": portas,
        "recomendações": recomendacoes,
    }.items():
        exigir(
            not colunas_proibidas.intersection(tabela.columns),
            f"{nome} expõe target na saída de inferência.",
        )

    if not portas.empty:
        limiar_5 = float(resumo["modelos"]["modelo_5"]["limiar"])
        decisao_5 = portas["probabilidade_porta"].ge(limiar_5).astype(int)
        exigir(
            decisao_5.equals(portas["alerta_porta"].astype(int)),
            "Decisões do Modelo 5 não correspondem ao limiar registrado.",
        )
        grupos = portas.sort_values(
            [*chave_plataforma, "posicao_risco_na_plataforma"]
        ).groupby(chave_plataforma, sort=False)
        exigir(
            all(
                np.array_equal(
                    grupo["posicao_risco_na_plataforma"].to_numpy(),
                    np.arange(1, len(grupo) + 1),
                )
                for _, grupo in grupos
            ),
            "O ranking das portas não é consecutivo dentro da plataforma.",
        )

    chaves_alerta = set(
        map(tuple, portas.loc[portas.get("alerta_porta", pd.Series(dtype=int)).eq(1), chave_porta].to_numpy())
    )
    chaves_priorizadas = set(map(tuple, priorizadas[chave_porta].to_numpy()))
    exigir(chaves_alerta == chaves_priorizadas, "A saída priorizada diverge dos alertas de porta.")

    maximo = int(resumo["maximo_portas_recomendadas_por_plataforma"])
    if not recomendacoes.empty:
        exigir(
            recomendacoes.groupby(chave_plataforma).size().le(maximo).all(),
            "O limite de recomendações por plataforma foi ultrapassado.",
        )
        exigir(
            recomendacoes["posicao_risco_na_plataforma"].between(1, maximo).all(),
            "A lista operacional não contém as primeiras posições do ranking.",
        )
        exigir(
            (recomendacoes["inicio_horizonte_previsao"] - recomendacoes["data_referencia"])
            .dt.days.eq(1).all(),
            "O início do horizonte não corresponde a D+1.",
        )
        exigir(
            (recomendacoes["fim_horizonte_previsao"] - recomendacoes["data_referencia"])
            .dt.days.eq(7).all(),
            "O fim do horizonte não corresponde a D+7.",
        )
        exigir(
            set(map(tuple, recomendacoes[chave_porta].to_numpy())).issubset(chaves_priorizadas),
            "A lista operacional contém porta que não superou os dois limiares.",
        )
    recomendacoes_esperadas = set(
        map(
            tuple,
            priorizadas.loc[
                priorizadas["posicao_risco_na_plataforma"].le(maximo), chave_porta
            ].to_numpy(),
        )
    )
    exigir(
        set(map(tuple, recomendacoes[chave_porta].to_numpy())) == recomendacoes_esperadas,
        "A lista operacional diverge das primeiras posições do ranking.",
    )

    contagens = resumo["contagens"]
    verificacoes_contagem = {
        "previsoes_plataforma_dia": len(status),
        "alertas_plataforma_dia": int(status["alerta_plataforma"].sum()),
        "registros_porta_dia_avaliados": len(portas),
        "alertas_porta_dia": len(priorizadas),
        "portas_distintas_com_alerta": int(priorizadas["id_porta_pipeline"].nunique()),
        "recomendacoes_porta_dia": len(recomendacoes),
        "portas_distintas_recomendadas": int(recomendacoes["id_porta_pipeline"].nunique()),
    }
    pares_avaliados = portas[chave_plataforma].drop_duplicates()
    verificacoes_contagem["plataformas_dia_analisadas_pelo_modelo_5"] = len(
        pares_avaliados
    )
    for nome, valor in verificacoes_contagem.items():
        exigir(int(contagens[nome]) == valor, f"Contagem inconsistente no resumo: {nome}.")

    for nome_modelo in ("modelo_7", "modelo_5"):
        registro = resumo["modelos"][nome_modelo]
        caminho_modelo = RAIZ_REPOSITORIO / registro["arquivo"]
        exigir(caminho_modelo.is_file(), f"Modelo ausente: {caminho_modelo}.")
        exigir(sha256(caminho_modelo) == registro["sha256"], f"Hash divergente: {nome_modelo}.")

    registro_modelo_5 = resumo["modelos"]["modelo_5"]
    caminho_metadata_5 = RAIZ_REPOSITORIO / registro_modelo_5["arquivo_metadata"]
    exigir(caminho_metadata_5.is_file(), f"Metadata ausente: {caminho_metadata_5}.")
    exigir(
        sha256(caminho_metadata_5) == registro_modelo_5["sha256_metadata"],
        "Hash divergente: metadata do Modelo 5.",
    )

    status_esperados = {
        str(nome): int(quantidade)
        for nome, quantidade in status["status_final"].value_counts().items()
    }
    exigir(
        resumo["quantidade_por_status_final"] == status_esperados,
        "As contagens por status final estão inconsistentes.",
    )

    exigir(
        metricas["status_avaliacao"] == resumo["status_avaliacao"],
        "Os relatórios discordam sobre o estado da avaliação.",
    )
    return verificacoes_contagem


def main() -> None:
    parser = argparse.ArgumentParser(description="Valida uma execução do pipeline sequencial.")
    parser.add_argument(
        "--pasta",
        type=Path,
        default=DIRETORIO_PIPELINE / "artefatos",
        help="Diretório que contém os artefatos da execução.",
    )
    argumentos = parser.parse_args()
    pasta = argumentos.pasta.resolve()
    try:
        contagens = validar(pasta)
    except (AssertionError, KeyError, ValueError, OSError, json.JSONDecodeError) as erro:
        raise SystemExit(f"VALIDAÇÃO FALHOU: {erro}") from erro

    print("VALIDAÇÃO CONCLUÍDA")
    for nome, valor in contagens.items():
        print(f"- {nome}: {valor:,}")


if __name__ == "__main__":
    main()
