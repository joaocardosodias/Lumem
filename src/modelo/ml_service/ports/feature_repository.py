from abc import ABC, abstractmethod
from datetime import date

import pandas as pd


class FeatureRepository(ABC):
    """Porta de leitura dos dados necessários para construir as features."""

    @abstractmethod
    def obter_data_referencia(self, estacoes: list[str]) -> date:
        raise NotImplementedError

    @abstractmethod
    def buscar_telemetria_recente(
        self, estacoes: list[str], hoje: date, janela_dias: int = 60
    ) -> pd.DataFrame:
        raise NotImplementedError

    @abstractmethod
    def buscar_manutencoes(self, estacoes: list[str]) -> pd.DataFrame:
        raise NotImplementedError

    @abstractmethod
    def listar_portas_ativas(self) -> list[dict]:
        raise NotImplementedError
