"""Compatibilidade temporária com imports antigos do adaptador PostgreSQL."""

from ml_service.adapters.postgres_feature_repository import PostgresFeatureRepository

_repository = PostgresFeatureRepository()


def obter_data_referencia(estacoes):
    return _repository.obter_data_referencia(estacoes)


def buscar_telemetria_recente(estacoes, hoje, janela_dias=60):
    return _repository.buscar_telemetria_recente(estacoes, hoje, janela_dias)


def buscar_manutencoes(estacoes):
    return _repository.buscar_manutencoes(estacoes)
