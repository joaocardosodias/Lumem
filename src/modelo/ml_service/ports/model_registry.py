from abc import ABC, abstractmethod
from pathlib import Path


class ModelRegistry(ABC):
    """Porta de acesso ao artefato ativo, independente de filesystem ou S3."""

    @abstractmethod
    def materialize_active(self) -> tuple[Path, dict]:
        """Retorna o caminho cacheado e o manifest verificado do modelo ativo."""
        raise NotImplementedError

    @abstractmethod
    def materialize_version(self, version: str) -> tuple[Path, dict]:
        """Materializa e verifica uma candidata sem alterar o ponteiro ativo."""
        raise NotImplementedError

    @abstractmethod
    def activate(self, version: str) -> dict:
        """Move o ponteiro ativo para uma versão já publicada."""
        raise NotImplementedError
