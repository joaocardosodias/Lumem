# Experimento Modelo 5.1

Abra `modelo_5.1.ipynb` e execute as células em ordem, com o diretório de trabalho dentro deste repositório. O experimento usa `painel.parquet` e `data/Corretiva Emergencial/processados/ZM01_limpa.xlsx`.

## Comparação

- Controle: todos os atributos candidatos, incluindo `num_porta`.
- Ablação direta: mesmos atributos, exceto `num_porta`.
- Candidato SHAP: seleção adicional de atributos do modelo sem `num_porta`, usando apenas uma amostra do treino.

Os três modelos usam os mesmos períodos, semente 42 e parâmetros do Random Forest. O limiar é escolhido na validação por F1. A imputação de intervalos sem histórico usa 30 dias nos três modelos, evitando estimar a mediana no teste. Os números históricos do notebook original não são uma comparação controlada com este protocolo.

## Saídas

Arquivos locais em `artefatos/modelo_5_1/`:

- `comparacao_ablacao.csv`: comparação principal, isolando a retirada de `num_porta`.
- `comparacao_modelos.csv`: inclui também o candidato reduzido por SHAP.
- `experimento.json`: períodos, atributos e métricas da execução.
- `modelo_5_1_sem_num_porta.joblib`: modelo da ablação direta.
- `modelo_5_1_sem_num_porta_shap.joblib`: candidato reduzido.

Os artefatos não são publicados nem ativados no serviço. Uma nova execução sobrescreve apenas estas saídas experimentais.

## Limites

A separação é temporal, com sete dias de intervalo. Não há reserva de estações ou portas inéditas. `est_cod`, `via_cod` e históricos individuais continuam presentes: retirar `num_porta` não prova que toda identificação indireta foi eliminada. A avaliação usa uma semente e um recorte temporal; diferenças pequenas não demonstram significância estatística. A avaliação complementar por evento refere-se somente ao candidato SHAP.
