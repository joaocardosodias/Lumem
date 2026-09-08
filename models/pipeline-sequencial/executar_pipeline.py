"""Executa o notebook do pipeline sequencial sem alterar o arquivo original.

Exemplo:
    python models/pipeline-sequencial/executar_pipeline.py \
        --inicio 2025-10-01 --fim 2025-10-31 \
        --pasta-saida execucoes/outubro-2025
"""

from __future__ import annotations

import argparse
import json
import os
import re
from datetime import date
from pathlib import Path


DIRETORIO_PIPELINE = Path(__file__).resolve().parent
RAIZ_REPOSITORIO = DIRETORIO_PIPELINE.parents[1]
CAMINHO_NOTEBOOK = DIRETORIO_PIPELINE / "pipeline_sequencial_modelos_7_5.ipynb"


def data_iso(valor: str) -> str:
    """Valida e devolve uma data no formato ISO."""
    try:
        return date.fromisoformat(valor).isoformat()
    except ValueError as erro:
        raise argparse.ArgumentTypeError(
            f"Data inválida: {valor!r}. Use AAAA-MM-DD."
        ) from erro


def pasta_relativa(valor: str) -> str:
    """Aceita somente uma pasta localizada dentro do diretório do pipeline."""
    caminho = Path(valor)
    if caminho.is_absolute() or not caminho.parts or ".." in caminho.parts:
        raise argparse.ArgumentTypeError(
            "A pasta de saída deve ser relativa a models/pipeline-sequencial/."
        )
    return caminho.as_posix()


def substituir_atribuicao(codigo: str, nome: str, valor: object) -> str:
    """Substitui uma atribuição simples da célula de configuração."""
    novo_codigo, quantidade = re.subn(
        rf"(?m)^{re.escape(nome)}\s*=.*$",
        f"{nome} = {valor!r}",
        codigo,
        count=1,
    )
    if quantidade != 1:
        raise RuntimeError(f"Configuração {nome!r} não encontrada no notebook.")
    return novo_codigo


def configurar_notebook(notebook: dict, argumentos: argparse.Namespace) -> None:
    """Aplica os argumentos somente à cópia do notebook carregada em memória."""
    celula = next(
        (
            item
            for item in notebook["cells"]
            if item.get("id") == "task-6-configuracao"
        ),
        None,
    )
    if celula is None:
        raise RuntimeError("Célula de configuração do pipeline não encontrada.")

    codigo = "".join(celula["source"])
    configuracoes = {
        "RETREINAR_MODELO_5": False,
        "AVALIAR_HISTORICO": not argumentos.sem_avaliacao,
        "INICIO_INFERENCIA_MODELO_7": argumentos.inicio,
        "FIM_INFERENCIA_MODELO_7": argumentos.fim,
        "MAXIMO_PORTAS_RECOMENDADAS_POR_PLATAFORMA": argumentos.max_portas,
        "PASTA_SAIDA": argumentos.pasta_saida,
    }
    for nome, valor in configuracoes.items():
        codigo = substituir_atribuicao(codigo, nome, valor)
    celula["source"] = codigo.splitlines(keepends=True)


def executar_notebook(notebook: dict) -> None:
    """Executa as células de código em um único espaço de nomes."""
    namespace = {"__name__": "__main__"}
    os.chdir(RAIZ_REPOSITORIO)
    for indice, celula in enumerate(notebook["cells"]):
        if celula.get("cell_type") != "code":
            continue
        codigo = "".join(celula.get("source", []))
        exec(
            compile(codigo, f"{CAMINHO_NOTEBOOK.name}:celula-{indice}", "exec"),
            namespace,
        )


def criar_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Executa a inferência sequencial dos Modelos 7 e 5."
    )
    parser.add_argument("--inicio", type=data_iso, default="2025-10-01")
    parser.add_argument("--fim", type=data_iso, default="2026-03-21")
    parser.add_argument(
        "--pasta-saida",
        type=pasta_relativa,
        default="artefatos",
        help="Pasta relativa a models/pipeline-sequencial/.",
    )
    parser.add_argument(
        "--max-portas",
        type=int,
        default=1,
        help="Máximo de portas recomendadas por plataforma e data.",
    )
    parser.add_argument(
        "--sem-avaliacao",
        action="store_true",
        help="Executa a inferência sem calcular métricas históricas.",
    )
    return parser


def main() -> None:
    argumentos = criar_parser().parse_args()
    if argumentos.fim < argumentos.inicio:
        raise SystemExit("A data final não pode ser anterior à data inicial.")
    if argumentos.max_portas < 1:
        raise SystemExit("--max-portas deve ser maior que zero.")

    notebook = json.loads(CAMINHO_NOTEBOOK.read_text(encoding="utf-8"))
    configurar_notebook(notebook, argumentos)
    executar_notebook(notebook)


if __name__ == "__main__":
    main()
