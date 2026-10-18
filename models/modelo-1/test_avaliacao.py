import ast
import json
from pathlib import Path
import unittest
from unittest.mock import patch

import numpy as np
import pandas as pd


notebook = json.loads(
    Path(__file__).with_name('03_modelo_1_treinamento.ipynb').read_text(encoding='utf-8')
)
source = next(
    ''.join(cell['source']) for cell in notebook['cells']
    if 'def avaliar_por_episodio(' in ''.join(cell['source'])
)
# Load the actual evaluation functions without training or exporting a model.
definitions = ast.Module(
    body=[node for node in ast.parse(source).body if isinstance(node, ast.FunctionDef)],
    type_ignores=[],
)
namespace = {'pd': pd, 'np': np}
exec(compile(definitions, '<notebook evaluation>', 'exec'), namespace)


class AvaliacaoTest(unittest.TestCase):
    def setUp(self):
        self.base = pd.DataFrame({
            'id_ativo': ['A'] * 5,
            'data_ref': pd.date_range('2026-01-01', periods=5),
            'prob_falha': [0.49, 0.46, 0.46, 0.46, 0.49],
        })
        self.eventos = pd.DataFrame({
            'id_ativo': ['A'], 'data_evento': pd.to_datetime(['2026-01-06'])
        })

    def avaliar(self, cortes):
        return namespace['avaliar_limiares'](self.base, self.eventos, cortes)

    def test_fragmentacao_nao_e_aumento_de_registros(self):
        tabela = self.avaliar([0.48, 0.45])
        self.assertEqual(tabela.registros_alertados.tolist(), [5, 2])
        self.assertEqual(tabela.episodios.tolist(), [1, 2])
        self.assertEqual(tabela.falhas_com_alerta_previo.tolist(), [1, 1])

    def test_igualdade_no_corte_e_sem_alertas(self):
        tabela = self.avaliar([0.49, 0.50, 1.0])
        self.assertEqual(tabela.registros_alertados.tolist(), [2, 0, 0])
        self.assertEqual(tabela.episodios.tolist(), [2, 0, 0])
        self.assertEqual(tabela.recall.tolist(), [1, 0, 0])

    def test_rejeita_probabilidades_invalidas(self):
        for valor in [np.nan, np.inf, -0.01, 1.01]:
            with self.subTest(valor=valor):
                self.base.loc[0, 'prob_falha'] = valor
                with self.assertRaises(ValueError):
                    self.avaliar([0.45])

    def test_rejeita_porta_dia_duplicada(self):
        self.base = pd.concat([self.base, self.base.iloc[[0]]])
        with self.assertRaises(ValueError):
            self.avaliar([0.45])

    def test_rejeita_cortes_invalidos(self):
        for cortes in [[], [np.nan], [np.inf], [-0.1], [1.1]]:
            with self.subTest(cortes=cortes), self.assertRaises(ValueError):
                self.avaliar(cortes)

    def test_bloqueia_contagem_incorreta(self):
        original = namespace['avaliar_por_episodio']

        def contagem_incorreta(*args, **kwargs):
            resultado = original(*args, **kwargs)
            resultado['registros_alertados'] += 1
            return resultado

        with patch.dict(namespace, avaliar_por_episodio=contagem_incorreta):
            with self.assertRaises(AssertionError):
                self.avaliar([0.45, 0.48])


if __name__ == '__main__':
    unittest.main()
